package data

import (
	"context"
	"time"
)

// Handler can be anything, in our case its web scraper.
type Handler interface {
	Handle(context.Context, *Job) error
}

type Job struct {
	ID        string
	URL       string
	CreatedAt time.Time
}

type NonRetryableError struct {
	Err error
}

func (e NonRetryableError) Error() string {
	return e.Err.Error()
}

func NonRetry(err error) error {
	return NonRetryableError{Err: err}
}
