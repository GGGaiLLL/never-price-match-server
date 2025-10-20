package product

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"never-price-match-server/internal/infra/logger" // <--- 1. 添加 "os" 包
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/page" // NEW: Import the 'page' package for stealth operations
	"github.com/chromedp/chromedp"
)

// Service defines the business logic interface for products.
// It now correctly uses the ScrapeResult type.
type Service interface {
	SearchAndScrape(productName string, category string) ([]ScrapeResult, error)
	GetProductSuggestions(name string) ([]string, error)
}

type service struct {
	repo Repo
}

// NewService creates a new product service instance.
func NewService(repo Repo) Service {
	return &service{repo: repo}
}

// SearchAndScrape is now fully updated to use ScrapeResult.
func (s *service) SearchAndScrape(productName string, category string) ([]ScrapeResult, error) {
	// 1. First, try to find the product in the database.
	cachedProducts, err := s.repo.SearchProductsByName(productName)
	if err != nil {
		// Log the error but don't block. We can still proceed with scraping.
		logger.L.Warn("Failed to search for cached products", logger.Err(err))
	}

	// If we found cached products, format them into the ScrapeResult structure and return.
	if len(cachedProducts) > 0 {
		return formatProductsToScrapeResults(cachedProducts), nil
	}

	// 2. If not found in the database, proceed with live scraping.
	scrapedResults, err := s.performScraping(productName, category)
	if err != nil {
		return nil, err // If scraping itself fails catastrophically, return the error.
	}

	// 3. Asynchronously save the new results to the database for future searches.
	if len(scrapedResults) > 0 {
		go func() {
			productsToSave := convertScrapeResultsToProducts(scrapedResults)
			if err := s.repo.SaveProducts(productsToSave); err != nil {
				logger.L.Error("Failed to save scraped products to database", logger.Err(err))
			}
		}()
	}

	return scrapedResults, nil
}

// formatProductsToScrapeResults converts a flat list of DB product entities
// into the grouped ScrapeResult format required by the API.
func formatProductsToScrapeResults(products []Product) []ScrapeResult {
	groupedByPlatform := make(map[string][]ScrapedProduct)

	for _, p := range products {
		sp := ScrapedProduct{
			Name:     p.Name,
			Price:    p.Price,
			ImageURL: p.ImageURL,
			Link:     p.Link,
		}
		groupedByPlatform[p.Platform] = append(groupedByPlatform[p.Platform], sp)
	}

	var results []ScrapeResult
	for platform, prods := range groupedByPlatform {
		results = append(results, ScrapeResult{
			Platform: platform,
			Products: prods,
		})
	}
	return results
}

// convertScrapeResultsToProducts flattens the grouped ScrapeResult structure
// into a flat list of Product entities suitable for saving to the database.
func convertScrapeResultsToProducts(results []ScrapeResult) []Product {
	var products []Product
	for _, res := range results {
		for _, p := range res.Products {
			products = append(products, Product{
				Name:     p.Name,
				Platform: res.Platform,
				Price:    p.Price,
				Link:     p.Link,
				ImageURL: p.ImageURL,
			})
		}
	}
	return products
}

// scraperFunc defines a standard signature for all scraper functions.
// This makes them interchangeable.
type scraperFunc func(productName string) (ScrapeResult, error)

// allScrapers acts as a central registry for all available scraping functions.
// To add a new scraper, simply add it to this map.
var allScrapers = map[string]scraperFunc{
	"Amazon AU": scrapeAmazonSearch,
	"Anaconda":  scrapeAnacondaSearch,
	"BCF":       scrapeBCFSearch,
	"Big W":     scrapeBigWSearch,
	"JB Hi-Fi":  scrapeJBHIFISearch,
	"EB Games":  scrapeEBGamesSearch,
}

// categoryPlatforms maps product categories to the platforms that should be scraped for them.
// This is the central configuration for category-based scraping.
// To add a new category or change which platforms are scraped, just edit this map.
var categoryPlatforms = map[string][]string{
	"outdoors": {
		"Anaconda",
		"BCF",
		"Amazon AU",
	},
	"electronics": { // Example of another specific category
		"EB Games",
		"JB Hi-Fi",
		"Amazon AU",
	},
	"default": { // Fallback for any category not explicitly defined
		"Big W",
		"JB Hi-Fi",
		"EB Games",
		"Amazon AU",
	},
}

