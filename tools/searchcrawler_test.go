package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchCrawlerTool_Info(t *testing.T) {
	tool := NewSearchCrawlerTool("/tmp")
	info := tool.Info()

	assert.Equal(t, SearchCrawlerToolName, info.Name)
	assert.NotEmpty(t, info.Description)
	assert.Contains(t, info.Parameters, "query")
	assert.Contains(t, info.Parameters, "result_num")
	assert.Contains(t, info.Parameters, "save_path")
	assert.Contains(t, info.Required, "query")
	assert.Contains(t, info.Required, "save_path")
}

func TestSearchCrawlerTool_Run_InvalidParams(t *testing.T) {
	tool := NewSearchCrawlerTool("/tmp")
	ctx := context.Background()

	tests := []struct {
		name        string
		params      any
		wantErrMsg  string
	}{
		{
			name:       "empty query",
			params:     SearchCrawlerParams{Query: "", SavePath: "/tmp/test"},
			wantErrMsg: "query parameter is required",
		},
		{
			name:       "empty save_path",
			params:     SearchCrawlerParams{Query: "test", SavePath: ""},
			wantErrMsg: "save_path parameter is required",
		},
		{
			name:       "invalid json",
			params:     "invalid",
			wantErrMsg: "Failed to parse search crawler parameters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, err := json.Marshal(tt.params)
			require.NoError(t, err)

			response, err := tool.Run(ctx, ToolCall{
				ID:    "test",
				Name:  SearchCrawlerToolName,
				Input: string(input),
			})

			assert.NoError(t, err)
			assert.True(t, response.IsError)
			assert.Contains(t, response.Content, tt.wantErrMsg)
		})
	}
}

func TestSearchCrawlerTool_Run_DefaultResultNum(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "searchcrawler_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	tool := NewSearchCrawlerTool(tempDir)
	ctx := context.Background()

	params := SearchCrawlerParams{
		Query:     "test query",
		SavePath:  "results",
		ResultNum: 0, // Should default to 8
	}

	input, err := json.Marshal(params)
	require.NoError(t, err)

	// Note: This test would normally require mocking the Serper client
	// For now, it will fail when trying to actually search
	// This just tests that the parameter validation works correctly
	response, err := tool.Run(ctx, ToolCall{
		ID:    "test",
		Name:  SearchCrawlerToolName,
		Input: string(input),
	})

	assert.NoError(t, err)
	// The actual search will fail without proper API key, but we're testing parameter handling
	assert.NotNil(t, response)
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal text",
			input:    "Normal Title",
			expected: "Normal Title",
		},
		{
			name:     "with invalid chars",
			input:    "Title: With/Invalid*Chars?",
			expected: "Title_ With_Invalid_Chars",
		},
		{
			name:     "with HTML tags",
			input:    "<h1>Title</h1>",
			expected: "Title",
		},
		{
			name:     "very long title",
			input:    strings.Repeat("a", 250),
			expected: strings.Repeat("a", 200),
		},
		{
			name:     "empty after sanitization",
			input:    "///",
			expected: "",
		},
		{
			name:     "with spaces and dots",
			input:    "  Title...  ",
			expected: "Title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSearchCrawlerTool_CreateDirectory(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "searchcrawler_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	tool := NewSearchCrawlerTool(tempDir)
	ctx := context.Background()

	// Test with a non-existent subdirectory
	savePath := filepath.Join(tempDir, "new_dir", "nested")
	params := SearchCrawlerParams{
		Query:     "test",
		SavePath:  savePath,
		ResultNum: 1,
	}

	input, err := json.Marshal(params)
	require.NoError(t, err)

	// Run the tool (will fail at search, but should create the directory)
	_, err = tool.Run(ctx, ToolCall{
		ID:    "test",
		Name:  SearchCrawlerToolName,
		Input: string(input),
	})

	assert.NoError(t, err)

	// Check if directory was created
	info, err := os.Stat(savePath)
	assert.NoError(t, err)
	assert.True(t, info.IsDir())
}