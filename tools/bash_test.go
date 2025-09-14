package tools

import (
	"context"
	"encoding/json"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBashTool(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	bashTool := NewBashTool(tempDir)

	t.Run("basic command execution", func(t *testing.T) {
		params := BashParams{
			Command: "echo 'hello world'",
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := bashTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)
		require.Contains(t, response.Content, "hello world")
	})

	t.Run("command with exit code", func(t *testing.T) {
		params := BashParams{
			Command: "exit 0",
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := bashTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)
	})

	t.Run("command with non-zero exit code", func(t *testing.T) {
		params := BashParams{
			Command: "exit 1",
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := bashTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "Exit code: 1")
	})

	t.Run("chained commands", func(t *testing.T) {
		params := BashParams{
			Command: "echo 'first' && echo 'second'",
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := bashTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)
		require.Contains(t, response.Content, "first")
		require.Contains(t, response.Content, "second")
	})

	t.Run("working directory", func(t *testing.T) {
		params := BashParams{
			Command: "pwd",
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := bashTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.False(t, response.IsError)
		require.Contains(t, response.Content, tempDir)
	})

	t.Run("banned command", func(t *testing.T) {
		params := BashParams{
			Command: "curl https://example.com",
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := bashTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "not allowed for security reasons")
	})

	t.Run("timeout", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("sleep command not available on Windows")
		}
		
		params := BashParams{
			Command: "sleep 10",
			Timeout: 100, // 100ms timeout
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := bashTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "Command timed out after")
	})

	t.Run("empty command", func(t *testing.T) {
		params := BashParams{
			Command: "",
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := ToolCall{Input: string(paramsJSON)}
		response, err := bashTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "command is required")
	})

	t.Run("invalid parameters", func(t *testing.T) {
		call := ToolCall{Input: "invalid json"}
		response, err := bashTool.Run(context.Background(), call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "invalid parameters")
	})
}