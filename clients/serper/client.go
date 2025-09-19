package serper

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/go-resty/resty/v2"
	"golang.org/x/time/rate"
)

// Global configuration variables
var (
	// API configuration with environment variable support
	SerperAPIKey        string
	SerperBaseURL       string
	SerperScrapeBaseURL string

	// Rate limiting configuration - easily removable by setting EnableRateLimit to false
	EnableRateLimit    = false // Set to true to enable rate limiting
	RateLimitPerMinute = 100   // Maximum 100 requests per minute (Serper's default limit)
)

func init() {
	// Initialize from environment variables with fallback defaults
	SerperAPIKey = os.Getenv("SERPER_API_KEY")
	if SerperAPIKey == "" {
		SerperAPIKey = "62adbfaba56f71225b562cb4704ccc7b28e73a6d"
	}

	SerperBaseURL = os.Getenv("SERPER_BASE_URL")
	if SerperBaseURL == "" {
		SerperBaseURL = "https://google.serper.dev"
	}

	SerperScrapeBaseURL = os.Getenv("SERPER_SCRAPE_BASE_URL")
	if SerperScrapeBaseURL == "" {
		SerperScrapeBaseURL = "https://scrape.serper.dev"
	}
}

// Client represents a Serper API client
type Client struct {
	apiKey        string
	baseURL       string
	scrapeBaseURL string
	httpClient    *resty.Client
	rateLimiter   *rate.Limiter // Rate limiter, nil when disabled
}

// NewClient creates a new Serper client with default configuration
func NewClient() *Client {
	return NewClientWithConfig(SerperAPIKey, SerperBaseURL, SerperScrapeBaseURL)
}

// NewClientWithConfig creates a new Serper client with custom configuration
func NewClientWithConfig(apiKey, baseURL, scrapeBaseURL string) *Client {
	client := resty.New()
	client.SetTimeout(30 * time.Second)
	client.SetHeader("Content-Type", "application/json")

	// Create rate limiter if enabled
	var limiter *rate.Limiter
	if EnableRateLimit {
		// Calculate rate: 100 requests per minute = 100/60 requests per second
		// Using a burst of 1 to ensure strict rate limiting
		limiter = rate.NewLimiter(rate.Every(time.Minute/time.Duration(RateLimitPerMinute)), 1)
	}

	return &Client{
		apiKey:        apiKey,
		baseURL:       baseURL,
		scrapeBaseURL: scrapeBaseURL,
		httpClient:    client,
		rateLimiter:   limiter,
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

// SetScrapeBaseURL updates the scrape base URL
func (c *Client) SetScrapeBaseURL(scrapeBaseURL string) {
	c.scrapeBaseURL = scrapeBaseURL
}

// SetTimeout sets the HTTP client timeout
func (c *Client) SetTimeout(timeout time.Duration) {
	c.httpClient.SetTimeout(timeout)
}

// createRequest creates a new resty request with authentication
func (c *Client) createRequest() *resty.Request {
	return c.httpClient.R().
		SetHeader("X-API-KEY", c.apiKey)
}

// createRequestWithContext creates a new request and applies rate limiting
// This is used internally by Search methods to ensure rate limiting
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
	if resp.StatusCode() == 401 {
		return fmt.Errorf("unauthorized: invalid API key")
	}
	if resp.StatusCode() == 402 {
		return fmt.Errorf("payment required: insufficient credits")
	}
	if resp.StatusCode() == 403 {
		return fmt.Errorf("forbidden: %s", resp.String())
	}
	if resp.StatusCode() == 429 {
		return fmt.Errorf("rate limit exceeded: %s", resp.String())
	}
	if resp.StatusCode() >= 500 {
		var errorResp map[string]interface{}
		if err := json.Unmarshal(resp.Body(), &errorResp); err == nil {
			if msg, ok := errorResp["message"].(string); ok {
				return fmt.Errorf("server error: %s", msg)
			}
		}
		return fmt.Errorf("server error: %s", resp.String())
	}
	if resp.StatusCode() >= 400 {
		return fmt.Errorf("client error (status %d): %s", resp.StatusCode(), resp.String())
	}
	return nil
}
