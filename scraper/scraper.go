package scraper

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/evolvedevlab/weaveset/data"
	"github.com/evolvedevlab/weaveset/store"
)

type Scraper interface {
	Scrape(context.Context) (*data.List, error)
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

	list, err := sc.Scrape(ctx)
	if err != nil {
		return err
	}

	return h.store.Save(list)
}

func detectAndGetScraper(urlStr string) (Scraper, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	switch u.Hostname() {
	case "goodreads.com", "www.goodreads.com":
		return NewGRScraper(urlStr)
	}

	return nil, fmt.Errorf("cannot handle the given URL")
}

func addRequestHeaders(r *http.Request) {
	r.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/122.0.0.0 Safari/537.36")
	r.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	r.Header.Set("Accept-Language", "en-US,en;q=0.9")
	r.Header.Set("Connection", "keep-alive")
}
