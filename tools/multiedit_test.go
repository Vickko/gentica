package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMultiEditTool(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	multiEditTool := NewMultiEditTool(tempDir)

	t.Run("multiple sequential edits", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "multi.txt")
		initialContent := "Line 1\nLine 2\nLine 3\nLine 4"
		err := os.WriteFile(filePath, []byte(initialContent), 0o644)
		require.NoError(t, err)

		params := MultiEditParams{
			FilePath: filePath,
			Edits: []MultiEditOperation{
				{OldString: "Line 1", NewString: "First Line"},
				{OldString: "Line 2", NewString: "Second Line"},
				{OldString: "Line 4", NewString: "Fourth Line"},
			},
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := multiEditTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)

		// Verify all edits were applied
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		require.Equal(t, "First Line\nSecond Line\nLine 3\nFourth Line", string(content))
	})

	t.Run("create new file with initial content", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "new_multi.txt")
		
		params := MultiEditParams{
			FilePath: filePath,
			Edits: []MultiEditOperation{
				{OldString: "", NewString: "Initial content\nWith multiple lines"},
			},
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := multiEditTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)

		// Verify file was created with content
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		require.Equal(t, "Initial content\nWith multiple lines", string(content))
	})

	t.Run("replace all occurrences", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "replace_all.txt")
		initialContent := "test test test\nother\ntest again"
		err := os.WriteFile(filePath, []byte(initialContent), 0o644)
		require.NoError(t, err)

		params := MultiEditParams{
			FilePath: filePath,
			Edits: []MultiEditOperation{
				{OldString: "test", NewString: "replaced", ReplaceAll: true},
				{OldString: "other", NewString: "changed"},
			},
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := multiEditTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)

		// Verify all edits were applied
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		require.Equal(t, "replaced replaced replaced\nchanged\nreplaced again", string(content))
	})

	t.Run("edits affect subsequent edits", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "sequential.txt")
		initialContent := "Hello World"
		err := os.WriteFile(filePath, []byte(initialContent), 0o644)
		require.NoError(t, err)

		params := MultiEditParams{
			FilePath: filePath,
			Edits: []MultiEditOperation{
				{OldString: "Hello", NewString: "Goodbye"},
				{OldString: "Goodbye World", NewString: "Goodbye Universe"},
			},
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := multiEditTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)

		// Verify sequential edits
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		require.Equal(t, "Goodbye Universe", string(content))
	})

	t.Run("one edit fails - all rollback", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "rollback.txt")
		initialContent := "Line 1\nLine 2\nLine 3"
		err := os.WriteFile(filePath, []byte(initialContent), 0o644)
		require.NoError(t, err)

		params := MultiEditParams{
			FilePath: filePath,
			Edits: []MultiEditOperation{
				{OldString: "Line 1", NewString: "First Line"},
				{OldString: "NonExistent", NewString: "Should Fail"},
				{OldString: "Line 3", NewString: "Third Line"},
			},
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := multiEditTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "string not found")

		// Verify file is unchanged (rollback)
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		require.Equal(t, initialContent, string(content))
	})

	t.Run("file not found", func(t *testing.T) {
		params := MultiEditParams{
			FilePath: filepath.Join(tempDir, "nonexistent.txt"),
			Edits: []MultiEditOperation{
				{OldString: "test", NewString: "replaced"},
			},
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := multiEditTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "File does not exist")
	})

	t.Run("empty edits array", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "empty_edits.txt")
		err := os.WriteFile(filePath, []byte("content"), 0o644)
		require.NoError(t, err)

		params := MultiEditParams{
			FilePath: filePath,
			Edits:    []MultiEditOperation{},
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := multiEditTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "at least one edit is required")
	})

	t.Run("missing file_path", func(t *testing.T) {
		params := MultiEditParams{
			Edits: []MultiEditOperation{
				{OldString: "test", NewString: "replaced"},
			},
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := multiEditTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "file_path is required")
	})

	t.Run("both old_string and new_string empty", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "empty_strings.txt")
		err := os.WriteFile(filePath, []byte("content"), 0o644)
		require.NoError(t, err)

		params := MultiEditParams{
			FilePath: filePath,
			Edits: []MultiEditOperation{
				{OldString: "", NewString: ""},
			},
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := multiEditTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "both old_string and new_string cannot be empty")
	})

	t.Run("invalid parameters", func(t *testing.T) {
		call := ToolCall{Input: "invalid json"}
		response, err := multiEditTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "invalid parameters")
	})
}