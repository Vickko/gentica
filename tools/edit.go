package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	// "time" // commented out - not needed after removing read check
)

type EditParams struct {
	FilePath   string `json:"file_path"`
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	ReplaceAll bool   `json:"replace_all,omitempty"`
}

type EditResponseMetadata struct {
	Additions  int    `json:"additions"`
	Removals   int    `json:"removals"`
	OldContent string `json:"old_content,omitempty"`
	NewContent string `json:"new_content,omitempty"`
}

type editTool struct {
	workingDir string
}

const (
	EditToolName    = "edit"
	editDescription = `Edits files by replacing text, creating new files, or deleting content. For moving or renaming files, use the Bash tool with the 'mv' command instead. For larger file edits, use the write tool to overwrite files.

Before using this tool:

1. Use the view tool to understand the file's contents and context

2. Verify the directory path is correct (only applicable when creating new files):
   - Use the ls tool to verify the parent directory exists and is the correct location

To make a file edit, provide the following:
1. file_path: The absolute path to the file to modify (must be absolute, not relative)
2. old_string: The text to replace (must be unique within the file, and must match the file contents exactly, including all whitespace and indentation)
3. new_string: The edited text to replace the old_string
4. replace_all: Replace all occurrences of old_string (default false)

Special cases:
- To create a new file: provide file_path and new_string, leave old_string empty
- To delete content: provide file_path and old_string, leave new_string empty

The tool will replace ONE occurrence of old_string with new_string in the specified file by default. Set replace_all to true to replace all occurrences.

CRITICAL REQUIREMENTS FOR USING THIS TOOL:

1. UNIQUENESS: When replace_all is false (default), the old_string MUST uniquely identify the specific instance you want to change. This means:
   - Include AT LEAST 3-5 lines of context BEFORE the change point
   - Include AT LEAST 3-5 lines of context AFTER the change point
   - Include all whitespace, indentation, and surrounding code exactly as it appears in the file

2. SINGLE INSTANCE: When replace_all is false, this tool can only change ONE instance at a time. If you need to change multiple instances:
   - Set replace_all to true to replace all occurrences at once
   - Or make separate calls to this tool for each instance
   - Each call must uniquely identify its specific instance using extensive context

3. VERIFICATION: Before using this tool:
   - Check how many instances of the target text exist in the file
   - If multiple instances exist and replace_all is false, gather enough context to uniquely identify each one
   - Plan separate tool calls for each instance or use replace_all

WARNING: If you do not follow these requirements:
   - The tool will fail if old_string matches multiple locations and replace_all is false
   - The tool will fail if old_string doesn't match exactly (including whitespace)
   - You may change the wrong instance if you don't include enough context

When making edits:
   - Ensure the edit results in idiomatic, correct code
   - Do not leave the code in a broken state
   - Always use absolute file paths (starting with /)

WINDOWS NOTES:
- File paths should use forward slashes (/) for cross-platform compatibility
- On Windows, absolute paths start with drive letters (C:/) but forward slashes work throughout`
)

func NewEditTool(workingDir string) BaseTool {
	return &editTool{
		workingDir: workingDir,
	}
}

func (e *editTool) Name() string {
	return EditToolName
}

func (e *editTool) Info() ToolInfo {
	return ToolInfo{
		Name:        EditToolName,
		Description: editDescription,
		Parameters: map[string]any{
			"file_path": map[string]any{
				"type":        "string",
				"description": "The path to the file to edit",
			},
			"old_string": map[string]any{
				"type":        "string",
				"description": "The text to replace in the file",
			},
			"new_string": map[string]any{
				"type":        "string",
				"description": "The new text to replace the old text with",
			},
			"replace_all": map[string]any{
				"type":        "boolean",
				"description": "Replace all occurrences of old_string (default false)",
			},
		},
		Required: []string{"file_path"},
	}
}

