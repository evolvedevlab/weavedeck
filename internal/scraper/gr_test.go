package scraper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGRScraper_URLValidation(t *testing.T) {
	a := assert.New(t)
	t.Parallel()

	sc := NewGRScraper()

	_, err := sc.Scrape(t.Context(), "https://www.evolveasdev.com/list/show/43")
	a.Error(err)

	_, err = sc.Scrape(t.Context(), "https://www.goodreads.com/list/")
	a.Error(err)

	_, err = sc.Scrape(t.Context(), "https://www.goodreads.com/list/show/")
	a.Error(err)
}

func TestGRScraper_e2e(t *testing.T) {
	a := assert.New(t)

	sc := NewGRScraper()

	list, err := sc.Scrape(t.Context(), "https://www.goodreads.com/list/show/399714")
	a.NoError(err)

	a.NotNil(list)
	a.NotEmpty(list.Name)
	a.NotEmpty(list.Items)
}
