package firecrawl

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestClient tests basic client functionality
func TestClient(t *testing.T) {
	t.Run("NewClient", func(t *testing.T) {
		client := NewClient()
		if client == nil {
			t.Fatal("Expected client to be created")
		}
		if client.apiKey != FirecrawlAPIKey {
			t.Errorf("Expected API key to be %s, got %s", FirecrawlAPIKey, client.apiKey)
		}
		if client.baseURL != FirecrawlAPIURL {
			t.Errorf("Expected base URL to be %s, got %s", FirecrawlAPIURL, client.baseURL)
		}
	})

	t.Run("NewClientWithConfig", func(t *testing.T) {
		apiKey := "test-api-key"
		baseURL := "https://test.api.com"
		
		client := NewClientWithConfig(apiKey, baseURL)
		if client == nil {
			t.Fatal("Expected client to be created")
		}
		if client.apiKey != apiKey {
			t.Errorf("Expected API key to be %s, got %s", apiKey, client.apiKey)
		}
		if client.baseURL != baseURL {
			t.Errorf("Expected base URL to be %s, got %s", baseURL, client.baseURL)
		}
	})

	t.Run("SetAPIKey", func(t *testing.T) {
		client := NewClient()
		newKey := "new-api-key"
		client.SetAPIKey(newKey)
		if client.apiKey != newKey {
			t.Errorf("Expected API key to be %s, got %s", newKey, client.apiKey)
		}
	})

	t.Run("SetBaseURL", func(t *testing.T) {
		client := NewClient()
		newURL := "https://new.api.com"
		client.SetBaseURL(newURL)
		if client.baseURL != newURL {
			t.Errorf("Expected base URL to be %s, got %s", newURL, client.baseURL)
		}
	})

	t.Run("SetTimeout", func(t *testing.T) {
		client := NewClient()
		timeout := 5 * time.Minute
		client.SetTimeout(timeout)
		// Note: We can't directly test the timeout value, but we can ensure the method doesn't panic
	})
}

// TestScrapeOptions tests the scrape options validation
func TestScrapeOptions(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	t.Run("NilOptions", func(t *testing.T) {
		_, err := client.Scrape(ctx, nil)
		if err == nil {
			t.Error("Expected error for nil options")
		}
	})

	t.Run("EmptyURL", func(t *testing.T) {
		_, err := client.Scrape(ctx, &ScrapeOptions{})
		if err == nil {
			t.Error("Expected error for empty URL")
		}
	})
}

// TestScrapeIntegration tests the actual scraping functionality
// This requires a valid API key to be set in environment variable
func TestScrapeIntegration(t *testing.T) {
	// Skip this test if no API key is provided
	apiKey := os.Getenv("FIRECRAWL_API_KEY")
	if apiKey == "" {
		t.Skip("FIRECRAWL_API_KEY environment variable not set, skipping integration tests")
	}

	client := NewClientWithConfig(apiKey, FirecrawlAPIURL)
	ctx := context.Background()

	t.Run("BasicScrape", func(t *testing.T) {
		resp, err := client.ScrapeSimple(ctx, "https://example.com")
		if err != nil {
			t.Fatalf("Failed to scrape: %v", err)
		}

		if resp == nil {
			t.Fatal("Expected response, got nil")
		}

		if !resp.Success {
			t.Error("Expected successful response")
		}

		if resp.Data == nil {
			t.Fatal("Expected data in response")
		}

		if resp.Data.Markdown == "" {
			t.Error("Expected markdown content")
		}
		
		t.Logf("Successfully scraped page with markdown length: %d", len(resp.Data.Markdown))
	})

	t.Run("ScrapeWithFormats", func(t *testing.T) {
		formats := []Format{
			SimpleFormat{Type: FormatMarkdown},
			SimpleFormat{Type: FormatHTML},
		}

		resp, err := client.ScrapeWithFormats(ctx, "https://example.com", formats)
		if err != nil {
			t.Fatalf("Failed to scrape with formats: %v", err)
		}

		if resp == nil || !resp.Success {
			t.Fatal("Expected successful response")
		}

		if resp.Data == nil {
			t.Fatal("Expected data in response")
		}

		if resp.Data.Markdown == "" {
			t.Error("Expected markdown content")
		}

		if resp.Data.HTML == nil || *resp.Data.HTML == "" {
			t.Error("Expected HTML content")
		}
	})

	t.Run("ScrapeWithMetadata", func(t *testing.T) {
		resp, err := client.Scrape(ctx, &ScrapeOptions{
			URL: "https://example.com",
		})
		if err != nil {
			t.Fatalf("Failed to scrape: %v", err)
		}

		if resp == nil || !resp.Success {
			t.Fatal("Expected successful response")
		}

		if resp.Data == nil {
			t.Fatal("Expected data in response")
		}

		if resp.Data.Metadata != nil {
			t.Logf("Page metadata - Title: %s, Status: %d", 
				resp.Data.Metadata.Title, 
				resp.Data.Metadata.StatusCode)
		}
	})
}

// TestFormatTypes tests the format type implementations
func TestFormatTypes(t *testing.T) {
	// Test simple format
	sf := SimpleFormat{Type: FormatMarkdown}
	sf.formatMarker() // Should not panic

	// Test screenshot format
	quality := 90
	fullPage := true
	scf := ScreenshotFormat{
		Type:     "screenshot",
		Quality:  &quality,
		FullPage: &fullPage,
		Viewport: &Viewport{Width: 1920, Height: 1080},
	}
	scf.formatMarker() // Should not panic

	// Test JSON format
	jf := JSONFormat{
		Type:   "json",
		Schema: map[string]interface{}{"type": "object"},
		Prompt: "Extract data",
	}
	jf.formatMarker() // Should not panic

	// Test change tracking format
	ctf := ChangeTrackingFormat{
		Type:  "changeTracking",
		Modes: []string{"git-diff", "json"},
	}
	ctf.formatMarker() // Should not panic
}

// TestActionTypes tests the action type implementations
func TestActionTypes(t *testing.T) {
	// Test wait action
	ms := 1000
	wa := WaitAction{Type: "wait", Milliseconds: &ms}
	wa.actionMarker() // Should not panic

	// Test click action
	ca := ClickAction{Type: "click", Selector: "#button"}
	ca.actionMarker() // Should not panic

	// Test write action
	wra := WriteAction{Type: "write", Text: "Hello"}
	wra.actionMarker() // Should not panic

	// Test other action types
	pa := PressAction{Type: "press", Key: "Enter"}
	pa.actionMarker()

	sa := ScrollAction{Type: "scroll", Direction: &[]string{"down"}[0]}
	sa.actionMarker()

	sca := ScrapeAction{Type: "scrape"}
	sca.actionMarker()

	eja := ExecuteJavaScriptAction{Type: "executeJavascript", Script: "console.log('test')"}
	eja.actionMarker()

	pdfa := PDFAction{Type: "pdf", Format: &[]string{"A4"}[0]}
	pdfa.actionMarker()
}

// TestParserTypes tests the parser type implementations
func TestParserTypes(t *testing.T) {
	// Test simple parser
	sp := SimpleParser(ParserPDF)
	sp.parserMarker() // Should not panic

	// Test PDF parser
	maxPages := 10
	pp := PDFParser{Type: "pdf", MaxPages: &maxPages}
	pp.parserMarker() // Should not panic
}