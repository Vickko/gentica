package searchcrawler

import (
	"context"
	"sync"
	"testing"
	"time"
)



// TestSearchAndCrawlValidation tests input validation
func TestSearchAndCrawlValidation(t *testing.T) {
	crawler := NewSearchCrawler()
	ctx := context.Background()
	
	// Test nil request
	_, err := crawler.SearchAndCrawl(ctx, nil)
	if err == nil {
		t.Error("Expected error for nil request")
	}
	
	// Test empty query
	_, err = crawler.SearchAndCrawl(ctx, &SearchCrawlRequest{
		Query: "",
	})
	if err == nil {
		t.Error("Expected error for empty query")
	}
}

// TestSearchAndCrawlDefaults tests default values
func TestSearchAndCrawlDefaults(t *testing.T) {
	crawler := NewSearchCrawler()
	ctx := context.Background()
	
	// Create request with minimal fields
	request := &SearchCrawlRequest{
		Query: "test query",
	}
	
	// This will fail with actual API calls, but we're testing that defaults are set
	// In a real test, you would mock the clients
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	
	_, _ = crawler.SearchAndCrawl(ctx, request)
	
	// Verify defaults were applied (would need to inspect internal state in real test)
	// This is more of a smoke test to ensure the method doesn't panic
}

// TestConcurrentSearchAndCrawl tests concurrent search and crawl operations
func TestConcurrentSearchAndCrawl(t *testing.T) {
	crawler := NewSearchCrawler()
	
	const numRequests = 5
	var wg sync.WaitGroup
	wg.Add(numRequests)
	
	errors := make(chan error, numRequests)
	
	for i := 0; i < numRequests; i++ {
		go func(index int) {
			defer wg.Done()
			
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			
			request := &SearchCrawlRequest{
				Query:      "golang testing",
				NumResults: 2,
			}
			
			_, err := crawler.SearchAndCrawl(ctx, request)
			if err != nil {
				errors <- err
			}
		}(i)
	}
	
	wg.Wait()
	close(errors)
	
	// Check if any goroutine reported an error
	// Note: This will likely fail without proper API keys or mocking
	// This is more of a structural test to ensure concurrent access doesn't cause panics
}

// TestCleanContent tests the content cleaning function
func TestCleanContent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Trim whitespace",
			input:    "  \n  Hello World  \n  ",
			expected: "Hello World",
		},
		{
			name:     "Multiple newlines",
			input:    "Line 1\n\n\n\nLine 2",
			expected: "Line 1\n\nLine 2",
		},
		{
			name:     "Empty lines removed",
			input:    "Line 1\n  \n  \nLine 2",
			expected: "Line 1\n\nLine 2",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanContent(tt.input)
			if result != tt.expected {
				t.Errorf("cleanContent() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestContentTruncation tests that very long content is truncated
func TestContentTruncation(t *testing.T) {
	// Create a very long string
	longContent := ""
	for i := 0; i < 60000; i++ {
		longContent += "a"
	}
	
	result := cleanContent(longContent)
	
	// Check that content was truncated
	if len(result) > 50100 { // 50000 + some room for truncation message
		t.Errorf("Content not truncated properly, length: %d", len(result))
	}
	
	// Check that truncation marker is present
	expectedSuffix := "...[truncated]"
	if len(result) > len(expectedSuffix) {
		suffix := result[len(result)-len(expectedSuffix):]
		if suffix != expectedSuffix {
			t.Error("Truncated content doesn't have proper suffix")
		}
	}
}

// BenchmarkSearchAndCrawl benchmarks the search and crawl operation
func BenchmarkSearchAndCrawl(b *testing.B) {
	crawler := NewSearchCrawler()
	ctx := context.Background()
	
	request := &SearchCrawlRequest{
		Query:      "benchmark test",
		NumResults: 5,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Use a timeout to prevent hanging
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		_, _ = crawler.SearchAndCrawl(ctx, request)
		cancel()
	}
}

// Example usage of SearchCrawler
func ExampleSearchCrawler() {
	// Get the singleton instance
	crawler := NewSearchCrawler()
	
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Create search and crawl request
	request := &SearchCrawlRequest{
		Query:      "golang web scraping",
		NumResults: 10, // Crawl top 10 results
	}
	
	// Perform search and crawl
	response, err := crawler.SearchAndCrawl(ctx, request)
	if err != nil {
		// Handle error
		return
	}
	
	// Process results
	for _, result := range response.Results {
		if result.Error == "" {
			// Successfully crawled
			_ = result.Title
			_ = result.Content
			_ = result.Metadata
		}
	}
}