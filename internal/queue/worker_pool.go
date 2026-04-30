package queue

import (
	"context"
	"errors"
	"log/slog"
	"math/rand"
	"time"

	"github.com/evolvedevlab/weavedeck/config"
	"github.com/evolvedevlab/weavedeck/data"
)

const (
	maxInFlight = 1000
	backoffBase = time.Second * 2
)

type wMessage struct {
	*data.Job
	retries  int
	inFlight bool
	last     time.Time
}

func newWMessage(job *data.Job) *wMessage {
	return &wMessage{
		Job: job,
	}
}

type result struct {
	messageID string
	err       error
}

// WARN: data map is not concurrent safe.
// Should only be touched in schedular loop
type WorkerPool struct {
	n int // number of workers

	data      map[string]*wMessage
	enqueuech chan *data.Job
	workch    chan *wMessage
	resultch  chan *result
}

func NewWorkerPool(concurrency, buffer int) Queuer {
	if concurrency == 0 {
		concurrency = 10
	}
	if buffer == 0 {
		buffer = 100
	}
	return &WorkerPool{
		n:         concurrency,
		data:      make(map[string]*wMessage),
		enqueuech: make(chan *data.Job, buffer),
		resultch:  make(chan *result, concurrency*2),
		workch:    make(chan *wMessage, concurrency),
	}
}

func (wp *WorkerPool) Consume(ctx context.Context, handler data.Handler) error {
	for i := 0; i < wp.n; i++ {
		go wp.worker(ctx, handler)
	}
	go wp.scheduler(ctx)

	<-ctx.Done()
	return ctx.Err()
}

func (wp *WorkerPool) Enqueue(ctx context.Context, job *data.Job) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case wp.enqueuech <- job:
		return nil
	}
}

func (wp *WorkerPool) scheduler(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case job := <-wp.enqueuech:
			if len(wp.data) >= maxInFlight {
				slog.Warn("queue_full", "job_id", job.ID)
				continue
			}

			msg := newWMessage(job)
			msg.inFlight = true
			wp.data[msg.ID] = msg

			wp.workch <- msg
		case res := <-wp.resultch:
			msg, ok := wp.data[res.messageID]
			if !ok {
				continue
			}

			if res.err != nil {
				// if its an non-retryable error, drop it entirely
				var nrErr data.NonRetryableError
				if errors.As(res.err, &nrErr) {
					wp.dropMessage(msg)
					continue
				}

				// drop if too many retries
				if msg.retries >= config.MaxJobRetryLimit {
					slog.Error("job_processing_dropped", "msg_id", res.messageID, "err", res.err)
					wp.dropMessage(msg)
					continue
				}

				// schedule for retrial
				wp.doRetry(msg)
			} else {
				// success
				wp.dropMessage(msg)
			}
		case <-ticker.C:
			now := time.Now()
			for _, msg := range wp.data {
				// if idle and retry limit hasn't reached
				if !msg.inFlight && now.Sub(msg.last) >= backoff(msg.retries) {
					msg.inFlight = true
					wp.workch <- msg
				}
			}
		}
	}
}

func (wp *WorkerPool) worker(ctx context.Context, handler data.Handler) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-wp.workch:
			err := handler.Handle(ctx, msg.Job)
			wp.resultch <- &result{
				messageID: msg.ID,
				err:       err,
			}
		}
	}
}

func (wp *WorkerPool) doRetry(msg *wMessage) {
	msg.retries++
	msg.last = time.Now()
	msg.inFlight = false
}

func (wp *WorkerPool) dropMessage(msg *wMessage) {
	delete(wp.data, msg.ID)
}

func backoff(retries int) time.Duration {
	const max = 1 * time.Minute
	if retries > 6 {
		return max
	}

	d := backoffBase * time.Duration(1<<retries)

	// jitter: ±25%
	jitter := d / 4
	d = d - jitter + time.Duration(rand.Int63n(int64(2*jitter)))
	if d > max {
		return max
	}
	return d
}
