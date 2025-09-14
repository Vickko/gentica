package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEditTool(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	editTool := NewEditTool(tempDir)

	t.Run("create new file", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "new_file.txt")
		params := EditParams{
			FilePath:  filePath,
			OldString: "",
			NewString: "Hello, World!\nThis is a new file.",
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := editTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)
		require.Contains(t, response.Content, "File created")

		// Verify file contents
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		require.Equal(t, "Hello, World!\nThis is a new file.", string(content))
	})

	t.Run("replace content in existing file", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "existing.txt")
		initialContent := "Line 1\nLine 2\nLine 3\nLine 4"
		err := os.WriteFile(filePath, []byte(initialContent), 0o644)
		require.NoError(t, err)

		params := EditParams{
			FilePath:  filePath,
			OldString: "Line 2",
			NewString: "Modified Line",
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := editTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)
		require.Contains(t, response.Content, "successfully edited")

		// Verify file contents
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		require.Equal(t, "Line 1\nModified Line\nLine 3\nLine 4", string(content))
	})

	t.Run("replace all occurrences", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "multiple.txt")
		initialContent := "test\ntest\nother\ntest"
		err := os.WriteFile(filePath, []byte(initialContent), 0o644)
		require.NoError(t, err)

		params := EditParams{
			FilePath:   filePath,
			OldString:  "test",
			NewString:  "replaced",
			ReplaceAll: true,
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := editTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)

		// Verify all occurrences were replaced
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		require.Equal(t, "replaced\nreplaced\nother\nreplaced", string(content))
	})

	t.Run("delete content", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "delete_test.txt")
		initialContent := "Keep this\nDelete this line\nKeep this too"
		err := os.WriteFile(filePath, []byte(initialContent), 0o644)
		require.NoError(t, err)

		params := EditParams{
			FilePath:  filePath,
			OldString: "Delete this line\n",
			NewString: "",
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := editTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)

		// Verify content was deleted
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		require.Equal(t, "Keep this\nKeep this too", string(content))
	})

	t.Run("file already exists error", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "existing_error.txt")
		err := os.WriteFile(filePath, []byte("existing content"), 0o644)
		require.NoError(t, err)

		params := EditParams{
			FilePath:  filePath,
			OldString: "",
			NewString: "New content",
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := editTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "file already exists")
	})

	t.Run("file not found error", func(t *testing.T) {
		params := EditParams{
			FilePath:  filepath.Join(tempDir, "nonexistent.txt"),
			OldString: "something",
			NewString: "else",
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := editTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "file does not exist")
	})

	t.Run("string not found error", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "notfound.txt")
		err := os.WriteFile(filePath, []byte("some content"), 0o644)
		require.NoError(t, err)

		params := EditParams{
			FilePath:  filePath,
			OldString: "not in file",
			NewString: "replacement",
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := editTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "old_string not found")
	})

	t.Run("relative path conversion", func(t *testing.T) {
		params := EditParams{
			FilePath:  "relative/new_file.txt",
			OldString: "",
			NewString: "Content in relative path",
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := editTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)

		// Verify file was created at the correct absolute path
		expectedPath := filepath.Join(tempDir, "relative/new_file.txt")
		content, err := os.ReadFile(expectedPath)
		require.NoError(t, err)
		require.Equal(t, "Content in relative path", string(content))
	})

	t.Run("missing file_path", func(t *testing.T) {
		params := EditParams{
			OldString: "old",
			NewString: "new",
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := editTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "file_path is required")
	})

	t.Run("invalid parameters", func(t *testing.T) {
		call := ToolCall{Input: "invalid json"}
		response, err := editTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "invalid parameters")
	})

	t.Run("old_string equals new_string error", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "same_strings.txt")
		err := os.WriteFile(filePath, []byte("test content"), 0o644)
		require.NoError(t, err)

		params := EditParams{
			FilePath:  filePath,
			OldString: "test",
			NewString: "test",
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := editTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "identical")
	})
}