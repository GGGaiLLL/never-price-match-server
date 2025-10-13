package product

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"
	"net/url"
	"never-price-match-server/internal/infra/logger"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/gocolly/colly/v2"
)

// Service defines the business logic interface for products.
// It now correctly uses the ScrapeResult type.
type Service interface {
	GetProductsByCategory(category string) ([]Product, error)
	SearchAndScrape(productName string) ([]ScrapeResult, error)
}

type service struct {
	repo Repo
}

// NewService creates a new product service instance.
func NewService(repo Repo) Service {
	return &service{repo: repo}
}

// GetProductsByCategory remains unchanged as it deals with database entities.
func (s *service) GetProductsByCategory(category string) ([]Product, error) {
	// ... (existing logic is correct and remains unchanged)
	// 1. Retrieve basic product information from the repository
	products, err := s.repo.GetProductsByCategory(category)
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup

	// 2. For each product, scrape the latest prices from different platforms concurrently
	for i := range products {
		// Simulate product main image if it's missing
		if products[i].ImageURL == "" {
			products[i].ImageURL = fmt.Sprintf("https://example.com/images/%s.jpg", products[i].Name)
		}

		wg.Add(len(products[i].Prices))

		for j := range products[i].Prices {
			// Use a goroutine for each scraping task
			go func(productIndex, priceIndex int) {
				defer wg.Done()

				priceInfo := &products[productIndex].Prices[priceIndex]
				var scrapedPrice float64
				var err error

				// 3. Call the specific scraper for each platform
				switch priceInfo.Platform {
				case "Amazon.com": // Changed from JD.com
					scrapedPrice, err = scrapeAmazon(priceInfo.Link) // Changed from scrapeJD
				case "eBay", "Walmart": // Changed from Taobao, Pinduoduo
					// You would have scrapeEbay, scrapeWalmart functions here
					// For now, we'll keep them as random values for demonstration
					scrapedPrice = 1000 + rand.Float64()*(500)
				default:
					// Fallback or error
					scrapedPrice = -1 // Indicate an error or unsupported platform
				}

				if err != nil {
					logger.L.Warn("Failed to scrape price",
						logger.Str("platform", priceInfo.Platform),
						logger.Str("url", priceInfo.Link),
						logger.Err(err))
				} else {
					priceInfo.Price = scrapedPrice
				}

			}(i, j)
		}
	}

	wg.Wait() // Wait for all scraping goroutines to finish

	// You can add logic here to find the lowest price among the scraped results
	// For now, we just return the updated products.
	return products, nil
}

// scrapeAmazon remains a helper function and is unchanged.
func scrapeAmazon(url string) (float64, error) {
	var price float64
	var found bool

	c := colly.NewCollector(
		// Using a realistic user agent is often necessary.
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
	)

	// This CSS selector is an EXAMPLE. You MUST inspect the actual product page on Amazon.com to find the correct one.
	// Amazon prices are often split into parts, e.g., '.a-price-whole' and '.a-price-fraction'.
	// A common selector for the full price (often visually hidden) is '.a-price .a-offscreen'.
	c.OnHTML(".a-price .a-offscreen", func(e *colly.HTMLElement) {
		// The price might be in a format like "$1,299.00". We need to parse the number.
		re := regexp.MustCompile(`[0-9\\.,]+`) // Updated regex to handle commas
		priceStr := re.FindString(e.Text)
		priceStr = regexp.MustCompile(`,`).ReplaceAllString(priceStr, "") // Remove commas for parsing

		p, err := strconv.ParseFloat(priceStr, 64)
		if err == nil {
			// Only take the first price found, as Amazon pages can have multiple price elements.
			if !found {
				price = p
				found = true
			}
		}
	})

	c.OnRequest(func(r *colly.Request) {
		logger.L.Info("Scraping URL", logger.Str("url", r.URL.String()))
	})

	c.OnError(func(r *colly.Response, err error) {
		logger.L.Error("Scraping request failed", logger.Int("status_code", r.StatusCode), logger.Err(err))
	})

	err := c.Visit(url)
	if err != nil {
		return 0, err
	}

	if !found {
		return 0, fmt.Errorf("could not find price on page %s with the given selector", url)
	}

	return price, nil
}

// SearchAndScrape is now fully updated to use ScrapeResult.
func (s *service) SearchAndScrape(productName string) ([]ScrapeResult, error) {
	collyPlatforms := map[string]func(string) (ScrapeResult, error){
		// "Amazon AU": scrapeAmazonSearch,
		// "eBay AU":   scrapeEbaySearch,
	}
	chromedpPlatforms := map[string]func(string) (ScrapeResult, error){
		"Big W": scrapeBigWSearch,
		// "JB Hi-Fi": scrapeJBHIFISearch,
		// "Kogan":    scrapeKoganSearch,
	}

	var wg sync.WaitGroup
	resultsChan := make(chan ScrapeResult, len(collyPlatforms)+len(chromedpPlatforms))
	errChan := make(chan error, len(collyPlatforms)+len(chromedpPlatforms))

	// Run Colly scrapers
	for platform, scraperFunc := range collyPlatforms {
		wg.Add(1)
		go func(pf string, sf func(string) (ScrapeResult, error)) {
			defer wg.Done()
			result, err := sf(productName)
			if err != nil {
				errChan <- fmt.Errorf("failed to scrape %s: %w", pf, err)
				return
			}
			result.Platform = pf
			resultsChan <- result
		}(platform, scraperFunc)
	}

	// Run ChromeDP scrapers
	for platform, scraperFunc := range chromedpPlatforms {
		wg.Add(1)
		go func(pf string, sf func(string) (ScrapeResult, error)) {
			defer wg.Done()
			result, err := sf(productName)
			if err != nil {
				errChan <- fmt.Errorf("failed to scrape %s: %w", pf, err)
				return
			}
			result.Platform = pf
			resultsChan <- result
		}(platform, scraperFunc)
	}

	wg.Wait()
	close(resultsChan)
	close(errChan)

	var finalResults []ScrapeResult
	for result := range resultsChan {
		finalResults = append(finalResults, result)
	}

	for err := range errChan {
		logger.L.Warn("Scraping error", logger.Err(err))
	}

	return finalResults, nil
}

