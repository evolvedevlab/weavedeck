package queue

import (
	"context"
	"errors"

	"github.com/evolvedevlab/weavedeck/data"
)

var (
	errExceededRetryLimit = errors.New("job has exceeded the set retry limit")
)

type Queuer interface {
	Enqueue(context.Context, *data.Job) error
	Consume(context.Context, data.Handler) error
}

func getRetryKey(jobID string) string {
	return "retries:" + jobID
}