func (s *service) performScraping(productName string, category string) ([]ScrapeResult, error) {
	// Look up the list of platform names for the given category.
	platformNames, ok := categoryPlatforms[category]
	if !ok {
		// If the category is not found in our map, use the 'default' list as a fallback.
		logger.L.Info("Category not found, using default platforms", logger.Str("category", category))
		platformNames = categoryPlatforms["default"]
	}

	resultsChan := make(chan ScrapeResult, len(platformNames))
	errChan := make(chan error, len(platformNames))

	// Run scrapers sequentially for the selected platforms to avoid being blocked.
	for _, platformName := range platformNames {
		scraper, exists := allScrapers[platformName]
		if !exists {
			logger.L.Warn("Scraper not defined for platform", logger.Str("platform", platformName))
			continue
		}

		// Execute the scraper function.
		result, err := scraper(productName)
		if err != nil {
			errChan <- fmt.Errorf("failed to scrape %s: %w", platformName, err)
			continue
		}
		result.Platform = platformName
		resultsChan <- result
	}

	close(resultsChan)
	close(errChan)

	var finalResults []ScrapeResult
	for result := range resultsChan {
		finalResults = append(finalResults, result)
	}

	// Log any errors that occurred during scraping.
	for err := range errChan {
		logger.L.Warn("Scraping error", logger.Err(err))
	}

	return finalResults, nil
}

