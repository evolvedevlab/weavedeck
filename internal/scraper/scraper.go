package scraper

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/evolvedevlab/weaveset/data"
	"github.com/evolvedevlab/weaveset/internal/store"
)

type Scraper interface {
	Scrape(context.Context, string) (*data.List, error)
}

// Handler implements the data.Handler interface.
// This handler is a responsible for scraping and storage.
type Handler struct {
	store store.Storer
}

func NewHandler(store store.Storer) data.Handler {
	return &Handler{
		store: store,
	}
}

func (h *Handler) Handle(ctx context.Context, job *data.Job) error {
	sc, err := detectAndGetScraper(job.URL)
	if err != nil {
		return err
	}
	sc = NewLoggingScraper(sc)

	list, err := sc.Scrape(ctx, job.URL)
	if err != nil {
		return err
	}

	return h.store.Save(list)
}

type loggingScraper struct {
	next Scraper
}

func NewLoggingScraper(sc Scraper) Scraper {
	return &loggingScraper{next: sc}
}

func (s *loggingScraper) Scrape(ctx context.Context, url string) (*data.List, error) {
	start := time.Now()
	slog.Info("scrape_start", "url", url, "start", start)

	list, err := s.next.Scrape(ctx, url)

	took := time.Since(start)
	if err != nil {
		slog.Error("scrape_failed", "url", url, "took_ms", took.Milliseconds(), "err", err)
		return nil, err
	}

	slog.Info("scrape_success", "url", url, "took_ms", took.Milliseconds())
	return list, nil
}

func detectAndGetScraper(urlStr string) (Scraper, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	switch u.Hostname() {
	case "goodreads.com", "www.goodreads.com":
		return NewGRScraper(), nil
	}

	return nil, fmt.Errorf("cannot handle the given URL")
}

func addRequestHeaders(r *http.Request) {
	r.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/122.0.0.0 Safari/537.36")
	r.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	r.Header.Set("Accept-Language", "en-US,en;q=0.9")
	r.Header.Set("Connection", "keep-alive")
}
