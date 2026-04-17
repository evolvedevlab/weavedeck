package scraper

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/evolvedevlab/weaveset/data"
)

type grScraper struct {
	URL *url.URL
}

func NewGRScraper(urlStr string) (Scraper, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	if !isValidGoodreadsURL(u) {
		return nil, fmt.Errorf("invalid goodreads list URL")
	}

	return &grScraper{
		URL: u,
	}, nil
}

func (sc *grScraper) Scrape(ctx context.Context) (*data.List, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sc.URL.String(), nil)
	if err != nil {
		return nil, err
	}
	addRequestHeaders(req)

	client := new(http.Client)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var list data.List
	doc.Find(".mainContent .leftContainer").Each(func(i int, s *goquery.Selection) {
		titleParts := strings.Split(s.Find(".pageHeader h2").Text(), ">")
		if len(titleParts) > 0 {
			list.Name = strings.TrimSpace(titleParts[len(titleParts)-1])
		}

		s.Find("#all_votes table tbody").Each(func(i int, s *goquery.Selection) {
			list.Items = sc.collectRows(s)
		})
	})

	return &list, nil
}

func (sc grScraper) collectRows(s *goquery.Selection) []data.Item {
	var items []data.Item
	s.Find("tr").Each(func(i int, s *goquery.Selection) {
		item := data.Item{
			Position: i + 1,
		}

		sc.collectItem(s, &item)
		items = append(items, item)
	})
	return items
}

func (sc grScraper) collectItem(s *goquery.Selection, item *data.Item) {
	s.Find("td").Each(func(i int, s *goquery.Selection) {
		d := s.Find(`div[data-resource-type="Book"]`)
		if id, ok := d.Attr("data-resource-id"); ok {
			item.ID = strings.TrimSpace(id)
		}

		title := strings.TrimSpace(s.Find(".bookTitle").Text())
		if len(title) > 0 {
			item.Title = title
		}

		coverURL, exists := s.Find(".bookCover").Attr("src")
		coverURL = strings.TrimSpace(coverURL)
		if exists {
			if _, err := url.Parse(coverURL); err == nil {
				item.Images = append(item.Images, coverURL)
			}
		}

		authorName := strings.TrimSpace(s.Find(".authorName").Text())
		if len(authorName) > 0 {
			item.By = append(item.By, authorName)
		}

		miniRatingParts := strings.Split(s.Find(".minirating").Text(), "—")
		if len(miniRatingParts) >= 2 {
			avgParts := strings.Split(strings.TrimSpace(miniRatingParts[0]), " ")
			totalParts := strings.Split(strings.TrimSpace(miniRatingParts[1]), " ")

			for _, nStr := range avgParts {
				n, err := strconv.ParseFloat(nStr, 64)
				if err == nil {
					item.AvgRating = n
					break
				}
			}
			for _, nStr := range totalParts {
				n, err := strconv.ParseInt(strings.ReplaceAll(nStr, ",", ""), 10, 64)
				if err == nil {
					item.TotalRatings = n
					break
				}
			}
		}

	})
}

func isValidGoodreadsURL(url *url.URL) bool {
	if !(url.Hostname() == "goodreads.com" || url.Hostname() == "www.goodreads.com") {
		return false
	}
	if !strings.HasPrefix(url.Path, "/list/show/") {
		return false
	}
	return true
}