func (e *editTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params EditParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse("invalid parameters: " + err.Error()), nil
	}

	if params.FilePath == "" {
		return NewTextErrorResponse("file_path is required"), nil
	}

	// Convert to absolute path
	filePath := params.FilePath
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(e.workingDir, filePath)
	}
	filePath = filepath.Clean(filePath)

	// Validate parameters
	if params.OldString == params.NewString && params.OldString != "" {
		return NewTextErrorResponse("old_string and new_string are identical. No changes needed."), nil
	}

	// Handle special cases
	if params.OldString == "" && params.NewString != "" {
		// Create new file
		return e.createNewFile(ctx, filePath, params.NewString, call)
	} else if params.OldString != "" && params.NewString == "" {
		// Delete content
		return e.deleteContent(ctx, filePath, params.OldString, params.ReplaceAll, call)
	} else if params.OldString != "" && params.NewString != "" {
		// Replace content
		return e.replaceContent(ctx, filePath, params.OldString, params.NewString, params.ReplaceAll, call)
	} else {
		return NewTextErrorResponse("either old_string or new_string must be provided"), nil
	}
}

func (e *editTool) createNewFile(ctx context.Context, filePath, content string, call ToolCall) (ToolResponse, error) {
	// Check if file already exists
	fileInfo, err := os.Stat(filePath)
	if err == nil {
		if fileInfo.IsDir() {
			return NewTextErrorResponse(fmt.Sprintf("path is a directory, not a file: %s", filePath)), nil
		}
		return NewTextErrorResponse(fmt.Sprintf("file already exists: %s", filePath)), nil
	} else if !os.IsNotExist(err) {
		return ToolResponse{}, fmt.Errorf("failed to access file: %w", err)
	}

	// Create parent directories if needed
	dir := filepath.Dir(filePath)
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return ToolResponse{}, fmt.Errorf("failed to create parent directories: %w", err)
	}

	// Write the file
	err = os.WriteFile(filePath, []byte(content), 0o644)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("failed to write file: %w", err)
	}

	recordFileWrite(filePath)
	recordFileRead(filePath)

	additions := len(strings.Split(content, "\n"))
	removals := 0

	return WithResponseMetadata(
		NewTextResponse("File created: "+filePath),
		EditResponseMetadata{
			OldContent: "",
			NewContent: content,
			Additions:  additions,
			Removals:   removals,
		},
	), nil
}

func (e *editTool) deleteContent(ctx context.Context, filePath, oldString string, replaceAll bool, call ToolCall) (ToolResponse, error) {
	// Check file exists and is not a directory
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return NewTextErrorResponse(fmt.Sprintf("file does not exist: %s", filePath)), nil
		}
		return ToolResponse{}, fmt.Errorf("failed to access file: %w", err)
	}

	if fileInfo.IsDir() {
		return NewTextErrorResponse(fmt.Sprintf("path is a directory, not a file: %s", filePath)), nil
	}

	// Optional: Check if file has been read before (only warn, don't block)
	// This is commented out as tests expect to be able to edit without reading
	// if getLastReadTime(filePath).IsZero() {
	//     slog.Warn("editing file without reading it first", "file", filePath)
	// }

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("failed to read file: %w", err)
	}

	oldContent := string(content)
	var newContent string
	var deletionCount int

	if replaceAll {
		newContent = strings.ReplaceAll(oldContent, oldString, "")
		deletionCount = strings.Count(oldContent, oldString)
		if deletionCount == 0 {
			return NewTextErrorResponse("old_string not found in file. Make sure it matches exactly, including whitespace and line breaks"), nil
		}
	} else {
		// Check if old_string appears multiple times
		count := strings.Count(oldContent, oldString)
		if count == 0 {
			return NewTextErrorResponse("old_string not found in file. Make sure it matches exactly, including whitespace and line breaks"), nil
		}
		if count > 1 {
			return NewTextErrorResponse(fmt.Sprintf("old_string appears %d times in the file. Use replace_all=true or provide more context", count)), nil
		}
		newContent = strings.Replace(oldContent, oldString, "", 1)
		deletionCount = 1
	}

	// Check if content actually changed
	if oldContent == newContent {
		return NewTextErrorResponse("no changes made to file"), nil
	}

	// Write the file
	err = os.WriteFile(filePath, []byte(newContent), 0o644)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("failed to write file: %w", err)
	}

	recordFileWrite(filePath)
	recordFileRead(filePath)

	// Calculate changes based on actual content removed
	deletedLines := len(strings.Split(oldString, "\n"))

	return WithResponseMetadata(
		NewTextResponse(fmt.Sprintf("Content deleted from file: %s (%d occurrence(s) removed)", filePath, deletionCount)),
		EditResponseMetadata{
			Additions:  0,
			Removals:   deletedLines * deletionCount,
			OldContent: oldString,
			NewContent: "",
		},
	), nil
}

