package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteTool(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	writeTool := NewWriteTool(tempDir)

	t.Run("create new file", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "new.txt")
		content := "Hello, World!\nThis is a test file."

		params := WriteParams{
			FilePath: filePath,
			Content:  content,
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := writeTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)
		require.Contains(t, response.Content, "successfully wrote")

		// Verify file contents
		readContent, err := os.ReadFile(filePath)
		require.NoError(t, err)
		require.Equal(t, content, string(readContent))
	})

	t.Run("overwrite existing file", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "existing.txt")
		originalContent := "Original content"
		err := os.WriteFile(filePath, []byte(originalContent), 0o644)
		require.NoError(t, err)

		newContent := "New content"
		params := WriteParams{
			FilePath: filePath,
			Content:  newContent,
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := writeTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)

		// Verify file was overwritten
		readContent, err := os.ReadFile(filePath)
		require.NoError(t, err)
		require.Equal(t, newContent, string(readContent))
	})

	t.Run("create file in nested directory", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "nested/dir/file.txt")
		content := "Nested file content"

		params := WriteParams{
			FilePath: filePath,
			Content:  content,
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := writeTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)

		// Verify file was created with parent directories
		readContent, err := os.ReadFile(filePath)
		require.NoError(t, err)
		require.Equal(t, content, string(readContent))
	})

	t.Run("write empty file", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "empty.txt")
		
		params := WriteParams{
			FilePath: filePath,
			Content:  "",
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := writeTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)

		// Verify empty file was created
		stat, err := os.Stat(filePath)
		require.NoError(t, err)
		require.Equal(t, int64(0), stat.Size())
	})

	t.Run("relative path", func(t *testing.T) {
		params := WriteParams{
			FilePath: "relative.txt",
			Content:  "Relative path content",
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := writeTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)

		// Verify file was created in working directory
		expectedPath := filepath.Join(tempDir, "relative.txt")
		readContent, err := os.ReadFile(expectedPath)
		require.NoError(t, err)
		require.Equal(t, "Relative path content", string(readContent))
	})

	t.Run("unchanged content", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "unchanged.txt")
		content := "Same content"
		
		// Write initial content
		err := os.WriteFile(filePath, []byte(content), 0o644)
		require.NoError(t, err)

		// Try to write same content
		params := WriteParams{
			FilePath: filePath,
			Content:  content,
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := writeTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)
		require.Contains(t, response.Content, "unchanged")
	})

	t.Run("write to directory", func(t *testing.T) {
		dirPath := filepath.Join(tempDir, "testdir")
		err := os.MkdirAll(dirPath, 0o755)
		require.NoError(t, err)

		params := WriteParams{
			FilePath: dirPath,
			Content:  "content",
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := writeTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "is a directory")
	})

	t.Run("missing file_path", func(t *testing.T) {
		params := WriteParams{
			Content: "content",
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := writeTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "file_path is required")
	})

	t.Run("invalid parameters", func(t *testing.T) {
		call := ToolCall{Input: "invalid json"}
		response, err := writeTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "invalid parameters")
	})

	t.Run("metadata includes diff info", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "diff_test.txt")
		originalContent := "line1\nline2\nline3"
		err := os.WriteFile(filePath, []byte(originalContent), 0o644)
		require.NoError(t, err)

		newContent := "line1\nmodified\nline3\nline4"
		params := WriteParams{
			FilePath: filePath,
			Content:  newContent,
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := writeTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)

		// Check metadata contains diff information
		if response.Metadata != "" {
			var metadata WriteResponseMetadata
			err = json.Unmarshal([]byte(response.Metadata), &metadata)
			require.NoError(t, err)
			require.Greater(t, metadata.Additions, 0)
		}
	})
}