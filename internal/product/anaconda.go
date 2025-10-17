package product

import (
	"fmt"
	"net/url"
)

func scrapeAnacondaSearch(searchTerm string) (ScrapeResult, error) {
	if searchTerm == "" {
		return ScrapeResult{}, fmt.Errorf("search term cannot be empty")
	}
	searchURL := fmt.Sprintf("https://www.anacondastores.com/search?text=%s", url.QueryEscape(searchTerm))
	return scrapeProducts(scrapeProductParams{
		SearchTerm:        searchTerm,
		Platform:          "Anaconda",
		SearchURL:         searchURL,
		ContainerSelector: `div.card-element-wrapper`,
		TitleSelector:     `[itemprop="name"]`,
		PriceSelectors: []string{
			`p.price-vip span.amount`,
			`p.price-regular span.amount`,
			`p.price-standard span.amount`,
		},
		ImageSelector: `img.productdetailimg`,
		ImageAttr:     "src",
		LinkSelector:  "a",
	})
}