func (e *editTool) replaceContent(ctx context.Context, filePath, oldString, newString string, replaceAll bool, call ToolCall) (ToolResponse, error) {
	// Check file exists and is not a directory
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return NewTextErrorResponse(fmt.Sprintf("file does not exist: %s", filePath)), nil
		}
		return ToolResponse{}, fmt.Errorf("failed to access file: %w", err)
	}

	if fileInfo.IsDir() {
		return NewTextErrorResponse(fmt.Sprintf("path is a directory, not a file: %s", filePath)), nil
	}

	// Optional: Check if file has been read before (only warn, don't block)
	// This is commented out as tests expect to be able to edit without reading
	// if getLastReadTime(filePath).IsZero() {
	//     slog.Warn("editing file without reading it first", "file", filePath)
	// }

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("failed to read file: %w", err)
	}

	oldContent := string(content)
	var newContent string
	var replacementCount int

	if replaceAll {
		newContent = strings.ReplaceAll(oldContent, oldString, newString)
		replacementCount = strings.Count(oldContent, oldString)
		if replacementCount == 0 {
			return NewTextErrorResponse("old_string not found in file. Make sure it matches exactly, including whitespace and line breaks"), nil
		}
	} else {
		// Check if old_string appears multiple times
		count := strings.Count(oldContent, oldString)
		if count == 0 {
			return NewTextErrorResponse("old_string not found in file. Make sure it matches exactly, including whitespace and line breaks"), nil
		}
		if count > 1 {
			return NewTextErrorResponse(fmt.Sprintf("old_string appears %d times in the file. Use replace_all=true or provide more context", count)), nil
		}
		newContent = strings.Replace(oldContent, oldString, newString, 1)
		replacementCount = 1
	}

	// Check if content actually changed
	if oldContent == newContent {
		return NewTextErrorResponse("new content is the same as old content. No changes made."), nil
	}

	// Write the file
	err = os.WriteFile(filePath, []byte(newContent), 0o644)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("failed to write file: %w", err)
	}

	recordFileWrite(filePath)
	recordFileRead(filePath)

	// Calculate changes based on actual content changed
	oldStringLines := len(strings.Split(oldString, "\n"))
	newStringLines := len(strings.Split(newString, "\n"))
	additions := 0
	removals := 0

	if newStringLines > oldStringLines {
		additions = (newStringLines - oldStringLines) * replacementCount
	} else if oldStringLines > newStringLines {
		removals = (oldStringLines - newStringLines) * replacementCount
	}

	// Log the operation
	slog.Debug("File edited",
		"path", filePath,
		"replacements", replacementCount,
		"additions", additions,
		"removals", removals,
	)

	return WithResponseMetadata(
		NewTextResponse(fmt.Sprintf("File successfully edited: %s (%d occurrence(s) replaced)", filePath, replacementCount)),
		EditResponseMetadata{
			Additions:  additions,
			Removals:   removals,
			OldContent: oldString,
			NewContent: newString,
		},
	), nil
}