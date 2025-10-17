package product

import (
	"fmt"
	"net/url"
)

func scrapeBigWSearch(searchTerm string) (ScrapeResult, error) {
	if searchTerm == "" {
		return ScrapeResult{}, fmt.Errorf("search term cannot be empty")
	}
	searchURL := fmt.Sprintf("https://www.bigw.com.au/search?text=%s", url.QueryEscape(searchTerm))
	return scrapeProducts(scrapeProductParams{
		SearchTerm:        searchTerm,
		Platform:          "Big W",
		SearchURL:         searchURL,
		ContainerSelector: "article",
		TitleSelector:     `[data-optly-product-tile-name="true"]`,
		PriceSelectors:    []string{`[data-testid="price-value"]`},
		ImageSelector:     "img",
		LinkSelector:      "a",
		ImageAttr:         "src",
	})
}
