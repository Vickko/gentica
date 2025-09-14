package searchcrawler

import "context"

// SearchCrawler is the interface for search and crawl operations
type SearchCrawler interface {
	// SearchAndCrawl searches for the query and then crawls each result
	SearchAndCrawl(ctx context.Context, request *SearchCrawlRequest) (*SearchCrawlResponse, error)
}

// SearchCrawlRequest represents the request for search and crawl operation
type SearchCrawlRequest struct {
	// Query is the search query
	Query string `json:"query"`
	
	// NumResults is the number of search results to fetch and crawl
	// Default is 16 if not specified
	NumResults int `json:"num_results,omitempty"`
}

// SearchCrawlResponse represents the response from search and crawl operation
type SearchCrawlResponse struct {
	// Query is the original search query
	Query string `json:"query"`
	
	// Results contains the crawled content for each search result
	Results []CrawledResult `json:"results"`
	
	// TotalSearchResults is the total number of search results found
	TotalSearchResults int `json:"total_search_results"`
	
	// TotalCrawled is the number of successfully crawled results
	TotalCrawled int `json:"total_crawled"`
	
	// Errors contains any errors that occurred during the process
	Errors []string `json:"errors,omitempty"`
}

// CrawledResult represents a single crawled search result
type CrawledResult struct {
	// Title is the title of the page
	Title string `json:"title"`
	
	// URL is the URL of the page
	URL string `json:"url"`
	
	// Content is the main text content of the page
	Content string `json:"content"`
	
	// Metadata contains essential metadata about the page
	Metadata *PageMetadata `json:"metadata,omitempty"`
	
	// Error contains any error that occurred while crawling this specific URL
	Error string `json:"error,omitempty"`
}

// PageMetadata contains essential metadata about a crawled page
type PageMetadata struct {
	// Description is the meta description of the page
	Description string `json:"description,omitempty"`
	
	// Language is the language of the page content
	Language string `json:"language,omitempty"`
	
	// PublishedDate is when the content was published (if available)
	PublishedDate string `json:"published_date,omitempty"`
	
	// Author is the author of the content (if available)
	Author string `json:"author,omitempty"`
}