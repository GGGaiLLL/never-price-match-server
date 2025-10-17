package product

import (
	"fmt"
	"net/url"
)

func scrapeEBGamesSearch(searchTerm string) (ScrapeResult, error) {
	if searchTerm == "" {
		return ScrapeResult{}, fmt.Errorf("search term cannot be empty")
	}
	searchURL := fmt.Sprintf("https://www.ebgames.com.au/search?q=%s", url.QueryEscape(searchTerm))
	return scrapeProducts(scrapeProductParams{
		SearchTerm:        searchTerm,
		Platform:          "EB Games",
		SearchURL:         searchURL,
		ContainerSelector: "div.product-tile",
		TitleSelector:     "div.name",
		PriceSelectors:    []string{"span.current-price"},
		ImageSelector:     "img",
		LinkSelector:      "a",
		ImageAttr:         "src",
	})
}
