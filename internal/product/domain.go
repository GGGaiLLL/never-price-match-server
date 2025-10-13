package product

// ScrapedProduct defines the structure for a single scraped product.
// This is the "information card" for each item we find.
type ScrapedProduct struct {
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	ImageURL string  `json:"image_url"`
	Link     string  `json:"link"`
}

// ScrapeResult holds all the results from a single scraping platform.
// It includes the platform's name and a list of products found there.
type ScrapeResult struct {
	Platform string           `json:"platform"`
	Products []ScrapedProduct `json:"products"`
}
