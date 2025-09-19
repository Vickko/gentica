package serper

import (
	"context"
	"fmt"
)

// ScrapeWebpage scrapes a webpage and returns its content in text and markdown formats
func (c *Client) ScrapeWebpage(ctx context.Context, request *WebpageRequest) (*WebpageResponse, error) {
	if request == nil {
		return nil, fmt.Errorf("webpage request cannot be nil")
	}

	if request.URL == "" {
		return nil, fmt.Errorf("URL is required")
	}

	// Set default value for IncludeMarkdown if not explicitly set to false
	if request.IncludeMarkdown == nil {
		includeMarkdown := true
		request.IncludeMarkdown = &includeMarkdown
	}

	var response WebpageResponse

	// Create request with rate limiting
	req, err := c.createRequestWithContext(ctx)
	if err != nil {
		return nil, err
	}

	// Use the scrape endpoint
	scrapeBaseURL := c.scrapeBaseURL
	if scrapeBaseURL == "" {
		scrapeBaseURL = SerperScrapeBaseURL
	}

	req = req.
		SetContext(ctx).
		SetBody(request).
		SetResult(&response)

	resp, err := req.Post(scrapeBaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	// Handle API errors
	if err := handleError(resp); err != nil {
		return nil, err
	}

	return &response, nil
}

// ScrapeWebpageSimple performs a simple webpage scrape with minimal options
func (c *Client) ScrapeWebpageSimple(ctx context.Context, url string) (*WebpageResponse, error) {
	includeMarkdown := true
	return c.ScrapeWebpage(ctx, &WebpageRequest{
		URL:             url,
		IncludeMarkdown: &includeMarkdown,
	})
}

// ScrapeWebpageTextOnly scrapes a webpage and returns only text content
func (c *Client) ScrapeWebpageTextOnly(ctx context.Context, url string) (*WebpageResponse, error) {
	includeMarkdown := false
	return c.ScrapeWebpage(ctx, &WebpageRequest{
		URL:             url,
		IncludeMarkdown: &includeMarkdown,
	})
}