package product

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"net/url"
	"never-price-match-server/internal/infra/logger" // <--- 1. 添加 "os" 包
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/page" // NEW: Import the 'page' package for stealth operations
	"github.com/chromedp/chromedp"
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
					// scrapedPrice, err = scrapeAmazon(priceInfo.Link) // Changed from scrapeJD
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

// SearchAndScrape is now fully updated to use ScrapeResult.
func (s *service) SearchAndScrape(productName string) ([]ScrapeResult, error) {
	collyPlatforms := map[string]func(string) (ScrapeResult, error){
		// "eBay AU":   scrapeEbaySearch,
	}
	chromedpPlatforms := map[string]func(string) (ScrapeResult, error){
		"Big W":     scrapeBigWSearch,
		"JB Hi-Fi":  scrapeJBHIFISearch,
		"EB Games":  scrapeEBGamesSearch,
		"Amazon AU": scrapeAmazonSearch,
		"Anaconda":  scrapeAnacondaSearch,
		"BCF":       scrapeBCFSearch,
	}

	var wg sync.WaitGroup
	resultsChan := make(chan ScrapeResult, len(collyPlatforms)+len(chromedpPlatforms))
	errChan := make(chan error, len(collyPlatforms)+len(chromedpPlatforms))

	// Run Colly scrapers in parallel
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

	// Run ChromeDP scrapers sequentially to avoid being blocked and to conserve resources.
	for platform, scraperFunc := range chromedpPlatforms {
		result, err := scraperFunc(productName)
		if err != nil {
			errChan <- fmt.Errorf("failed to scrape %s: %w", platform, err)
			continue
		}
		result.Platform = platform
		resultsChan <- result
	}

	wg.Wait() // Wait for the parallel Colly scrapers to finish.
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
		chromedp.Sleep(2*time.Second), // Wait a moment for cookie banners to appear.

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
