package searchcrawler

import (
	"sync"
)

var (
	// singletonInstance holds the singleton instance of SearchCrawler
	singletonInstance SearchCrawler
	
	// singletonOnce ensures the singleton is initialized only once
	singletonOnce sync.Once
)

// GetSingleton returns the singleton instance of SearchCrawler
// This ensures that:
// 1. Only one instance of SearchCrawler exists in the application
// 2. The underlying Serper and Firecrawl clients are also singletons
// 3. Rate limiting is properly applied across all requests
func GetSingleton() SearchCrawler {
	singletonOnce.Do(func() {
		singletonInstance = NewSearchCrawler()
	})
	return singletonInstance
}

// ResetSingleton resets the singleton instance (mainly for testing)
// This should NOT be used in production code
func ResetSingleton() {
	singletonOnce = sync.Once{}
	singletonInstance = nil
}