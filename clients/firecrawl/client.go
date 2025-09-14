package firecrawl

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
	"golang.org/x/time/rate"
)

// Global configuration variables (temporarily hardcoded)
var (
	FirecrawlAPIKey = "fc-5c481f4a021b4c979fe68080072618a8"
	FirecrawlAPIURL = "https://api.firecrawl.dev/v2"
	
	// Rate limiting configuration - easily removable by setting EnableRateLimit to false
	EnableRateLimit    = true  // Set to false to disable rate limiting
	RateLimitPerMinute = 10    // Maximum 10 requests per minute
)

// Client represents a Firecrawl API client
type Client struct {
	apiKey      string
	baseURL     string
	httpClient  *resty.Client
	rateLimiter *rate.Limiter // Rate limiter, nil when disabled
}


// NewClient creates a new Firecrawl client with default configuration
func NewClient() *Client {
	return NewClientWithConfig(FirecrawlAPIKey, FirecrawlAPIURL)
}

// NewClientWithConfig creates a new Firecrawl client with custom configuration
func NewClientWithConfig(apiKey, baseURL string) *Client {
	client := resty.New()
	client.SetTimeout(2 * time.Minute)
	client.SetHeader("Content-Type", "application/json")

	// Create rate limiter if enabled
	var limiter *rate.Limiter
	if EnableRateLimit {
		// Calculate rate: 10 requests per minute = 10/60 requests per second
		// Using a burst of 1 to ensure strict rate limiting
		limiter = rate.NewLimiter(rate.Every(time.Minute/time.Duration(RateLimitPerMinute)), 1)
	}

	return &Client{
		apiKey:      apiKey,
		baseURL:     baseURL,
		httpClient:  client,
		rateLimiter: limiter,
	}
}

// SetAPIKey updates the API key
func (c *Client) SetAPIKey(apiKey string) {
	c.apiKey = apiKey
}

// SetBaseURL updates the base URL
func (c *Client) SetBaseURL(baseURL string) {
	c.baseURL = baseURL
}

// SetTimeout sets the HTTP client timeout
func (c *Client) SetTimeout(timeout time.Duration) {
	c.httpClient.SetTimeout(timeout)
}

// createRequest creates a new resty request with authentication
// It also applies rate limiting if enabled
func (c *Client) createRequest() *resty.Request {
	return c.httpClient.R().
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
}

// createRequestWithContext creates a new request and applies rate limiting
// This is used internally by Scrape methods to ensure rate limiting
func (c *Client) createRequestWithContext(ctx context.Context) (*resty.Request, error) {
	// Apply rate limiting if enabled
	if c.rateLimiter != nil {
		// Wait will block until rate limit allows the request
		// This appears as network I/O delay to the caller
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limit wait cancelled: %w", err)
		}
	}
	
	return c.createRequest(), nil
}

// handleError processes error responses from the API
func handleError(resp *resty.Response) error {
	if resp.StatusCode() == 402 {
		return fmt.Errorf("payment required: %s", resp.String())
	}
	if resp.StatusCode() == 429 {
		return fmt.Errorf("rate limit exceeded: %s", resp.String())
	}
	if resp.StatusCode() >= 500 {
		var errorResp ScrapeResponse
		if err := json.Unmarshal(resp.Body(), &errorResp); err == nil && errorResp.Error != "" {
			return fmt.Errorf("server error: %s (code: %s)", errorResp.Error, errorResp.Code)
		}
		return fmt.Errorf("server error: %s", resp.String())
	}
	if resp.StatusCode() >= 400 {
		return fmt.Errorf("client error (status %d): %s", resp.StatusCode(), resp.String())
	}
	return nil
}