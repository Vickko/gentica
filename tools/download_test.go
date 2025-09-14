package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDownloadTool(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	downloadTool := NewDownloadTool(tempDir)

	t.Run("successful download", func(t *testing.T) {
		// Create test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "test content")
		}))
		defer server.Close()

		outputPath := filepath.Join(tempDir, "downloaded.txt")
		params := DownloadParams{
			URL:      server.URL,
			FilePath: outputPath,
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := downloadTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)
		require.Contains(t, response.Content, "Successfully downloaded")
		require.Contains(t, response.Content, "12 bytes")

		// Verify file contents
		content, err := os.ReadFile(outputPath)
		require.NoError(t, err)
		require.Equal(t, "test content", string(content))
	})

	t.Run("relative file path", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "relative path test")
		}))
		defer server.Close()

		params := DownloadParams{
			URL:      server.URL,
			FilePath: "relative/path/file.txt",
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := downloadTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)

		// Verify file was created in the correct location
		expectedPath := filepath.Join(tempDir, "relative/path/file.txt")
		_, err = os.Stat(expectedPath)
		require.NoError(t, err)
	})

	t.Run("404 not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		params := DownloadParams{
			URL:      server.URL,
			FilePath: filepath.Join(tempDir, "notfound.txt"),
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := downloadTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "status code: 404")
	})

	t.Run("file too large", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set content length header to exceed limit
			w.Header().Set("Content-Length", fmt.Sprintf("%d", 101*1024*1024))
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		params := DownloadParams{
			URL:      server.URL,
			FilePath: filepath.Join(tempDir, "large.txt"),
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := downloadTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "File too large")
	})

	t.Run("invalid URL scheme", func(t *testing.T) {
		params := DownloadParams{
			URL:      "ftp://example.com/file.txt",
			FilePath: filepath.Join(tempDir, "file.txt"),
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := downloadTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "must start with http:// or https://")
	})

	t.Run("missing URL", func(t *testing.T) {
		params := DownloadParams{
			FilePath: filepath.Join(tempDir, "file.txt"),
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := downloadTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "URL parameter is required")
	})

	t.Run("missing file path", func(t *testing.T) {
		params := DownloadParams{
			URL: "https://example.com/file.txt",
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := downloadTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "file_path parameter is required")
	})

	t.Run("invalid parameters", func(t *testing.T) {
		call := ToolCall{Input: "invalid json"}
		response, err := downloadTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "Failed to parse download parameters")
	})
}