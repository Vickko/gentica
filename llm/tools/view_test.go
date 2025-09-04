package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestViewTool(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	viewTool := NewViewTool(tempDir)

	t.Run("read entire file", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "test.txt")
		content := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"
		err := os.WriteFile(filePath, []byte(content), 0o644)
		require.NoError(t, err)

		params := ViewParams{
			FilePath: filePath,
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := viewTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)
		
		// Should contain all lines with line numbers
		require.Contains(t, response.Content, "1→Line 1")
		require.Contains(t, response.Content, "2→Line 2")
		require.Contains(t, response.Content, "3→Line 3")
		require.Contains(t, response.Content, "4→Line 4")
		require.Contains(t, response.Content, "5→Line 5")
	})

	t.Run("read with offset", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "offset.txt")
		content := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"
		err := os.WriteFile(filePath, []byte(content), 0o644)
		require.NoError(t, err)

		params := ViewParams{
			FilePath: filePath,
			Offset:   2, // Start from line 3 (0-based)
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := viewTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)
		
		// Should not contain first two lines
		require.NotContains(t, response.Content, "Line 1")
		require.NotContains(t, response.Content, "Line 2")
		// Should contain lines from offset
		require.Contains(t, response.Content, "3→Line 3")
		require.Contains(t, response.Content, "4→Line 4")
		require.Contains(t, response.Content, "5→Line 5")
	})

	t.Run("read with limit", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "limit.txt")
		content := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"
		err := os.WriteFile(filePath, []byte(content), 0o644)
		require.NoError(t, err)

		params := ViewParams{
			FilePath: filePath,
			Limit:    3,
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := viewTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)
		
		// Should only contain first 3 lines
		require.Contains(t, response.Content, "1→Line 1")
		require.Contains(t, response.Content, "2→Line 2")
		require.Contains(t, response.Content, "3→Line 3")
		require.NotContains(t, response.Content, "Line 4")
		require.NotContains(t, response.Content, "Line 5")
	})

	t.Run("read with offset and limit", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "both.txt")
		content := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\nLine 6"
		err := os.WriteFile(filePath, []byte(content), 0o644)
		require.NoError(t, err)

		params := ViewParams{
			FilePath: filePath,
			Offset:   1,
			Limit:    3,
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := viewTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)
		
		// Should only contain lines 2-4
		require.NotContains(t, response.Content, "Line 1")
		require.Contains(t, response.Content, "2→Line 2")
		require.Contains(t, response.Content, "3→Line 3")
		require.Contains(t, response.Content, "4→Line 4")
		require.NotContains(t, response.Content, "Line 5")
		require.NotContains(t, response.Content, "Line 6")
	})

	t.Run("truncate long lines", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "long.txt")
		longLine := strings.Repeat("x", 3000)
		err := os.WriteFile(filePath, []byte(longLine), 0o644)
		require.NoError(t, err)

		params := ViewParams{
			FilePath: filePath,
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := viewTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)
		
		// Line should be truncated
		require.Contains(t, response.Content, "... (truncated)")
		require.True(t, len(response.Content) < 3000)
	})

	t.Run("empty file", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "empty.txt")
		err := os.WriteFile(filePath, []byte(""), 0o644)
		require.NoError(t, err)

		params := ViewParams{
			FilePath: filePath,
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := viewTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)
		require.Contains(t, response.Content, "(empty file)")
	})

	t.Run("file not found", func(t *testing.T) {
		params := ViewParams{
			FilePath: filepath.Join(tempDir, "nonexistent.txt"),
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := viewTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "no such file or directory")
	})

	t.Run("directory instead of file", func(t *testing.T) {
		dirPath := filepath.Join(tempDir, "testdir")
		err := os.MkdirAll(dirPath, 0o755)
		require.NoError(t, err)

		params := ViewParams{
			FilePath: dirPath,
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := viewTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "is a directory")
	})

	t.Run("relative path", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "relative.txt")
		err := os.WriteFile(filePath, []byte("relative content"), 0o644)
		require.NoError(t, err)

		params := ViewParams{
			FilePath: "relative.txt",
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := viewTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)
		require.Contains(t, response.Content, "relative content")
	})

	t.Run("missing file_path", func(t *testing.T) {
		params := ViewParams{}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := viewTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "file_path is required")
	})

	t.Run("invalid parameters", func(t *testing.T) {
		call := ToolCall{Input: "invalid json"}
		response, err := viewTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "invalid parameters")
	})
}