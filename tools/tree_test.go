package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTreeTool(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	treeTool := NewTreeTool(tempDir)

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
		params := TreeParams{
			Path: tempDir,
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := treeTool.Run(context.Background(), call)
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
		params := TreeParams{
			Path: filepath.Join(tempDir, "src"),
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := treeTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)
		
		require.Contains(t, response.Content, "index.js")
		require.Contains(t, response.Content, "app.js")
		require.Contains(t, response.Content, "components/")
		require.Contains(t, response.Content, "Button.jsx")
		require.Contains(t, response.Content, "Form.jsx")
	})

	t.Run("default path", func(t *testing.T) {
		params := TreeParams{}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := treeTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)
		// Should use working directory as default
		require.Contains(t, response.Content, "file1.txt")
	})

	t.Run("with ignore patterns", func(t *testing.T) {
		params := TreeParams{
			Path:   tempDir,
			Ignore: []string{"*.txt", "test/**"},
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := treeTool.Run(context.Background(), call)
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
		params := TreeParams{
			Path: filepath.Join(tempDir, "nonexistent"),
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := treeTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "does not exist")
	})

	t.Run("list file instead of directory", func(t *testing.T) {
		params := TreeParams{
			Path: filepath.Join(tempDir, "file1.txt"),
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := treeTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "not a directory")
	})

	t.Run("metadata includes file count", func(t *testing.T) {
		params := TreeParams{
			Path: tempDir,
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := treeTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)
		
		// Check metadata
		if response.Metadata != "" {
			var metadata TreeResponseMetadata
			err = json.Unmarshal([]byte(response.Metadata), &metadata)
			require.NoError(t, err)
			require.Greater(t, metadata.NumberOfFiles, 0)
		}
	})

	t.Run("invalid parameters", func(t *testing.T) {
		call := ToolCall{Input: "invalid json"}
		response, err := treeTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "invalid parameters")
	})

	t.Run("empty directory", func(t *testing.T) {
		emptyDir := filepath.Join(tempDir, "empty")
		err := os.MkdirAll(emptyDir, 0o755)
		require.NoError(t, err)

		params := TreeParams{
			Path: emptyDir,
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := treeTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)
		require.Contains(t, response.Content, "(empty)")
	})

	t.Run("max depth limit", func(t *testing.T) {
		// Create a deep directory structure
		deepDir := filepath.Join(tempDir, "deep")
		os.MkdirAll(filepath.Join(deepDir, "level1", "level2", "level3", "level4", "level5"), 0o755)
		os.WriteFile(filepath.Join(deepDir, "level1", "level2", "level3", "level4", "level5", "deep.txt"), []byte("deep file"), 0o644)

		maxDepth := 2
		params := TreeParams{
			Path:     deepDir,
			MaxDepth: &maxDepth,
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := treeTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)

		// Should see level1 and level2, but not deeper levels
		require.Contains(t, response.Content, "level1")
		require.Contains(t, response.Content, "level2")
		require.NotContains(t, response.Content, "level3")
		require.NotContains(t, response.Content, "deep.txt")
		require.Contains(t, response.Content, "...")
		// Should NOT have warning since depth was explicitly set
		require.NotContains(t, response.Content, "by default")

		// Check metadata
		if response.Metadata != "" {
			var metadata TreeResponseMetadata
			err = json.Unmarshal([]byte(response.Metadata), &metadata)
			require.NoError(t, err)
			require.True(t, metadata.DepthLimited)
		}
	})

	t.Run("unlimited depth with max_depth=0", func(t *testing.T) {
		// Create a deep directory structure
		deepDir := filepath.Join(tempDir, "unlimited")
		os.MkdirAll(filepath.Join(deepDir, "a", "b", "c", "d", "e"), 0o755)
		os.WriteFile(filepath.Join(deepDir, "a", "b", "c", "d", "e", "file.txt"), []byte("content"), 0o644)

		maxDepth := 0 // Explicitly set to 0 for unlimited
		params := TreeParams{
			Path:     deepDir,
			MaxDepth: &maxDepth,
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := treeTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)

		// Should see all levels
		require.Contains(t, response.Content, "a/")
		require.Contains(t, response.Content, "b/")
		require.Contains(t, response.Content, "c/")
		require.Contains(t, response.Content, "d/")
		require.Contains(t, response.Content, "e/")
		require.Contains(t, response.Content, "file.txt")
	})

	t.Run("dirs only mode", func(t *testing.T) {
		params := TreeParams{
			Path:     tempDir,
			DirsOnly: true,
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := treeTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)

		// Should show directories
		require.Contains(t, response.Content, "src/")
		require.Contains(t, response.Content, "test/")
		require.Contains(t, response.Content, "docs/")

		// Should not show files
		require.NotContains(t, response.Content, "file1.txt")
		require.NotContains(t, response.Content, "file2.txt")
		require.NotContains(t, response.Content, "index.js")
		require.NotContains(t, response.Content, "README.md")
	})

	t.Run("default max depth is 4", func(t *testing.T) {
		// Create directory structure deeper than 4 levels
		deepDefaultDir := filepath.Join(tempDir, "default_depth")
		os.MkdirAll(filepath.Join(deepDefaultDir, "l1", "l2", "l3", "l4", "l5", "l6"), 0o755)

		// Don't specify max_depth, should default to 4
		params := TreeParams{
			Path: deepDefaultDir,
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := treeTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)

		// Should see up to level 4
		require.Contains(t, response.Content, "l1/")
		require.Contains(t, response.Content, "l2/")
		require.Contains(t, response.Content, "l3/")
		require.Contains(t, response.Content, "l4/")
		// Should not see level 5 and beyond
		require.NotContains(t, response.Content, "l5/")
		require.NotContains(t, response.Content, "l6/")

		// Should have depth limit warning with "by default" text and suggestion
		require.Contains(t, response.Content, "Output is limited to 4 levels deep by default")
		require.Contains(t, response.Content, "Use max_depth=0 to see the full tree or specify a custom depth")
	})

	t.Run("explicit max depth no warning", func(t *testing.T) {
		// Create directory structure deeper than specified depth
		explicitDepthDir := filepath.Join(tempDir, "explicit_depth")
		os.MkdirAll(filepath.Join(explicitDepthDir, "a", "b", "c", "d", "e"), 0o755)

		// Explicitly set max_depth to 3
		maxDepth := 3
		params := TreeParams{
			Path:     explicitDepthDir,
			MaxDepth: &maxDepth,
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := treeTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)

		// Should see up to level 3
		require.Contains(t, response.Content, "a/")
		require.Contains(t, response.Content, "b/")
		require.Contains(t, response.Content, "c/")
		// Should not see level 4 and beyond
		require.NotContains(t, response.Content, "d/")
		require.NotContains(t, response.Content, "e/")

		// Should NOT have any depth limit warning since user explicitly set it
		require.NotContains(t, response.Content, "Output is limited")
		require.NotContains(t, response.Content, "by default")
	})
}