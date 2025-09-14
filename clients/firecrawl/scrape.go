package firecrawl

import (
	"context"
	"fmt"
)

// Scrape performs a web scraping operation on the specified URL
func (c *Client) Scrape(ctx context.Context, options *ScrapeOptions) (*ScrapeResponse, error) {
	if options == nil {
		return nil, fmt.Errorf("scrape options cannot be nil")
	}

	if options.URL == "" {
		return nil, fmt.Errorf("URL is required")
	}

	// Set default values if not provided
	if options.OnlyMainContent == nil {
		defaultVal := true
		options.OnlyMainContent = &defaultVal
	}

	if options.MaxAge == nil {
		defaultVal := 172800000 // 2 days in milliseconds
		options.MaxAge = &defaultVal
	}

	if options.WaitFor == nil {
		defaultVal := 0
		options.WaitFor = &defaultVal
	}

	if options.Mobile == nil {
		defaultVal := false
		options.Mobile = &defaultVal
	}

	if options.SkipTLSVerification == nil {
		defaultVal := true
		options.SkipTLSVerification = &defaultVal
	}

	if options.RemoveBase64Images == nil {
		defaultVal := true
		options.RemoveBase64Images = &defaultVal
	}

	if options.BlockAds == nil {
		defaultVal := true
		options.BlockAds = &defaultVal
	}

	if options.Proxy == nil {
		defaultProxy := string(ProxyAuto)
		options.Proxy = &defaultProxy
	}

	if options.StoreInCache == nil {
		defaultVal := true
		options.StoreInCache = &defaultVal
	}

	if options.ZeroDataRetention == nil {
		defaultVal := false
		options.ZeroDataRetention = &defaultVal
	}

	// Set default formats if not provided
	if len(options.Formats) == 0 {
		options.Formats = []Format{
			SimpleFormat{Type: FormatMarkdown},
		}
	}

	// Set default parsers if not provided
	if len(options.Parsers) == 0 {
		options.Parsers = []Parser{
			SimpleParser(ParserPDF),
		}
	}

	// Set default location if not provided
	if options.Location == nil {
		options.Location = &Location{
			Country: "US",
		}
	}

	var response ScrapeResponse
	
	// Create request with rate limiting
	req, err := c.createRequestWithContext(ctx)
	if err != nil {
		return nil, err
	}
	
	req = req.
		SetContext(ctx).
		SetBody(options).
		SetResult(&response)

	resp, err := req.Post(fmt.Sprintf("%s/scrape", c.baseURL))
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	// Handle API errors
	if err := handleError(resp); err != nil {
		return nil, err
	}
	
	// Resty automatically unmarshals JSON response into the struct we provided with SetResult
	// The response variable should now contain the unmarshaled data

	// Check if the response indicates failure
	if !response.Success {
		if response.Error != "" {
			return nil, fmt.Errorf("scrape failed: %s", response.Error)
		}
		return nil, fmt.Errorf("scrape failed with unknown error")
	}

	return &response, nil
}

// ScrapeSimple performs a simple scrape with minimal options
func (c *Client) ScrapeSimple(ctx context.Context, url string) (*ScrapeResponse, error) {
	return c.Scrape(ctx, &ScrapeOptions{
		URL: url,
	})
}

// ScrapeWithFormats performs a scrape with specific output formats
func (c *Client) ScrapeWithFormats(ctx context.Context, url string, formats []Format) (*ScrapeResponse, error) {
	return c.Scrape(ctx, &ScrapeOptions{
		URL:     url,
		Formats: formats,
	})
}

// ScrapeWithActions performs a scrape with page actions
func (c *Client) ScrapeWithActions(ctx context.Context, url string, actions []Action) (*ScrapeResponse, error) {
	return c.Scrape(ctx, &ScrapeOptions{
		URL:     url,
		Actions: actions,
	})
}