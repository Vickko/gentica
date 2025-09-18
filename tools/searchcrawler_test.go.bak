package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test tool creation
func TestNewSearchCrawlerTool(t *testing.T) {
	tool := NewSearchCrawlerTool()
	require.NotNil(t, tool)

	// Verify it implements BaseTool interface
	_, ok := tool.(BaseTool)
	assert.True(t, ok, "SearchCrawlerTool should implement BaseTool interface")
}

// Test Name method
func TestSearchCrawlerTool_Name(t *testing.T) {
	tool := NewSearchCrawlerTool()
	assert.Equal(t, SearchCrawlerToolName, tool.Name())
	assert.Equal(t, "searchcrawler", tool.Name())
}

// Test Info method
func TestSearchCrawlerTool_Info(t *testing.T) {
	tool := NewSearchCrawlerTool()
	info := tool.Info()

	// Verify basic info
	assert.Equal(t, SearchCrawlerToolName, info.Name)
	assert.Contains(t, info.Description, "Search the web and crawl")
	assert.Contains(t, info.Description, "WHEN TO USE THIS TOOL")
	assert.Contains(t, info.Description, "HOW TO USE")
	assert.Contains(t, info.Description, "FEATURES")
	assert.Contains(t, info.Description, "LIMITATIONS")

	// Verify required parameters
	assert.Contains(t, info.Required, "query")
	assert.Len(t, info.Required, 1)

	// Verify parameters structure
	assert.Contains(t, info.Parameters, "query")
	assert.Contains(t, info.Parameters, "num_results")

	queryParam, ok := info.Parameters["query"].(map[string]any)
	require.True(t, ok, "query parameter should be a map")
	assert.Equal(t, "string", queryParam["type"])
	assert.Contains(t, queryParam["description"], "search query")

	numResultsParam, ok := info.Parameters["num_results"].(map[string]any)
	require.True(t, ok, "num_results parameter should be a map")
	assert.Equal(t, "number", numResultsParam["type"])
	assert.Contains(t, numResultsParam["description"], "Number of search results")
}

// Test Run with invalid JSON input
func TestSearchCrawlerTool_Run_InvalidJSON(t *testing.T) {
	tool := NewSearchCrawlerTool()

	call := ToolCall{
		ID:    "test-1",
		Name:  "searchcrawler",
		Input: "invalid json {",
	}

	response, err := tool.Run(context.Background(), call)
	require.NoError(t, err)
	assert.True(t, response.IsError)
	assert.Equal(t, ToolResponseTypeText, response.Type)
	assert.Contains(t, response.Content, "Failed to parse searchcrawler parameters")
}

// Test Run with empty query
func TestSearchCrawlerTool_Run_EmptyQuery(t *testing.T) {
	tool := NewSearchCrawlerTool()

	params := SearchCrawlerParams{
		Query: "",
	}
	paramsJSON, err := json.Marshal(params)
	require.NoError(t, err)

	call := ToolCall{
		ID:    "test-2",
		Name:  "searchcrawler",
		Input: string(paramsJSON),
	}

	response, err := tool.Run(context.Background(), call)
	require.NoError(t, err)
	assert.True(t, response.IsError)
	assert.Contains(t, response.Content, "Query parameter is required and cannot be empty")
}

// Test Run with missing query
func TestSearchCrawlerTool_Run_MissingQuery(t *testing.T) {
	tool := NewSearchCrawlerTool()

	call := ToolCall{
		ID:    "test-3",
		Name:  "searchcrawler",
		Input: "{}",
	}

	response, err := tool.Run(context.Background(), call)
	require.NoError(t, err)
	assert.True(t, response.IsError)
	assert.Contains(t, response.Content, "Query parameter is required and cannot be empty")
}

// Test Run with NumResults over limit
func TestSearchCrawlerTool_Run_NumResultsLimit(t *testing.T) {
	t.Skip("Skipping real API test - requires valid API keys")

	tool := NewSearchCrawlerTool()

	params := SearchCrawlerParams{
		Query:      "golang testing",
		NumResults: 100, // Over the limit of 50
	}
	paramsJSON, err := json.Marshal(params)
	require.NoError(t, err)

	call := ToolCall{
		ID:    "test-4",
		Name:  "searchcrawler",
		Input: string(paramsJSON),
	}

	ctx := context.Background()
	response, err := tool.Run(ctx, call)
	require.NoError(t, err)

	// The tool should succeed but cap NumResults at 50
	// We can't verify the exact behavior without making real API calls
	// but we can verify the response structure
	if !response.IsError {
		var result map[string]any
		err = json.Unmarshal([]byte(response.Content), &result)
		require.NoError(t, err)
	}
}

