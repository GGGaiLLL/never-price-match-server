package product

import (
	"fmt"
	"net/url"
)

func scrapeBCFSearch(searchTerm string) (ScrapeResult, error) {
	if searchTerm == "" {
		return ScrapeResult{}, fmt.Errorf("search term cannot be empty")
	}
	searchURL := fmt.Sprintf("https://www.bcf.com.au/search?q=%s", url.QueryEscape(searchTerm))
	return scrapeProducts(scrapeProductParams{
		SearchTerm:        searchTerm,
		Platform:          "BCF",
		SearchURL:         searchURL,
		ContainerSelector: `li.grid-tile`,
		TitleSelector:     `div.product-name`,
		PriceSelectors: []string{
			`span.product-sales-price`,
		},
		ImageSelector: `div.product-image img`,
		ImageAttr:     "src",
		LinkSelector:  "a",
	})
}

// Scraper defines the interface for different scraping strategies.
type Scraper interface {
	SearchAndScrape(productName string) ([]ScrapeResult, error)
}
