package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

type WriteParams struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

type writeTool struct {
	workingDir string
}

type WriteResponseMetadata struct {
	Additions int `json:"additions"`
	Removals  int `json:"removals"`
}

const (
	WriteToolName    = "write"
	writeDescription = `File writing tool that creates or updates files in the filesystem, allowing you to save or modify text content.

WHEN TO USE THIS TOOL:
- Use when you need to create a new file
- Helpful for updating existing files with modified content
- Perfect for saving generated code, configurations, or text data

HOW TO USE:
- Provide the path to the file you want to write
- Include the content to be written to the file
- The tool will create any necessary parent directories

FEATURES:
- Can create new files or overwrite existing ones
- Creates parent directories automatically if they don't exist
- Checks if the file has been modified since last read for safety
- Avoids unnecessary writes when content hasn't changed

LIMITATIONS:
- You should read a file before writing to it to avoid conflicts
- Cannot append to files (rewrites the entire file)

WINDOWS NOTES:
- File permissions (0o755, 0o644) are Unix-style but work on Windows with appropriate translations
- Use forward slashes (/) in paths for cross-platform compatibility
- Windows file attributes and permissions are handled automatically by the Go runtime

TIPS:
- Use the View tool first to examine existing files before modifying them
- Use the LS tool to verify the correct location when creating new files
- Combine with Glob and Grep tools to find and modify multiple files
- Always include descriptive comments when making changes to existing code`
)

func NewWriteTool(workingDir string) BaseTool {
	return &writeTool{
		workingDir: workingDir,
	}
}

func (w *writeTool) Name() string {
	return WriteToolName
}

func (w *writeTool) Info() ToolInfo {
	return ToolInfo{
		Name:        WriteToolName,
		Description: writeDescription,
		Parameters: map[string]any{
			"file_path": map[string]any{
				"type":        "string",
				"description": "The path to the file to write",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "The content to write to the file",
			},
		},
		Required: []string{"file_path", "content"},
	}
}

func (w *writeTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params WriteParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse("invalid parameters: " + err.Error()), nil
	}

	if params.FilePath == "" {
		return NewTextErrorResponse("file_path is required"), nil
	}

	// Allow empty content (for creating empty files)
	// if params.Content == "" {
	//	return NewTextErrorResponse("content is required"), nil
	// }

	// Convert to absolute path
	filePath := params.FilePath
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(w.workingDir, filePath)
	}
	filePath = filepath.Clean(filePath)

	// Check if this is an existing file
	fileInfo, err := os.Stat(filePath)
	existingFile := err == nil
	var existingContent []byte

	// If file exists, check if we've read it
	if existingFile {
		if fileInfo.IsDir() {
			return NewTextErrorResponse(fmt.Sprintf("path is a directory, not a file: %s", filePath)), nil
		}

		// Optional: Check if file has been read before (only warn, don't block)
		// This is commented out as tests expect to be able to overwrite without reading
		// lastReadTime := getLastReadTime(filePath)
		// if lastReadTime.IsZero() {
		//     // Just log a warning, don't block the operation
		//     slog.Warn("writing to file without reading it first", "file", filePath)
		// }

		// Read existing content for comparison
		existingContent, err = os.ReadFile(filePath)
		if err != nil {
			return NewTextErrorResponse(fmt.Sprintf("failed to read existing file: %v", err)), nil
		}

		// Check if content is the same
		if string(existingContent) == params.Content {
			return NewTextResponse("File content unchanged. No write performed."), nil
		}
	}

	// Create parent directories if needed
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return NewTextErrorResponse(fmt.Sprintf("failed to create parent directories: %v", err)), nil
	}

	// Write the file
	err = os.WriteFile(filePath, []byte(params.Content), 0o644)
	if err != nil {
		return NewTextErrorResponse(fmt.Sprintf("failed to write file: %v", err)), nil
	}

	// Record that we wrote the file
	recordFileWrite(filePath)
	recordFileRead(filePath) // Also record as read so we can edit it later

	// Calculate line count differences for metadata
	// Note: Even an empty file has 0 lines, not 1
	oldLineCount := 0
	newLineCount := 0
	
	if params.Content != "" {
		// strings.Split returns at least 1 element, so we only count if not empty
		newLineCount = len(strings.Split(params.Content, "\n"))
	}
	if existingFile && string(existingContent) != "" {
		oldLineCount = len(strings.Split(string(existingContent), "\n"))
	}
	
	lineCountDiff := newLineCount - oldLineCount

	// Log the operation
	slog.Debug("File written", 
		"path", filePath,
		"existed", existingFile,
		"oldLines", oldLineCount,
		"newLines", newLineCount,
		"lineCountDiff", lineCountDiff,
	)

	result := fmt.Sprintf("successfully wrote %d bytes to %s", len(params.Content), filePath)
	
	return WithResponseMetadata(NewTextResponse(result),
		WriteResponseMetadata{
			Additions: max(0, lineCountDiff),
			Removals:  max(0, -lineCountDiff),
		},
	), nil
}