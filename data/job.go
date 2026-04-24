package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/evolvedevlab/weaveset/config"
	"github.com/redis/go-redis/v9"
)

var (
	errExceededRetryLimit = errors.New("job has exceeded the set retry limit")
)

// Handler is a generic interface for handling a job.
// Handler can be anything, in our case its web scraper.
type Handler interface {
	Handle(context.Context, *Job) error
}

type Queuer interface {
	Enqueue(context.Context, *Job) error
	Consume(context.Context, Handler) error
}

type Job struct {
	ID        string
	URL       string
	CreatedAt time.Time
}

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

func (q *RedisQueue) Consume(ctx context.Context, handler Handler) error {
	// loop to re-claim stale jobs (mostly due to failures)
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

			slog.Error("Consume error", "hostname", q.hostname, "err", err)
			continue
		}

		for _, stream := range streams {
			for _, msg := range stream.Messages {
				if err := q.handleMessage(ctx, msg, handler); err != nil {
					if !errors.Is(err, errExceededRetryLimit) {
						slog.Error("Consume error", "hostname", q.hostname, "msg", msg, "err", err)
					}
				}
			}
		}
	}
}

func (q *RedisQueue) Enqueue(ctx context.Context, job *Job) error {
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

func (q *RedisQueue) reaperLoop(ctx context.Context, handler Handler) error {
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

			slog.Error("Reaper loop error", "hostname", q.hostname, "err", err)
			break
		}

		if len(messages) == 0 {
			start = "0" // reset for next cycle
			break
		}
		start = next

		for _, msg := range messages {
			if err := q.handleMessage(ctx, msg, handler); err != nil {
				slog.Error("Reaper loop error", "hostname", q.hostname, "err", err)
			}
		}
	}

	return nil
}

func (q *RedisQueue) handleMessage(ctx context.Context, msg redis.XMessage, handler Handler) error {
	retries, err := q.getRetryCount(ctx, msg.ID)
	if err != nil {
		return err
	}

	// ACK if too many retries
	if retries >= config.MaxJobRetryLimit {
		err := q.client.XAck(ctx, q.stream, q.group, msg.ID).Err()
		return err
	}

	job, err := q.readMessageData(msg)

	err = handler.Handle(ctx, job)
	if err != nil {
		// increment retry count
		if err := q.incrRetryCount(ctx, msg.ID); err != nil {
			return err
		}
		return errExceededRetryLimit // leave as is
	}

	err = q.client.XAck(ctx, q.stream, q.group, msg.ID).Err()
	if err != nil {
		return err
	}

	// delete retries count
	err = q.client.Del(ctx, "retries:"+msg.ID).Err()
	return err
}

func (q *RedisQueue) incrRetryCount(ctx context.Context, msgID string) error {
	key := "retries:" + msgID

	pipe := q.client.TxPipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, time.Hour*4)

	_, err := pipe.Exec(ctx)
	return err
}

func (q *RedisQueue) getRetryCount(ctx context.Context, msgID string) (int64, error) {
	n, err := q.client.Get(ctx, "retries:"+msgID).Int64()
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		return 0, err
	}
	return n, nil
}

func (q *RedisQueue) readMessageData(msg redis.XMessage) (*Job, error) {
	raw, ok := msg.Values["data"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid payload data")
	}

	var job Job
	if err := json.Unmarshal([]byte(raw), &job); err != nil {
		return nil, err
	}

	return &job, nil
}
