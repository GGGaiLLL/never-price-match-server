package product

import (
	"fmt"
	"net/url"
)

func scrapeAmazonSearch(searchTerm string) (ScrapeResult, error) {
	if searchTerm == "" {
		return ScrapeResult{}, fmt.Errorf("search term cannot be empty")
	}
	searchURL := fmt.Sprintf("https://www.amazon.com.au/s?k=%s", url.QueryEscape(searchTerm))
	return scrapeProducts(scrapeProductParams{
		SearchTerm:        searchTerm,
		Platform:          "Amazon AU",
		SearchURL:         searchURL,
		ContainerSelector: `[role="listitem"]`,
		TitleSelector:     `h2.a-size-base-plus span`,
		PriceSelectors:    []string{`span.a-price span.a-offscreen`},
		ImageSelector:     "img.s-image",
		LinkSelector:      "a",
		ImageAttr:         "src",
	})
}
