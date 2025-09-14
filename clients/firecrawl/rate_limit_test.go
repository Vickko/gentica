package firecrawl

import (
	"context"
	"testing"
	"time"
)

// TestRateLimiting tests the rate limiting functionality
func TestRateLimiting(t *testing.T) {
	// Save original settings
	originalEnable := EnableRateLimit
	originalRate := RateLimitPerMinute
	
	// Restore settings after test
	defer func() {
		EnableRateLimit = originalEnable
		RateLimitPerMinute = originalRate
	}()
	
	t.Run("RateLimitEnabled", func(t *testing.T) {
		// Enable rate limiting with 60 requests per minute for testing
		// This means 1 request per second
		EnableRateLimit = true
		RateLimitPerMinute = 60
		
		client := NewClient()
		
		// Verify rate limiter is created
		if client.rateLimiter == nil {
			t.Fatal("Expected rate limiter to be created when enabled")
		}
		
		ctx := context.Background()
		startTime := time.Now()
		
		// Make 3 quick requests
		// First should be immediate, others should be rate limited
		for i := 0; i < 3; i++ {
			req, err := client.createRequestWithContext(ctx)
			if err != nil {
				t.Fatalf("Request %d failed: %v", i+1, err)
			}
			if req == nil {
				t.Fatal("Expected request to be created")
			}
		}
		
		elapsed := time.Since(startTime)
		
		// With 60 req/min (1 req/sec), 3 requests should take at least 2 seconds
		if elapsed < 2*time.Second {
			t.Errorf("Expected rate limiting to slow down requests, took only %v", elapsed)
		}
		
		t.Logf("3 requests with rate limiting took %v", elapsed)
	})
	
	t.Run("RateLimitDisabled", func(t *testing.T) {
		// Disable rate limiting
		EnableRateLimit = false
		
		client := NewClient()
		
		// Verify rate limiter is not created
		if client.rateLimiter != nil {
			t.Fatal("Expected rate limiter to be nil when disabled")
		}
		
		ctx := context.Background()
		
		// Make multiple requests quickly
		startTime := time.Now()
		for i := 0; i < 5; i++ {
			req, err := client.createRequestWithContext(ctx)
			if err != nil {
				t.Fatalf("Request %d failed: %v", i+1, err)
			}
			if req == nil {
				t.Fatal("Expected request to be created")
			}
		}
		elapsed := time.Since(startTime)
		
		// All requests should complete quickly (< 1 second)
		if elapsed > 1*time.Second {
			t.Errorf("Requests took too long without rate limiting: %v", elapsed)
		}
	})
	
	
	t.Run("ContextCancellation", func(t *testing.T) {
		// Enable rate limiting
		EnableRateLimit = true
		RateLimitPerMinute = 1
		
		client := NewClient()
		
		// Make first request
		ctx := context.Background()
		_, err := client.createRequestWithContext(ctx)
		if err != nil {
			t.Fatalf("First request failed: %v", err)
		}
		
		// Create a context that we'll cancel immediately
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately
		
		// Try to make second request with cancelled context
		_, err = client.createRequestWithContext(ctx)
		if err == nil {
			t.Error("Expected error when context is cancelled")
		}
	})
}