func scrapeWithChromeDP(params scrapeProductParams) (ScrapeResult, error) {
	var result ScrapeResult
	searchURL := params.SearchURL
	itemSelector := params.ContainerSelector
	nameSelector := params.TitleSelector
	priceSelectors := params.PriceSelectors
	imageSelector := params.ImageSelector
	linkSelector := params.LinkSelector
	imageAttr := params.ImageAttr

	// A helper function to create a non-fatal "click if exists" action.
	// It waits for the selector to be visible and then clicks, ignoring any errors.
	tryClick := func(selector string) chromedp.Action {
		return chromedp.ActionFunc(func(ctx context.Context) error {
			ctx, cancel := context.WithTimeout(ctx, 2*time.Second) // Short timeout for each attempt.
			defer cancel()
			// We run this and ignore the error. If the element is not found or the click fails,
			// we don't want to stop the entire scraping process.
			_ = chromedp.Run(ctx,
				chromedp.WaitVisible(selector, chromedp.BySearch),
				chromedp.Click(selector, chromedp.BySearch, chromedp.NodeVisible),
			)
			return nil // Always return nil to indicate this action is optional and non-critical.
		})
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
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

	// This is the main context for the browser tab.
	taskCtx, cancelTask := chromedp.NewContext(allocCtx)
	defer cancelTask()

	// --- ANTI-BOT DETECTION ---
	// This script runs on every new document loaded in the browser.
	// It deletes the `navigator.webdriver` property, which is a primary flag
	// used by websites to detect automated browsers like chromedp.
	// Hiding this flag makes our scraper appear more like a regular user,
	// bypassing "Your browser is not supported" errors.
	if err := chromedp.Run(taskCtx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			script := "Object.defineProperty(navigator, 'webdriver', {get: () => undefined})"
			_, err := page.AddScriptToEvaluateOnNewDocument(script).Do(ctx)
			if err != nil {
				return fmt.Errorf("could not add stealth script: %w", err)
			}
			return nil
		}),
	); err != nil {
		return ScrapeResult{}, err // If we can't set up stealth, we shouldn't proceed.
	}

	// 1. Create a context specifically for loading the page.
	loadCtx, cancelLoad := context.WithTimeout(taskCtx, 45*time.Second)
	defer cancelLoad()

	var nodes []*cdp.Node
	// Use the loading context for the initial page load and node retrieval.
	err := chromedp.Run(loadCtx,
		chromedp.Navigate(searchURL),
		chromedp.Sleep(2*time.Second),

		// --- Enhanced Cookie Banner Handling ---
		// Try to click a series of common cookie consent buttons.
		tryClick(`//button[contains(translate(., 'ABCDEFGHIJKLMNOPQRSTUVWXYZ', 'abcdefghijklmnopqrstuvwxyz'), 'accept all')]`),
		tryClick(`//button[contains(translate(., 'ABCDEFGHIJKLMNOPQRSTUVWXYZ', 'abcdefghijklmnopqrstuvwxyz'), 'allow all')]`),
		tryClick(`//button[contains(translate(., 'ABCDEFGHIJKLMNOPQRSTUVWXYZ', 'abcdefghijklmnopqrstuvwxyz'), 'accept')]`),
		tryClick(`//button[contains(translate(., 'ABCDEFGHIJKLMNOPQRSTUVWXYZ', 'abcdefghijklmnopqrstuvwxyz'), 'agree')]`),
		tryClick(`//button[contains(translate(., 'ABCDEFGHIJKLMNOPQRSTUVWXYZ', 'abcdefghijklmnopqrstuvwxyz'), 'got it')]`),
		tryClick(`//button[contains(translate(., 'ABCDEFGHIJKLMNOPQRSTUVWXYZ', 'abcdefghijklmnopqrstuvwxyz'), 'ok')]`),
		tryClick(`//button[contains(translate(., 'ABCDEFGHIJKLMNOPQRSTUVWXYZ', 'abcdefghijklmnopqrstuvwxyz'), 'continue')]`),

		tryClick(`#onetrust-accept-btn-handler`), // Specific selector for OneTrust
		tryClick(`[id*="cookie-accept"]`),
		tryClick(`[id*="consent-accept"]`),
		tryClick(`.cookie-notify-closeBtn`),

		chromedp.Sleep(2*time.Second), // Wait a moment for the banner to disappear after a potential click.

		chromedp.WaitVisible(itemSelector, chromedp.ByQuery),
		chromedp.Sleep(3*time.Second),
		chromedp.Nodes(itemSelector, &nodes, chromedp.ByQueryAll),
	)
	if err != nil {
		return result, fmt.Errorf("chromedp failed to get product nodes from %s: %w", searchURL, err)
	}

	if len(nodes) == 0 {
		// It's not an error if no products are found, just return an empty result.
		return result, nil
	}

	// Now iterate, using a separate, short-lived context for each extraction.
	for i, node := range nodes {
		var name, price, img, link string
		var err error

		// Derive a new context from the main task context with a 10-second timeout.
		extractCtx, cancelExtract := context.WithTimeout(taskCtx, 10*time.Second)

		// --- Extraction with Detailed Logging ---

		// Extract name
		err = chromedp.Run(extractCtx, chromedp.Text(nameSelector, &name, chromedp.ByQuery, chromedp.FromNode(node)))
		if err != nil {
			log.Printf("[Product %d] Failed to extract name with selector '%s': %v", i+1, nameSelector, err)
			cancelExtract()
			continue
		}

		// Extract image
		err = chromedp.Run(extractCtx, chromedp.AttributeValue(imageSelector, imageAttr, &img, nil, chromedp.ByQuery, chromedp.FromNode(node)))
		if err != nil {
			log.Printf("[Product %d] Failed to extract image with selector '%s' (attr '%s'): %v", i+1, imageSelector, imageAttr, err)
			cancelExtract()
			continue
		}

		// Extract link
		err = chromedp.Run(extractCtx, chromedp.AttributeValue(linkSelector, "href", &link, nil, chromedp.ByQuery, chromedp.FromNode(node)))
		if err != nil {
			log.Printf("[Product %d] Failed to extract link with selector '%s': %v", i+1, linkSelector, err)
			cancelExtract()
			continue
		}

		// --- DYNAMIC PRICE EXTRACTION ---
		var priceFound bool

		if params.Platform == "Anaconda" {
			// --- METHOD 1: Use JavaScript to pierce Shadow DOM ---
			selectorsJSON, _ := json.Marshal(priceSelectors)

			jsGetPriceInShadowScript := fmt.Sprintf(`(function(selectors){
  			const findInShadow = (root, selector, depth=0) => {
    		if (!root || depth>4) return null;
    		const el = root.querySelector?.(selector);
				if (el) return el;
				const nodes = root.querySelectorAll ? root.querySelectorAll('*') : [];
				for (const host of nodes) {
					if (host.shadowRoot) {
						const found = findInShadow(host.shadowRoot, selector, depth+1);
						if (found) return found;
					}
				}
    		return null;
  			};
				for (const sel of selectors) {
					const el = findInShadow(document, sel);
					if (el) {
						const t=(el.innerText||el.textContent||'').trim();
						if(t) return t;
					}
				}
				return '';
			})(%s)`, selectorsJSON)

			// Re-assign to the loop's err variable to correctly handle logging
			err = chromedp.Run(extractCtx,
				chromedp.EvaluateAsDevTools(jsGetPriceInShadowScript, &price),
			)

			if err != nil {
				log.Printf("[Product %d] Failed to execute Shadow DOM price script: %v", i+1, err)
			} else if strings.TrimSpace(price) != "" {
				priceFound = true
			}

		} else {
			// --- METHOD 2: Standard WaitVisible logic ---
			for _, selector := range priceSelectors {
				var priceText string
				waitCtx, cancelWait := context.WithTimeout(extractCtx, 3*time.Second)
				if err := chromedp.Run(waitCtx, chromedp.WaitVisible(selector, chromedp.ByQuery, chromedp.FromNode(node))); err == nil {
					if err := chromedp.Run(extractCtx, chromedp.Text(selector, &priceText, chromedp.ByQuery, chromedp.FromNode(node))); err == nil && strings.TrimSpace(priceText) != "" {
						price = priceText
						priceFound = true
						cancelWait()
						break
					}
				}
				cancelWait()
			}
		}

		if !priceFound {
			log.Printf("[Product %d] Failed to extract price with any method. Selectors: %v", i+1, priceSelectors)
			cancelExtract()
			continue
		}
		// --- END OF DYNAMIC PRICE EXTRACTION ---

		cancelExtract() // Release context resources for this iteration.

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

func scrapeProducts(input scrapeProductParams) (ScrapeResult, error) {
	scrapedData, err := scrapeWithChromeDP(input)
	if err != nil {
		return ScrapeResult{}, fmt.Errorf("failed to scrape %s: %w", input.Platform, err)
	}

	var filteredProducts []ScrapedProduct
	lowerSearchTerm := strings.ToLower(input.SearchTerm)
	searchWords := strings.Fields(lowerSearchTerm)

	// If there are no search words, there's nothing to filter by. Return empty.
	if len(searchWords) == 0 {
		return ScrapeResult{Products: []ScrapedProduct{}, Platform: input.Platform}, nil
	}

	for _, product := range scrapedData.Products {

		reNonAlnum := regexp.MustCompile(`[^a-z0-9]`)
		searchable := reNonAlnum.ReplaceAllString(strings.ToLower(product.Name), "")

		matchedWordsCount := 0
		for _, w := range searchWords {
			if strings.Contains(searchable, reNonAlnum.ReplaceAllString(w, "")) {
				matchedWordsCount++
			}
		}

		// Calculate the ratio of matched words.
		matchRatio := float64(matchedWordsCount) / float64(len(searchWords))

		// If 80% or more of the search words are found, consider it a match.
		// This handles cases where brand names like "Apple" are omitted in the product title.
		if matchRatio >= 0.8 {
			filteredProducts = append(filteredProducts, product)
		}
	}

	// Crucially, return the filtered products, not the original full list.
	return ScrapeResult{Products: filteredProducts, Platform: input.Platform}, nil
}

// GetProductSuggestions returns a list of product names for search-as-you-type suggestions.
// It only queries the database if the search term is 2 or more characters long.
func (s *service) GetProductSuggestions(name string) ([]string, error) {
	// To avoid excessive queries, only search if the input is non-trivial.
	if len(name) < 2 {
		return []string{}, nil // Return empty slice if not enough characters
	}
	return s.repo.GetProductNamesByName(name)
}
