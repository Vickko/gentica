package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLsTool(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	lsTool := NewLsTool(tempDir)

	// Create test file structure
	testStructure := map[string]bool{ // true for directories, false for files
		"file1.txt":                 false,
		"file2.txt":                 false,
		"src":                       true,
		"src/index.js":              false,
		"src/app.js":                false,
		"src/components":            true,
		"src/components/Button.jsx": false,
		"src/components/Form.jsx":   false,
		"test":                      true,
		"test/test1.spec.js":        false,
		"docs":                      true,
		"docs/README.md":            false,
		".hidden":                   true,
		".hidden/secret.txt":        false,
		"__pycache__":               true,
		"__pycache__/cache.pyc":     false,
	}

	for path, isDir := range testStructure {
		fullPath := filepath.Join(tempDir, path)
		if isDir {
			err := os.MkdirAll(fullPath, 0o755)
			require.NoError(t, err)
		} else {
			err := os.MkdirAll(filepath.Dir(fullPath), 0o755)
			require.NoError(t, err)
			err = os.WriteFile(fullPath, []byte("test content"), 0o644)
			require.NoError(t, err)
		}
	}

	t.Run("list root directory", func(t *testing.T) {
		params := LSParams{
			Path: tempDir,
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := lsTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)
		
		// Should show visible files and directories
		require.Contains(t, response.Content, "file1.txt")
		require.Contains(t, response.Content, "file2.txt")
		require.Contains(t, response.Content, "src/")
		require.Contains(t, response.Content, "test/")
		require.Contains(t, response.Content, "docs/")
		
		// Should not show hidden or system directories
		require.NotContains(t, response.Content, ".hidden")
		require.NotContains(t, response.Content, "__pycache__")
	})

	t.Run("list subdirectory", func(t *testing.T) {
		params := LSParams{
			Path: filepath.Join(tempDir, "src"),
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := lsTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)
		
		require.Contains(t, response.Content, "index.js")
		require.Contains(t, response.Content, "app.js")
		require.Contains(t, response.Content, "components/")
		require.Contains(t, response.Content, "Button.jsx")
		require.Contains(t, response.Content, "Form.jsx")
	})

	t.Run("default path", func(t *testing.T) {
		params := LSParams{}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := lsTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)
		// Should use working directory as default
		require.Contains(t, response.Content, "file1.txt")
	})

	t.Run("with ignore patterns", func(t *testing.T) {
		params := LSParams{
			Path:   tempDir,
			Ignore: []string{"*.txt", "test/**"},
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := lsTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)
		
		// Should not show ignored files
		require.NotContains(t, response.Content, "file1.txt")
		require.NotContains(t, response.Content, "file2.txt")
		require.NotContains(t, response.Content, "test/")
		
		// Should still show non-ignored items
		require.Contains(t, response.Content, "src/")
		require.Contains(t, response.Content, "docs/")
	})

	t.Run("non-existent directory", func(t *testing.T) {
		params := LSParams{
			Path: filepath.Join(tempDir, "nonexistent"),
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := lsTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "does not exist")
	})

	t.Run("list file instead of directory", func(t *testing.T) {
		params := LSParams{
			Path: filepath.Join(tempDir, "file1.txt"),
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := lsTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "not a directory")
	})

	t.Run("metadata includes file count", func(t *testing.T) {
		params := LSParams{
			Path: tempDir,
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := lsTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)
		
		// Check metadata
		if response.Metadata != "" {
			var metadata LSResponseMetadata
			err = json.Unmarshal([]byte(response.Metadata), &metadata)
			require.NoError(t, err)
			require.Greater(t, metadata.NumberOfFiles, 0)
		}
	})

	t.Run("invalid parameters", func(t *testing.T) {
		call := ToolCall{Input: "invalid json"}
		response, err := lsTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "invalid parameters")
	})

	t.Run("empty directory", func(t *testing.T) {
		emptyDir := filepath.Join(tempDir, "empty")
		err := os.MkdirAll(emptyDir, 0o755)
		require.NoError(t, err)

		params := LSParams{
			Path: emptyDir,
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := lsTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)
		require.Contains(t, response.Content, "(empty)")
	})
}