// --- Helper Functions ---

func createSearchURL(baseURL, queryParam, productName string) string {
	return fmt.Sprintf("%s?%s=%s", baseURL, queryParam, url.QueryEscape(productName))
}

// parsePrice is a new name for the old cleanPrice function.
func parsePrice(priceStr string) (float64, error) {
	re := regexp.MustCompile(`[0-9,]+(\\.[0-9]+)?`)
	priceMatch := re.FindString(priceStr)
	if priceMatch == "" {
		return 0, fmt.Errorf("no price-like string found in '%s'", priceStr)
	}
	priceCleaned := strings.ReplaceAll(priceMatch, ",", "")
	return strconv.ParseFloat(priceCleaned, 64)
}

// --- Colly Scraper Implementations (for simpler sites) ---
// Note: These are placeholders and need to be updated to return ScrapeResult if used.
// func scrapeAmazonSearch(productName string) (ScrapeResult, error) { ... }
// func scrapeEbaySearch(productName string) (ScrapeResult, error) { ... }

// --- ChromeDP Scraper Implementations (for complex sites) ---

// scrapeWithChromeDP is now fully updated to return ScrapeResult and use ScrapedProduct.
func scrapeWithChromeDP(searchURL, itemSelector, nameSelector, priceSelector, imageSelector, linkSelector, imageAttr string) (ScrapeResult, error) {
	var result ScrapeResult

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-features", "Translate"),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36"),
	)
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	var nodes []*cdp.Node
	err := chromedp.Run(ctx,
		chromedp.Navigate(searchURL),
		chromedp.WaitVisible(itemSelector, chromedp.ByQuery),
		chromedp.Sleep(3*time.Second),
		chromedp.Nodes(itemSelector, &nodes, chromedp.ByQueryAll),
	)
	if err != nil {
		return result, fmt.Errorf("chromedp failed to get product nodes from %s: %w", searchURL, err)
	}

	if len(nodes) == 0 {
		// It's not an error if no products are found, just return an empty result.
		log.Printf("Info: no products found on page %s with selector %s", searchURL, itemSelector)
		return result, nil
	}

	for _, node := range nodes {
		var name, price, img, link string
		err := chromedp.Run(ctx,
			chromedp.Text(nameSelector, &name, chromedp.ByQuery, chromedp.FromNode(node)),
			chromedp.Text(priceSelector, &price, chromedp.ByQuery, chromedp.FromNode(node)),
			chromedp.AttributeValue(imageSelector, imageAttr, &img, nil, chromedp.ByQuery, chromedp.FromNode(node)),
			chromedp.AttributeValue(linkSelector, "href", &link, nil, chromedp.ByQuery, chromedp.FromNode(node)),
		)
		if err != nil {
			log.Printf("Warning: could not extract details for a product, skipping: %v", err)
			continue
		}

		parsedPrice, _ := parsePrice(price)
		absoluteLink := link
		if !strings.HasPrefix(link, "http") {
			baseURL, _ := url.Parse(searchURL)
			relativeURL, _ := url.Parse(link)
			if baseURL != nil && relativeURL != nil {
				absoluteLink = baseURL.ResolveReference(relativeURL).String()
			}
		}

		product := ScrapedProduct{
			Name:     strings.TrimSpace(name),
			Price:    parsedPrice,
			ImageURL: strings.TrimSpace(img),
			Link:     absoluteLink,
		}

		if product.Name != "" && product.Price > 0 {
			result.Products = append(result.Products, product)
		}
	}

	return result, nil
}

// scrapeBigWSearch is now fully updated to use and return ScrapeResult.
func scrapeBigWSearch(searchTerm string) (ScrapeResult, error) {
	if searchTerm == "" {
		return ScrapeResult{}, fmt.Errorf("search term cannot be empty")
	}

	exactSearchTerm := fmt.Sprintf("\"%s\"", searchTerm)
	searchURL := fmt.Sprintf("https://www.bigw.com.au/search?text=%s", url.QueryEscape(exactSearchTerm))

	scrapedData, err := scrapeWithChromeDP(
		searchURL,
		"article",
		`[data-optly-product-tile-name="true"]`,
		`[data-testid="price-value"]`,
		"img",
		"a",
		"src",
	)
	if err != nil {
		return ScrapeResult{}, err
	}

	var filteredProducts []ScrapedProduct
	lowerSearchTerm := strings.ToLower(searchTerm)

	for _, product := range scrapedData.Products {
		productNameLower := strings.ToLower(product.Name)

		// This is the new, stricter filtering logic you requested.
		// 1. The product name must contain the search term.
		// 2. The product name's length can be at most 20 characters longer than the search term.
		if strings.HasPrefix(productNameLower, lowerSearchTerm) && len(productNameLower) <= (len(lowerSearchTerm)+20) {
			filteredProducts = append(filteredProducts, product)
		}
	}

	return ScrapeResult{Products: filteredProducts, Platform: "Big W"}, nil
}

// Scraper defines the interface for different scraping strategies.
// It is now correctly using the ScrapeResult type.
type Scraper interface {
	SearchAndScrape(productName string) ([]ScrapeResult, error)
}
