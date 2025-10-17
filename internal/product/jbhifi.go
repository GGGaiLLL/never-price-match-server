package product

import (
	"fmt"
	"net/url"
)

func scrapeJBHIFISearch(searchTerm string) (ScrapeResult, error) {
	if searchTerm == "" {
		return ScrapeResult{}, fmt.Errorf("search term cannot be empty")
	}
	searchURL := fmt.Sprintf("https://www.jbhifi.com.au/search?query=%s", url.QueryEscape(searchTerm))

	return scrapeProducts(scrapeProductParams{
		SearchTerm:        searchTerm,
		Platform:          "JB Hi-Fi",
		SearchURL:         searchURL,
		ContainerSelector: "div.ProductCard",
		TitleSelector:     `[data-testid="product-card-title"]`,
		PriceSelectors:    []string{`[data-testid="ticket-price"]`},
		ImageSelector:     "img",
		LinkSelector:      "a",
		ImageAttr:         "src",
	})
}
