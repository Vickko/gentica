package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"gentica/clients/firecrawl"
	"gentica/clients/serper"
	"gentica/searchcrawler"
	"sync"
)

// SearchCrawlerParams represents the parameters for the search crawler tool
type SearchCrawlerParams struct {
	Query      string `json:"query"`
	NumResults int    `json:"num_results,omitempty"`
}

// searchCrawlerTool implements the BaseTool interface for SearchCrawler
type searchCrawlerTool struct {
	crawler searchcrawler.SearchCrawler
}

var (
	// Singleton instances for the SearchCrawler and its dependencies
	singletonSearchCrawler   searchcrawler.SearchCrawler
	singletonSerperClient    *serper.Client
	singletonFirecrawlClient *firecrawl.Client
	crawlerOnce              sync.Once
)

const (
	SearchCrawlerToolName        = "searchcrawler"
	searchCrawlerToolDescription = `Search the web and crawl the results to extract content.

WHEN TO USE THIS TOOL:
- Use when you need to search for information on the web and get the actual content
- Helpful for researching topics, finding documentation, or gathering information
- Useful when you need both search results and their full content

HOW TO USE:
- Provide a search query to find relevant web pages
- Optionally specify the number of results to crawl (default: 16)

FEATURES:
- Searches using Google (via Serper API)
- Crawls each search result to extract full content
- Automatic parallel crawling of all results
- Built-in rate limiting (Serper: 100/min, Firecrawl: 10/min)
- Automatic content cleaning and truncation

LIMITATIONS:
- Only crawls public web pages
- Some websites may block automated crawling
- Rate limits apply (Serper: 100/min, Firecrawl: 10/min)
- Content is truncated to ~50KB per page

EXAMPLES:
- Basic search: {"query": "golang concurrency patterns"}
- With options: {"query": "react hooks tutorial", "num_results": 10}

TIPS:
- Use specific search queries for better results
- Adjust num_results based on how much information you need
- The system automatically handles rate limiting`
)

// getSingletonSearchCrawler returns the singleton SearchCrawler instance
func getSingletonSearchCrawler() searchcrawler.SearchCrawler {
	crawlerOnce.Do(func() {
		// Create singleton client instances
		singletonSerperClient = serper.NewClient()
		singletonFirecrawlClient = firecrawl.NewClient()

		// Create SearchCrawler with singleton clients
		singletonSearchCrawler = searchcrawler.NewSearchCrawlerWithClients(
			singletonSerperClient,
			singletonFirecrawlClient,
		)
	})
	return singletonSearchCrawler
}

// NewSearchCrawlerTool creates a new SearchCrawlerTool using the singleton instance
func NewSearchCrawlerTool() BaseTool {
	return &searchCrawlerTool{
		crawler: getSingletonSearchCrawler(),
	}
}

// Name returns the name of the tool
func (t *searchCrawlerTool) Name() string {
	return SearchCrawlerToolName
}

// Info returns the tool information
func (t *searchCrawlerTool) Info() ToolInfo {
	return ToolInfo{
		Name:        SearchCrawlerToolName,
		Description: searchCrawlerToolDescription,
		Parameters: map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "The search query to find relevant web pages",
			},
			"num_results": map[string]any{
				"type":        "number",
				"description": "Number of search results to crawl (default: 16, max: 50)",
			},
		},
		Required: []string{"query"},
	}
}

// Run executes the search and crawl operation
func (t *searchCrawlerTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params SearchCrawlerParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse("Failed to parse searchcrawler parameters: " + err.Error()), nil
	}

	// Validate parameters
	if params.Query == "" {
		return NewTextErrorResponse("Query parameter is required and cannot be empty"), nil
	}

	// Apply reasonable limits
	if params.NumResults > 50 {
		params.NumResults = 50
	}

	// Create request
	request := &searchcrawler.SearchCrawlRequest{
		Query:      params.Query,
		NumResults: params.NumResults,
	}

	// Execute search and crawl
	response, err := t.crawler.SearchAndCrawl(ctx, request)
	if err != nil {
		return NewTextErrorResponse(fmt.Sprintf("Search and crawl failed: %v", err)), nil
	}

	// Format response as JSON
	responseJSON, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return NewTextErrorResponse("Failed to format response: " + err.Error()), nil
	}

	// Create metadata for the response
	metadata := map[string]any{
		"total_search_results": response.TotalSearchResults,
		"total_crawled":        response.TotalCrawled,
		"errors_count":         len(response.Errors),
	}

	// Return successful response with metadata
	return WithResponseMetadata(
		NewTextResponse(string(responseJSON)),
		metadata,
	), nil
}
