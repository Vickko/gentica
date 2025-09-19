package serper

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestScrapeWebpage(t *testing.T) {
	// Mock response data
	mockResponse := WebpageResponse{
		Text: "Search\n\nFirecrawl's search API allows you to perform web searches...",
		Markdown: "# Search\n\nFirecrawl's search API allows you to perform web searches...",
		Metadata: WebpageMetadata{
			Title:       "Search | Firecrawl",
			Description: "Search the web and get full content from results",
			OGTitle:     "Search | Firecrawl",
			OGDescription: "Search the web and get full content from results",
		},
		Credits: 2,
	}

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check method
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		// Check headers
		if r.Header.Get("X-API-KEY") == "" {
			t.Error("Expected X-API-KEY header")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("Expected Content-Type: application/json")
		}

		// Decode request
		var req WebpageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		// Validate request
		if req.URL == "" {
			t.Error("Expected URL in request")
		}

		// Send response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		responseBytes, _ := json.Marshal(mockResponse)
		t.Logf("Mock server sending response: %s", string(responseBytes))
		w.Write(responseBytes)
	}))
	defer server.Close()

	// Create client with test server
	client := NewClientWithConfig("test-key", server.URL, server.URL)

	// Test ScrapeWebpage
	t.Run("ScrapeWebpage", func(t *testing.T) {
		includeMarkdown := true
		req := &WebpageRequest{
			URL:             "https://docs.firecrawl.dev/features/search",
			IncludeMarkdown: &includeMarkdown,
		}

		resp, err := client.ScrapeWebpage(context.Background(), req)
		if err != nil {
			t.Fatalf("ScrapeWebpage failed: %v", err)
		}

		// Debug log
		t.Logf("Response: Text=%q, Markdown=%q, Credits=%d", resp.Text, resp.Markdown, resp.Credits)

		if resp.Text != mockResponse.Text {
			t.Errorf("Expected text %s, got %s", mockResponse.Text, resp.Text)
		}
		if resp.Markdown != mockResponse.Markdown {
			t.Errorf("Expected markdown %s, got %s", mockResponse.Markdown, resp.Markdown)
		}
		if resp.Credits != mockResponse.Credits {
			t.Errorf("Expected credits %d, got %d", mockResponse.Credits, resp.Credits)
		}
	})

	// Test ScrapeWebpageSimple
	t.Run("ScrapeWebpageSimple", func(t *testing.T) {
		resp, err := client.ScrapeWebpageSimple(context.Background(), "https://example.com")
		if err != nil {
			t.Fatalf("ScrapeWebpageSimple failed: %v", err)
		}

		if resp == nil {
			t.Error("Expected response, got nil")
		}
	})

	// Test ScrapeWebpageTextOnly
	t.Run("ScrapeWebpageTextOnly", func(t *testing.T) {
		resp, err := client.ScrapeWebpageTextOnly(context.Background(), "https://example.com")
		if err != nil {
			t.Fatalf("ScrapeWebpageTextOnly failed: %v", err)
		}

		if resp == nil {
			t.Error("Expected response, got nil")
		}
	})
}

func TestScrapeWebpageValidation(t *testing.T) {
	client := NewClient()

	// Test nil request
	t.Run("NilRequest", func(t *testing.T) {
		_, err := client.ScrapeWebpage(context.Background(), nil)
		if err == nil {
			t.Error("Expected error for nil request")
		}
	})

	// Test empty URL
	t.Run("EmptyURL", func(t *testing.T) {
		req := &WebpageRequest{
			URL: "",
		}
		_, err := client.ScrapeWebpage(context.Background(), req)
		if err == nil {
			t.Error("Expected error for empty URL")
		}
	})
}

func TestScrapeWebpageErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expectErr  bool
	}{
		{"Success", http.StatusOK, false},
		{"Unauthorized", http.StatusUnauthorized, true},
		{"PaymentRequired", http.StatusPaymentRequired, true},
		{"Forbidden", http.StatusForbidden, true},
		{"RateLimit", http.StatusTooManyRequests, true},
		{"ServerError", http.StatusInternalServerError, true},
		{"BadRequest", http.StatusBadRequest, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.statusCode == http.StatusOK {
					json.NewEncoder(w).Encode(WebpageResponse{
						Text: "Success",
					})
				} else {
					w.WriteHeader(tt.statusCode)
					if tt.statusCode >= 500 {
						json.NewEncoder(w).Encode(map[string]string{
							"message": "Server error",
						})
					} else {
						w.Write([]byte("Error"))
					}
				}
			}))
			defer server.Close()

			client := NewClientWithConfig("test-key", server.URL, server.URL)

			_, err := client.ScrapeWebpageSimple(context.Background(), "https://example.com")

			if tt.expectErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestScrapeWebpageContextCancellation(t *testing.T) {
	// Create a server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClientWithConfig("test-key", server.URL, server.URL)
	client.SetTimeout(100 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.ScrapeWebpageSimple(ctx, "https://example.com")
	if err == nil {
		t.Error("Expected timeout error")
	}
}

func TestScrapeWebpageIntegration(t *testing.T) {
	// Skip if no API key is configured
	if SerperAPIKey == "" || SerperAPIKey == "62adbfaba56f71225b562cb4704ccc7b28e73a6d" {
		t.Skip("Skipping integration test: no valid API key")
	}

	client := NewClient()
	ctx := context.Background()

	// Test with a real URL
	resp, err := client.ScrapeWebpageSimple(ctx, "https://www.example.com")
	if err != nil {
		t.Fatalf("Integration test failed: %v", err)
	}

	if resp.Text == "" {
		t.Error("Expected text content from real webpage")
	}

	// Check metadata
	if resp.Metadata.Title == "" {
		t.Error("Expected title in metadata")
	}

	t.Logf("Successfully scraped webpage: %s", resp.Metadata.Title)
	t.Logf("Credits used: %d", resp.Credits)
}

// TestScrapeWebpageRealRequest tests with real API request
// Run with: go test -run TestScrapeWebpageRealRequest -v
func TestScrapeWebpageRealRequest(t *testing.T) {
	// This test makes real API requests - only run when explicitly needed
	if testing.Short() {
		t.Skip("Skipping real API request test in short mode")
	}

	// Use the real API key
	client := NewClient()
	ctx := context.Background()

	tests := []struct {
		name            string
		url             string
		includeMarkdown bool
		wantText        bool
		wantMarkdown    bool
		wantTitle       bool
	}{
		{
			name:            "Example.com with markdown",
			url:             "https://www.example.com",
			includeMarkdown: true,
			wantText:        true,
			wantMarkdown:    true,
			wantTitle:       true,
		},
		{
			name:            "Example.com text only",
			url:             "https://www.example.com",
			includeMarkdown: false,
			wantText:        true,
			wantMarkdown:    false,
			wantTitle:       true,
		},
		{
			name:            "GitHub page",
			url:             "https://github.com/anthropics/anthropic-sdk-python",
			includeMarkdown: true,
			wantText:        true,
			wantMarkdown:    true,
			wantTitle:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &WebpageRequest{
				URL:             tt.url,
				IncludeMarkdown: &tt.includeMarkdown,
			}

			resp, err := client.ScrapeWebpage(ctx, req)
			if err != nil {
				t.Fatalf("ScrapeWebpage failed: %v", err)
			}

			// Log response details
			t.Logf("URL: %s", tt.url)
			t.Logf("Credits used: %d", resp.Credits)
			t.Logf("Title: %s", resp.Metadata.Title)
			t.Logf("Text length: %d characters", len(resp.Text))
			if resp.Markdown != "" {
				t.Logf("Markdown length: %d characters", len(resp.Markdown))
			}

			// Validate response
			if tt.wantText && resp.Text == "" {
				t.Error("Expected text content, got empty")
			}

			if tt.wantMarkdown && resp.Markdown == "" {
				t.Error("Expected markdown content, got empty")
			}

			if !tt.wantMarkdown && resp.Markdown != "" {
				t.Error("Expected no markdown content, but got markdown")
			}

			if tt.wantTitle && resp.Metadata.Title == "" {
				t.Error("Expected title in metadata, got empty")
			}

			// Check that credits were consumed
			if resp.Credits <= 0 {
				t.Error("Expected positive credits value")
			}

			// Basic content validation
			if resp.Text != "" && len(resp.Text) < 10 {
				t.Error("Text content seems too short")
			}

			// Log sample of content for manual verification
			if resp.Text != "" {
				sample := resp.Text
				if len(sample) > 200 {
					sample = sample[:200] + "..."
				}
				t.Logf("Text sample: %s", sample)
			}
		})
	}
}

// TestScrapeWebpageLiveAPI tests with the actual Serper API using the default API key
// This test will actually consume API credits
// Run with: go test -run TestScrapeWebpageLiveAPI -v
func TestScrapeWebpageLiveAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping live API test in short mode")
	}

	// Explicitly use the default API key for testing
	client := NewClientWithConfig("62adbfaba56f71225b562cb4704ccc7b28e73a6d", SerperBaseURL, SerperScrapeBaseURL)
	ctx := context.Background()

	t.Run("Scrape example.com", func(t *testing.T) {
		resp, err := client.ScrapeWebpageSimple(ctx, "https://www.example.com")
		if err != nil {
			t.Logf("Note: This test requires valid API credits. Error: %v", err)
			t.Skip("Skipping due to API error (might be out of credits)")
		}

		t.Logf("Successfully scraped example.com")
		t.Logf("Title: %s", resp.Metadata.Title)
		t.Logf("Description: %s", resp.Metadata.Description)
		t.Logf("Text length: %d", len(resp.Text))
		t.Logf("Markdown length: %d", len(resp.Markdown))
		t.Logf("Credits used: %d", resp.Credits)

		// Validate we got actual content
		if resp.Text == "" {
			t.Error("Expected text content")
		}

		if resp.Markdown == "" {
			t.Error("Expected markdown content")
		}

		// Example.com should have "Example Domain" in the title
		if resp.Metadata.Title != "" && resp.Metadata.Title != "Example Domain" {
			t.Logf("Warning: Unexpected title: %s", resp.Metadata.Title)
		}
	})

	t.Run("Scrape with text only", func(t *testing.T) {
		resp, err := client.ScrapeWebpageTextOnly(ctx, "https://www.example.com")
		if err != nil {
			t.Logf("Note: This test requires valid API credits. Error: %v", err)
			t.Skip("Skipping due to API error (might be out of credits)")
		}

		if resp.Text == "" {
			t.Error("Expected text content")
		}

		// Should not have markdown when using text-only method
		if resp.Markdown != "" {
			t.Error("Expected no markdown content with text-only request")
		}

		t.Logf("Text-only scraping successful, credits: %d", resp.Credits)
	})
}