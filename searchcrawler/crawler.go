package searchcrawler

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"gentica/clients/firecrawl"
	"gentica/clients/serper"
)

// defaultSearchCrawler is the concrete implementation of SearchCrawler
type defaultSearchCrawler struct {
	serperClient    *serper.Client
	firecrawlClient *firecrawl.Client
	mu              sync.RWMutex
}

// NewSearchCrawler creates a new SearchCrawler instance with default clients
func NewSearchCrawler() SearchCrawler {
	return &defaultSearchCrawler{
		serperClient:    serper.NewClient(),
		firecrawlClient: firecrawl.NewClient(),
	}
}

// NewSearchCrawlerWithClients creates a new SearchCrawler instance with provided client instances
func NewSearchCrawlerWithClients(serperClient *serper.Client, firecrawlClient *firecrawl.Client) SearchCrawler {
	return &defaultSearchCrawler{
		serperClient:    serperClient,
		firecrawlClient: firecrawlClient,
	}
}

// SearchAndCrawl performs search and then crawls each result concurrently
func (sc *defaultSearchCrawler) SearchAndCrawl(ctx context.Context, request *SearchCrawlRequest) (*SearchCrawlResponse, error) {
	// Validate request
	if request == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	if request.Query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	// Set defaults
	if request.NumResults <= 0 {
		request.NumResults = 16
	}

	// Perform search
	searchResp, err := sc.performSearch(ctx, request.Query, request.NumResults)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Prepare response
	response := &SearchCrawlResponse{
		Query:              request.Query,
		Results:            []CrawledResult{},
		TotalSearchResults: len(searchResp.Organic),
		Errors:             []string{},
	}

	// Extract URLs from search results
	urls := sc.extractURLs(searchResp)
	if len(urls) == 0 {
		return response, nil
	}

	// Limit URLs to requested number
	if len(urls) > request.NumResults {
		urls = urls[:request.NumResults]
	}

	// Crawl URLs concurrently - all URLs are crawled in parallel
	// Rate limiting is handled by the Firecrawl client
	crawledResults := sc.crawlURLsConcurrently(ctx, urls, searchResp)

	// Process results
	for _, result := range crawledResults {
		if result.Error != "" {
			response.Errors = append(response.Errors, fmt.Sprintf("%s: %s", result.URL, result.Error))
		} else {
			response.TotalCrawled++
		}
		response.Results = append(response.Results, result)
	}

	return response, nil
}

// performSearch executes the search using Serper client
func (sc *defaultSearchCrawler) performSearch(ctx context.Context, query string, numResults int) (*serper.SearchResponse, error) {
	searchReq := &serper.SearchRequest{
		Query: query,
		Type:  serper.SearchTypeSearch,
		Num:   numResults,
		GL:    "us",
		HL:    "en",
	}

	return sc.serperClient.Search(ctx, searchReq)
}

// extractURLs extracts URLs from search results
func (sc *defaultSearchCrawler) extractURLs(searchResp *serper.SearchResponse) []string {
	var urls []string

	if searchResp == nil {
		return urls
	}

	// Extract URLs from organic results
	for _, result := range searchResp.Organic {
		if result.Link != "" {
			urls = append(urls, result.Link)
		}
	}

	return urls
}

// crawlURLsConcurrently crawls multiple URLs concurrently
// All URLs are crawled in parallel, with rate limiting handled by the Firecrawl client
func (sc *defaultSearchCrawler) crawlURLsConcurrently(ctx context.Context, urls []string, searchResp *serper.SearchResponse) []CrawledResult {
	results := make([]CrawledResult, len(urls))

	// Create wait group for synchronization
	var wg sync.WaitGroup

	// Create a map to match URLs with their titles from search results
	titleMap := make(map[string]string)
	for _, result := range searchResp.Organic {
		if result.Link != "" && result.Title != "" {
			titleMap[result.Link] = result.Title
		}
	}

	// Crawl each URL in parallel
	// The Firecrawl client's rate limiter will automatically throttle requests
	for i, url := range urls {
		wg.Add(1)
		go func(index int, targetURL string) {
			defer wg.Done()

			// Check if context is cancelled
			select {
			case <-ctx.Done():
				results[index] = CrawledResult{
					URL:   targetURL,
					Title: titleMap[targetURL],
					Error: "context cancelled",
				}
				return
			default:
			}

			// Crawl the URL
			crawledResult := sc.crawlSingleURL(ctx, targetURL, titleMap[targetURL])
			results[index] = crawledResult
		}(i, url)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	return results
}

// crawlSingleURL crawls a single URL using Firecrawl client
func (sc *defaultSearchCrawler) crawlSingleURL(ctx context.Context, url string, title string) CrawledResult {
	result := CrawledResult{
		URL:   url,
		Title: title,
	}

	// Prepare scrape options
	scrapeOptions := &firecrawl.ScrapeOptions{
		URL: url,
		Formats: []firecrawl.Format{
			firecrawl.SimpleFormat{Type: firecrawl.FormatMarkdown},
		},
	}

	// Set reasonable defaults for scraping
	onlyMainContent := true
	scrapeOptions.OnlyMainContent = &onlyMainContent

	skipTLS := true
	scrapeOptions.SkipTLSVerification = &skipTLS

	removeBase64 := true
	scrapeOptions.RemoveBase64Images = &removeBase64

	blockAds := true
	scrapeOptions.BlockAds = &blockAds

	// Perform the scrape
	scrapeResp, err := sc.firecrawlClient.Scrape(ctx, scrapeOptions)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	// Extract content from response
	if scrapeResp != nil && scrapeResp.Data != nil {
		// Get main content
		result.Content = cleanContent(scrapeResp.Data.Markdown)

		// Extract metadata
		if scrapeResp.Data.Metadata != nil {
			result.Metadata = &PageMetadata{
				Description: scrapeResp.Data.Metadata.Description,
			}

			// Extract language if available
			if scrapeResp.Data.Metadata.Language != nil {
				result.Metadata.Language = *scrapeResp.Data.Metadata.Language
			}

			// Use title from metadata if not already set
			if result.Title == "" && scrapeResp.Data.Metadata.Title != "" {
				result.Title = scrapeResp.Data.Metadata.Title
			}
		}
	}

	return result
}

// cleanContent cleans and truncates content to a reasonable size
func cleanContent(content string) string {
	// Remove excessive whitespace
	content = strings.TrimSpace(content)

	// Replace multiple newlines with double newline
	lines := strings.Split(content, "\n")
	var cleanedLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			cleanedLines = append(cleanedLines, trimmed)
		}
	}
	content = strings.Join(cleanedLines, "\n\n")

	// Limit content length to prevent huge responses
	const maxContentLength = 50000 // ~50KB of text
	if len(content) > maxContentLength {
		content = content[:maxContentLength] + "...[truncated]"
	}

	return content
}
