# Serper Client

A Go client for the Serper API, providing easy access to Google search results programmatically.

## Features

- Multiple search types support (web, images, videos, news, shopping, places, scholar, patents)
- Webpage scraping with markdown and text extraction
- Environment variable configuration
- Optional rate limiting
- Singleton pattern for global rate limiting
- Comprehensive error handling
- Simple and advanced search methods

## Configuration

The client can be configured using environment variables:

```bash
export SERPER_API_KEY="your-api-key"
export SERPER_BASE_URL="https://google.serper.dev"  # optional, defaults to official API
export SERPER_SCRAPE_BASE_URL="https://scrape.serper.dev"  # optional, defaults to official scrape API
```

## Usage

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "gentica/clients/serper"
)

func main() {
    // Create a new client (uses environment variables)
    client := serper.NewClient()
    
    // Or use the singleton instance for global rate limiting
    client = serper.GetSingleton()
    
    ctx := context.Background()
    
    // Simple search
    response, err := client.SearchSimple(ctx, "golang programming")
    if err != nil {
        panic(err)
    }
    
    // Process results
    for _, result := range response.Organic {
        fmt.Printf("Title: %s\n", result.Title)
        fmt.Printf("Link: %s\n", result.Link)
        fmt.Printf("Snippet: %s\n\n", result.Snippet)
    }
}
```

### Advanced Search Options

```go
// Custom search request with all options
request := &serper.SearchRequest{
    Query:       "machine learning",
    Type:        serper.SearchTypeSearch,
    GL:          "us",      // Country
    HL:          "en",      // Language
    Num:         20,        // Number of results
    Page:        1,         // Page number
    SafeSearch:  true,      // Safe search filter
    AutoCorrect: true,      // Auto-correct typos
    Location:    "New York", // Location for local results
}

response, err := client.Search(ctx, request)
```

### Search Different Content Types

```go
// Image search
imageResults, err := client.SearchImages(ctx, "cute puppies")

// Video search
videoResults, err := client.SearchVideos(ctx, "golang tutorial")

// News search
newsResults, err := client.SearchNews(ctx, "technology news")

// Shopping search
shoppingResults, err := client.SearchShopping(ctx, "laptop")

// Places search
placesResults, err := client.SearchPlaces(ctx, "restaurants near me")

// Google Scholar search
scholarResults, err := client.SearchScholar(ctx, "machine learning papers")

// Patent search
patentResults, err := client.SearchPatents(ctx, "artificial intelligence")
```

### Search with Specific Options

```go
// Search with pagination
response, err := client.SearchWithPagination(ctx, "golang", 2, 10) // page 2, 10 results

// Search with location
response, err := client.SearchWithLocation(ctx, "coffee shops", "San Francisco")

// Search with language settings
response, err := client.SearchWithLanguage(ctx, "news", "fr", "fr") // French country and language
```

### Webpage Scraping

Scrape webpages and extract content in text and markdown formats:

```go
// Simple webpage scraping with markdown
response, err := client.ScrapeWebpageSimple(ctx, "https://docs.firecrawl.dev/features/search")
if err != nil {
    panic(err)
}

fmt.Printf("Title: %s\n", response.Metadata.Title)
fmt.Printf("Text: %s\n", response.Text)
fmt.Printf("Markdown: %s\n", response.Markdown)
fmt.Printf("Credits used: %d\n", response.Credits)

// Scrape webpage with text only (no markdown)
response, err = client.ScrapeWebpageTextOnly(ctx, "https://example.com")

// Advanced scraping with custom options
includeMarkdown := true
request := &serper.WebpageRequest{
    URL:             "https://example.com",
    IncludeMarkdown: &includeMarkdown,
}
response, err = client.ScrapeWebpage(ctx, request)
```

The webpage scraping response includes:
- **Text**: Plain text content of the webpage
- **Markdown**: Formatted markdown content (if requested)
- **Metadata**: Page metadata including title, description, Open Graph tags, Twitter cards, etc.
- **Credits**: Number of API credits consumed

### Custom Client Configuration

```go
// Create client with custom configuration
client := serper.NewClientWithConfig("your-api-key", "https://custom.serper.dev", "https://custom.scraper.dev")

// Update configuration
client.SetAPIKey("new-api-key")
client.SetBaseURL("https://new.serper.dev")
client.SetScrapeBaseURL("https://new.scraper.dev")
client.SetTimeout(60 * time.Second)
```

### Rate Limiting

Rate limiting can be enabled globally:

```go
// Enable rate limiting (in client.go)
serper.EnableRateLimit = true
serper.RateLimitPerMinute = 100 // Default is 100 requests per minute

// Create client with rate limiting
client := serper.NewClient()
// Now all requests through this client will be rate limited
```

## Response Structure

The `SearchResponse` includes various types of results depending on the search type:

- **Organic Results**: Standard web search results
- **Answer Box**: Direct answer to the query
- **Knowledge Graph**: Structured information about entities
- **People Also Ask**: Related questions
- **Related Searches**: Similar search queries
- **Images/Videos/News**: Media-specific results
- **Shopping**: Product listings with prices
- **Places**: Location-based results with coordinates
- **Scholar**: Academic papers and citations
- **Patents**: Patent information

## Error Handling

The client provides detailed error messages for various scenarios:

```go
response, err := client.SearchSimple(ctx, "query")
if err != nil {
    switch {
    case strings.Contains(err.Error(), "unauthorized"):
        // Invalid API key
    case strings.Contains(err.Error(), "payment required"):
        // Insufficient credits
    case strings.Contains(err.Error(), "rate limit"):
        // Rate limit exceeded
    default:
        // Other errors
    }
}
```

## Testing

Run tests with:

```bash
# Run all tests
go test ./clients/serper

# Run with verbose output
go test ./clients/serper -v

# Run integration tests (requires valid API key)
go test ./clients/serper -v -run TestSearchIntegration
```