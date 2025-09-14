package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFetchTool(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	fetchTool := NewFetchTool(tempDir)

	t.Run("fetch text format", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprint(w, "Hello, World!")
		}))
		defer server.Close()

		params := FetchParams{
			URL:    server.URL,
			Format: "text",
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := fetchTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)
		require.Contains(t, response.Content, "Hello, World!")
	})

	t.Run("fetch raw format", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"key": "value"}`)
		}))
		defer server.Close()

		params := FetchParams{
			URL:    server.URL,
			Format: "raw",
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := fetchTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)
		require.Contains(t, response.Content, `{"key": "value"}`)
	})

	t.Run("404 not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, "Not Found")
		}))
		defer server.Close()

		params := FetchParams{
			URL:    server.URL,
			Format: "text",
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := fetchTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "status code: 404")
	})

	t.Run("redirect handling", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/redirect" {
				http.Redirect(w, r, "/final", http.StatusFound)
			} else {
				fmt.Fprint(w, "Redirected content")
			}
		}))
		defer server.Close()

		params := FetchParams{
			URL:    server.URL + "/redirect",
			Format: "text",
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := fetchTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)
		require.Contains(t, response.Content, "Redirected content")
	})

	t.Run("invalid URL scheme", func(t *testing.T) {
		params := FetchParams{
			URL:    "ftp://example.com/file.txt",
			Format: "text",
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := fetchTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "must start with http:// or https://")
	})

	t.Run("missing URL", func(t *testing.T) {
		params := FetchParams{
			Format: "text",
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := fetchTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "URL parameter is required")
	})

	t.Run("missing format", func(t *testing.T) {
		params := FetchParams{
			URL: "https://example.com",
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := fetchTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "format parameter is required")
	})

	t.Run("invalid format", func(t *testing.T) {
		params := FetchParams{
			URL:    "https://example.com",
			Format: "invalid",
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := fetchTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "format must be 'text' or 'raw'")
	})

	t.Run("invalid parameters", func(t *testing.T) {
		call := ToolCall{Input: "invalid json"}
		response, err := fetchTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "Failed to parse fetch parameters")
	})

	t.Run("text format preserves indentation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			// Write code-like content with indentation
			fmt.Fprint(w, "function example() {\n    if (true) {\n        console.log('hello');\n    }\n}")
		}))
		defer server.Close()

		params := FetchParams{
			URL:    server.URL,
			Format: "text",
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := fetchTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)
		// Check that indentation is preserved
		require.Contains(t, response.Content, "    if (true)")
		require.Contains(t, response.Content, "        console.log")
	})

	t.Run("text format removes excessive blank lines", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			// Write content with many blank lines
			fmt.Fprint(w, "Line 1\n\n\n\n\nLine 2\n\n\n\n\n\n\nLine 3")
		}))
		defer server.Close()

		params := FetchParams{
			URL:    server.URL,
			Format: "text",
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := fetchTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)
		// Should limit consecutive blank lines
		require.Contains(t, response.Content, "Line 1")
		require.Contains(t, response.Content, "Line 2")
		require.Contains(t, response.Content, "Line 3")
		// Check that we don't have excessive newlines
		require.NotContains(t, response.Content, "\n\n\n\n")
	})

	t.Run("content truncation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			// Write large content (over 250KB)
			for i := 0; i < 300*1024; i++ {
				fmt.Fprint(w, "a")
			}
		}))
		defer server.Close()

		params := FetchParams{
			URL:    server.URL,
			Format: "raw",
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := fetchTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)
		// Should be truncated to 250KB
		require.Contains(t, response.Content, "[Content truncated to 256000 bytes]")
		require.LessOrEqual(t, len(response.Content), 256100) // 250KB + truncation message
	})
}