// Test Run with valid parameters
func TestSearchCrawlerTool_Run_Success(t *testing.T) {

	tool := NewSearchCrawlerTool()
	query := "AI technique"
	params := SearchCrawlerParams{
		Query:      query,
		NumResults: 2,
	}
	paramsJSON, err := json.Marshal(params)
	require.NoError(t, err)

	call := ToolCall{
		ID:    "test-5",
		Name:  "searchcrawler",
		Input: string(paramsJSON),
	}

	ctx := context.Background()
	response, err := tool.Run(ctx, call)
	require.NoError(t, err)

	// Print the full response
	t.Logf("Response IsError: %v", response.IsError)
	t.Logf("Response Type: %s", response.Type)
	t.Logf("Response Content: %s", response.Content)
	t.Logf("Response Metadata: %s", response.Metadata)

	if response.IsError {
		t.Logf("API call failed (expected if API keys not configured): %s", response.Content)
		return
	}

	assert.False(t, response.IsError)
	assert.Equal(t, ToolResponseTypeText, response.Type)

	// Verify response is valid JSON
	var result map[string]any
	err = json.Unmarshal([]byte(response.Content), &result)
	require.NoError(t, err)

	// Print parsed result
	t.Logf("Parsed Result:")
	t.Logf("  Query: %v", result["query"])
	t.Logf("  Total Search Results: %v", result["total_search_results"])
	t.Logf("  Total Crawled: %v", result["total_crawled"])

	// Print results array if exists
	if results, ok := result["results"].([]interface{}); ok {
		t.Logf("  Number of Results: %d", len(results))
		for i, r := range results {
			if res, ok := r.(map[string]interface{}); ok {
				t.Logf("  Result %d:", i+1)
				t.Logf("    Title: %v", res["title"])
				t.Logf("    URL: %v", res["url"])
				if content, ok := res["content"].(string); ok {
					t.Logf("    Content: %s", content)
				}
				t.Logf("    Error: %v", res["error"])
			}
		}
	}

	assert.Equal(t, query, result["query"])
	assert.Contains(t, result, "results")
	assert.Contains(t, result, "total_search_results")
	assert.Contains(t, result, "total_crawled")

	// Verify metadata
	assert.NotEmpty(t, response.Metadata)
	var metadata map[string]any
	err = json.Unmarshal([]byte(response.Metadata), &metadata)
	require.NoError(t, err)

	// Print metadata
	t.Logf("Metadata:")
	t.Logf("  Total Search Results: %v", metadata["total_search_results"])
	t.Logf("  Total Crawled: %v", metadata["total_crawled"])
	t.Logf("  Errors Count: %v", metadata["errors_count"])

	assert.Contains(t, metadata, "total_search_results")
	assert.Contains(t, metadata, "total_crawled")
	assert.Contains(t, metadata, "errors_count")
}

// Test parameter validation with various NumResults values
func TestSearchCrawlerTool_Run_NumResultsValues(t *testing.T) {
	tool := NewSearchCrawlerTool()

	testCases := []struct {
		name       string
		numResults int
		shouldFail bool
	}{
		{"Zero NumResults", 0, false},      // Should use default
		{"Negative NumResults", -5, false}, // Should use default
		{"Normal NumResults", 10, false},
		{"Max NumResults", 50, false},
		{"Over Max NumResults", 60, false}, // Should be capped at 50
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params := SearchCrawlerParams{
				Query:      "test query",
				NumResults: tc.numResults,
			}
			paramsJSON, err := json.Marshal(params)
			require.NoError(t, err)

			call := ToolCall{
				ID:    "test-" + tc.name,
				Name:  "searchcrawler",
				Input: string(paramsJSON),
			}

			// Just validate that the parameters are accepted
			// We can't test actual API calls without credentials
			response, err := tool.Run(context.Background(), call)
			require.NoError(t, err)

			// If API keys are not configured, it will return an error
			// but not due to parameter validation
			if response.IsError && !tc.shouldFail {
				assert.NotContains(t, response.Content, "Query parameter")
				assert.NotContains(t, response.Content, "Failed to parse")
			}
		})
	}
}

// Test context cancellation
func TestSearchCrawlerTool_Run_ContextCancellation(t *testing.T) {
	tool := NewSearchCrawlerTool()

	params := SearchCrawlerParams{
		Query:      "test query",
		NumResults: 5,
	}
	paramsJSON, err := json.Marshal(params)
	require.NoError(t, err)

	call := ToolCall{
		ID:    "test-cancel",
		Name:  "searchcrawler",
		Input: string(paramsJSON),
	}

	// Create and immediately cancel context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	response, err := tool.Run(ctx, call)
	require.NoError(t, err)

	// The response might be an error due to context cancellation
	// or API configuration issues
	if response.IsError {
		t.Logf("Response error (expected): %s", response.Content)
	}
}
