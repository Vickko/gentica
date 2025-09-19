package serper

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient()
	
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	
	if client.apiKey == "" {
		t.Error("API key is empty")
	}
	
	if client.baseURL == "" {
		t.Error("Base URL is empty")
	}
}

func TestNewClientWithConfig(t *testing.T) {
	apiKey := "test-api-key"
	baseURL := "https://test.serper.dev"
	
	client := NewClientWithConfig(apiKey, baseURL, "https://scrape.serper.dev")
	
	if client == nil {
		t.Fatal("NewClientWithConfig returned nil")
	}
	
	if client.apiKey != apiKey {
		t.Errorf("Expected API key %s, got %s", apiKey, client.apiKey)
	}
	
	if client.baseURL != baseURL {
		t.Errorf("Expected base URL %s, got %s", baseURL, client.baseURL)
	}
}


func TestSetters(t *testing.T) {
	client := NewClient()
	
	newAPIKey := "new-api-key"
	client.SetAPIKey(newAPIKey)
	if client.apiKey != newAPIKey {
		t.Errorf("SetAPIKey failed: expected %s, got %s", newAPIKey, client.apiKey)
	}
	
	newBaseURL := "https://new.serper.dev"
	client.SetBaseURL(newBaseURL)
	if client.baseURL != newBaseURL {
		t.Errorf("SetBaseURL failed: expected %s, got %s", newBaseURL, client.baseURL)
	}
	
	newTimeout := 60 * time.Second
	client.SetTimeout(newTimeout)
	// Note: We can't directly test the timeout value as it's internal to resty
}

func TestSearchValidation(t *testing.T) {
	client := NewClient()
	ctx := context.Background()
	
	// Test nil request
	_, err := client.Search(ctx, nil)
	if err == nil {
		t.Error("Expected error for nil request")
	}
	
	// Test empty query
	_, err = client.Search(ctx, &SearchRequest{})
	if err == nil {
		t.Error("Expected error for empty query")
	}
}

func TestSearchDefaults(t *testing.T) {
	// This test verifies that default values are set correctly
	// In a real test, you would mock the HTTP response
	_ = NewClient() // client created but not used in this test
	
	request := &SearchRequest{
		Query: "test query",
	}
	
	// Check that defaults would be applied (without making actual API call)
	if request.Type != "" {
		t.Error("Type should be empty before search")
	}
	
	// Note: To fully test this, you would need to mock the HTTP client
	// or use a test server that validates the request
}

func TestSearchTypes(t *testing.T) {
	// Test that all search type constants are defined correctly
	searchTypes := []SearchType{
		SearchTypeSearch,
		SearchTypeImages,
		SearchTypeVideos,
		SearchTypeNews,
		SearchTypeShopping,
		SearchTypePlaces,
		SearchTypeScholar,
		SearchTypePatents,
	}
	
	expectedTypes := []string{
		"search",
		"images",
		"videos",
		"news",
		"shopping",
		"places",
		"scholar",
		"patents",
	}
	
	for i, st := range searchTypes {
		if string(st) != expectedTypes[i] {
			t.Errorf("SearchType %d: expected %s, got %s", i, expectedTypes[i], string(st))
		}
	}
}

func TestEnvironmentVariables(t *testing.T) {
	// Save original values
	originalAPIKey := os.Getenv("SERPER_API_KEY")
	originalBaseURL := os.Getenv("SERPER_BASE_URL")
	
	// Set test values
	testAPIKey := "test-env-api-key"
	testBaseURL := "https://test-env.serper.dev"
	
	os.Setenv("SERPER_API_KEY", testAPIKey)
	os.Setenv("SERPER_BASE_URL", testBaseURL)
	
	// Note: Since we removed singleton, we just reinitialize the global vars
	
	// Reinitialize with new env vars
	SerperAPIKey = os.Getenv("SERPER_API_KEY")
	if SerperAPIKey == "" {
		SerperAPIKey = "62adbfaba56f71225b562cb4704ccc7b28e73a6d"
	}
	
	SerperBaseURL = os.Getenv("SERPER_BASE_URL")
	if SerperBaseURL == "" {
		SerperBaseURL = "https://google.serper.dev"
	}
	
	client := NewClient()
	
	if client.apiKey != testAPIKey {
		t.Errorf("Expected API key from env %s, got %s", testAPIKey, client.apiKey)
	}
	
	if client.baseURL != testBaseURL {
		t.Errorf("Expected base URL from env %s, got %s", testBaseURL, client.baseURL)
	}
	
	// Restore original values
	if originalAPIKey != "" {
		os.Setenv("SERPER_API_KEY", originalAPIKey)
	} else {
		os.Unsetenv("SERPER_API_KEY")
	}
	
	if originalBaseURL != "" {
		os.Setenv("SERPER_BASE_URL", originalBaseURL)
	} else {
		os.Unsetenv("SERPER_BASE_URL")
	}
	
	// Reset globals to original values
	SerperAPIKey = originalAPIKey
	if SerperAPIKey == "" {
		SerperAPIKey = "62adbfaba56f71225b562cb4704ccc7b28e73a6d"
	}
	
	SerperBaseURL = originalBaseURL
	if SerperBaseURL == "" {
		SerperBaseURL = "https://google.serper.dev"
	}
}

func TestRateLimiting(t *testing.T) {
	// Save original value
	originalEnableRateLimit := EnableRateLimit
	
	// Test with rate limiting enabled
	EnableRateLimit = true
	RateLimitPerMinute = 60 // 1 per second for testing
	
	client := NewClientWithConfig("test-key", "https://test.serper.dev", "https://scrape.test.serper.dev")
	
	if client.rateLimiter == nil {
		t.Error("Rate limiter should be created when enabled")
	}
	
	// Test with rate limiting disabled
	EnableRateLimit = false
	
	client2 := NewClientWithConfig("test-key", "https://test.serper.dev", "https://scrape.test.serper.dev")
	
	if client2.rateLimiter != nil {
		t.Error("Rate limiter should be nil when disabled")
	}
	
	// Restore original value
	EnableRateLimit = originalEnableRateLimit
}

// Integration test - only run with actual API key
func TestSearchIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	// Check if we have a valid API key
	if SerperAPIKey == "" || SerperAPIKey == "62adbfaba56f71225b562cb4704ccc7b28e73a6d" {
		t.Skip("Skipping integration test - no valid API key set")
	}
	
	client := NewClient()
	ctx := context.Background()
	
	// Test simple search
	response, err := client.SearchSimple(ctx, "golang testing")
	if err != nil {
		t.Fatalf("SearchSimple failed: %v", err)
	}
	
	if response == nil {
		t.Fatal("Response is nil")
	}
	
	if len(response.Organic) == 0 {
		t.Error("No organic results returned")
	}
	
	// Test image search
	imageResponse, err := client.SearchImages(ctx, "golang gopher")
	if err != nil {
		t.Fatalf("SearchImages failed: %v", err)
	}
	
	if imageResponse == nil {
		t.Fatal("Image response is nil")
	}
	
	// Test search with pagination
	pageResponse, err := client.SearchWithPagination(ctx, "golang", 2, 5)
	if err != nil {
		t.Fatalf("SearchWithPagination failed: %v", err)
	}
	
	if pageResponse == nil {
		t.Fatal("Pagination response is nil")
	}
}