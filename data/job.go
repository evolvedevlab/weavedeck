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
