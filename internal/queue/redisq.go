package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/evolvedevlab/weaveset/config"
	"github.com/evolvedevlab/weaveset/data"
	"github.com/redis/go-redis/v9"
)

// RedisQueue is an implementation of Queuer interface.
// hostname is required and has to be unique across workers.
type RedisQueue struct {
	hostname string
	stream   string
	group    string
	client   *redis.Client
}

func NewRedisQueue(hostname, stream, group string, client *redis.Client) Queuer {
	return &RedisQueue{
		hostname: hostname,
		stream:   stream,
		group:    group,
		client:   client,
	}
}

func (q *RedisQueue) Consume(ctx context.Context, handler data.Handler) error {
	go q.reaperLoop(ctx, handler)
	for {
		streams, err := q.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    q.group,
			Consumer: q.hostname,
			Streams:  []string{q.stream, ">"},
			Count:    10,
			Block:    0, // block forever
		}).Result()
		if err != nil {
			if err := ctx.Err(); err != nil {
				return err
			}

			slog.Error("consume_error", "op", "XReadGroup", "hostname", q.hostname, "err", err)
			continue
		}

		for _, stream := range streams {
			for _, msg := range stream.Messages {
				if err := q.handleMessage(ctx, msg, handler); err != nil {
					slog.Error("job_processing_failed",
						"op", "consume_handleMessage",
						"hostname", q.hostname,
						"msg_id", msg.ID,
						"err", err)
				}
			}
		}
	}
}

func (q *RedisQueue) Enqueue(ctx context.Context, job *data.Job) error {
	payload, err := json.Marshal(job)
	if err != nil {
		return err
	}

	err = q.client.XAdd(ctx, &redis.XAddArgs{
		Stream: q.stream,
		MaxLen: 100000, // keep last 100k messages
		Approx: true,
		Values: map[string]any{
			"data": payload,
		},
	}).Err()
	return err
}

// loop to re-claim stale jobs (mostly due to failures)
func (q *RedisQueue) reaperLoop(ctx context.Context, handler data.Handler) error {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	start := "0"
	for range ticker.C {
		messages, next, err := q.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{
			Stream:   q.stream,
			Group:    q.group,
			Consumer: q.hostname,
			MinIdle:  time.Minute,
			Start:    start,
			Count:    10,
		}).Result()
		if err != nil {
			if err := ctx.Err(); err != nil {
				return err
			}

			slog.Error("reaper_error", "op", "XAutoClaim", "hostname", q.hostname, "err", err)
			continue
		}

		if len(messages) == 0 {
			start = "0" // reset for next cycle
			continue
		}
		start = next

		for _, msg := range messages {
			if err := q.handleMessage(ctx, msg, handler); err != nil {
				if !errors.Is(err, errExceededRetryLimit) {
					slog.Error("job_processing_failed",
						"op", "reaper_handleMessage",
						"hostname", q.hostname,
						"msg_id", msg.ID,
						"err", err)
				}
			}
		}
	}

	return nil
}

func (q *RedisQueue) handleMessage(ctx context.Context, msg redis.XMessage, handler data.Handler) error {
	job, err := q.readMessagePayload(msg)
	if err != nil {
		return err
	}

	retries, err := q.getRetryCount(ctx, job.ID)
	if err != nil {
		return err
	}

	if retries >= config.MaxJobRetryLimit {
		return q.dropMessage(ctx, msg.ID)
	}

	return q.processMessage(ctx, msg, job, handler)
}

func (q *RedisQueue) processMessage(ctx context.Context, msg redis.XMessage, job *data.Job, handler data.Handler) error {
	start := time.Now()
	// handle job
	err := handler.Handle(ctx, job)
	// metrics
	jobDuration.Observe(time.Since(start).Seconds())

	if err != nil {
		// metrics
		jobsProcessedTotal.WithLabelValues("fail").Inc()

		// increment retry count
		if err := q.incrRetryCount(ctx, job.ID); err != nil {
			return err
		}
		// metrics
		jobRetriesTotal.Inc()

		return err // leave as is
	}

	// metrics
	jobsProcessedTotal.WithLabelValues("success").Inc()

	if err := q.client.XAck(ctx, q.stream, q.group, msg.ID).Err(); err != nil {
		return err
	}

	// delete retries count
	err = q.client.Del(ctx, getRetryKey(job.ID)).Err()
	return err
}

func (q *RedisQueue) dropMessage(ctx context.Context, msgID string) error {
	// metrics
	jobsDroppedTotal.Inc()

	if err := q.client.XAck(ctx, q.stream, q.group, msgID).Err(); err != nil {
		return err
	}
	return errExceededRetryLimit
}

func (q *RedisQueue) incrRetryCount(ctx context.Context, jobID string) error {
	key := getRetryKey(jobID)

	pipe := q.client.TxPipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, time.Hour)

	_, err := pipe.Exec(ctx)
	return err
}

func (q *RedisQueue) getRetryCount(ctx context.Context, jobID string) (int64, error) {
	n, err := q.client.Get(ctx, getRetryKey(jobID)).Int64()
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		return 0, err
	}
	return n, nil
}

func (q *RedisQueue) readMessagePayload(msg redis.XMessage) (*data.Job, error) {
	raw, ok := msg.Values["data"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid payload data")
	}

	var job data.Job
	if err := json.Unmarshal([]byte(raw), &job); err != nil {
		return nil, err
	}

	return &job, nil
}
