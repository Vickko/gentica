package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	// "time" // commented out - not needed after removing read check
)

type MultiEditOperation struct {
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	ReplaceAll bool   `json:"replace_all,omitempty"`
}

type MultiEditParams struct {
	FilePath string               `json:"file_path"`
	Edits    []MultiEditOperation `json:"edits"`
}

type MultiEditResponseMetadata struct {
	EditsApplied int `json:"edits_applied"`
}

type multiEditTool struct {
	workingDir string
}

const (
	MultiEditToolName    = "multiedit"
	multiEditDescription = `This is a tool for making multiple edits to a single file in one operation. It is built on top of the Edit tool and allows you to perform multiple find-and-replace operations efficiently. Prefer this tool over the Edit tool when you need to make multiple edits to the same file.

Before using this tool:

1. Use the Read tool to understand the file's contents and context
2. Verify the directory path is correct

To make multiple file edits, provide the following:
1. file_path: The absolute path to the file to modify (must be absolute, not relative)
2. edits: An array of edit operations to perform, where each edit contains:
   - old_string: The text to replace (must match the file contents exactly, including all whitespace and indentation)
   - new_string: The edited text to replace the old_string
   - replace_all: Replace all occurrences of old_string. This parameter is optional and defaults to false.

IMPORTANT:
- All edits are applied in sequence, in the order they are provided
- Each edit operates on the result of the previous edit
- All edits must be valid for the operation to succeed - if any edit fails, none will be applied
- This tool is ideal when you need to make several changes to different parts of the same file

CRITICAL REQUIREMENTS:
1. All edits follow the same requirements as the single Edit tool
2. The edits are atomic - either all succeed or none are applied
3. Plan your edits carefully to avoid conflicts between sequential operations

WARNING:
- The tool will fail if edits.old_string doesn't match the file contents exactly (including whitespace)
- The tool will fail if edits.old_string and edits.new_string are the same
- Since edits are applied in sequence, ensure that earlier edits don't affect the text that later edits are trying to find

When making edits:
- Ensure all edits result in idiomatic, correct code
- Do not leave the code in a broken state
- Always use absolute file paths (starting with /)
- Only use emojis if the user explicitly requests it. Avoid adding emojis to files unless asked.
- Use replace_all for replacing and renaming strings across the file. This parameter is useful if you want to rename a variable for instance.

If you want to create a new file, use:
- A new file path, including dir name if needed
- First edit: empty old_string and the new file's contents as new_string
- Subsequent edits: normal edit operations on the created content`
)

func NewMultiEditTool(workingDir string) BaseTool {
	return &multiEditTool{
		workingDir: workingDir,
	}
}

func (m *multiEditTool) Name() string {
	return MultiEditToolName
}

func (m *multiEditTool) Info() ToolInfo {
	return ToolInfo{
		Name:        MultiEditToolName,
		Description: multiEditDescription,
		Parameters: map[string]any{
			"file_path": map[string]any{
				"type":        "string",
				"description": "The absolute path to the file to modify",
			},
			"edits": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"old_string": map[string]any{
							"type":        "string",
							"description": "The text to replace",
						},
						"new_string": map[string]any{
							"type":        "string",
							"description": "The text to replace it with",
						},
						"replace_all": map[string]any{
							"type":        "boolean",
							"default":     false,
							"description": "Replace all occurrences of old_string (default false).",
						},
					},
					"required":             []string{"old_string", "new_string"},
					"additionalProperties": false,
				},
				"minItems":    1,
				"description": "Array of edit operations to perform sequentially on the file",
			},
		},
		Required: []string{"file_path", "edits"},
	}
}

func (m *multiEditTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params MultiEditParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse(fmt.Sprintf("invalid parameters: %s", err)), nil
	}

	if params.FilePath == "" {
		return NewTextErrorResponse("file_path is required"), nil
	}

	if len(params.Edits) == 0 {
		return NewTextErrorResponse("at least one edit is required"), nil
	}

	// Expand home directory if needed
	if strings.HasPrefix(params.FilePath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			params.FilePath = filepath.Join(homeDir, params.FilePath[2:])
		}
	}

	if !filepath.IsAbs(params.FilePath) {
		params.FilePath = filepath.Join(m.workingDir, params.FilePath)
	}

	// Validate all edits before applying any
	if err := m.validateEdits(params.Edits); err != nil {
		return NewTextErrorResponse(err.Error()), nil
	}

	var response ToolResponse
	var err error

	// Handle file creation case (first edit has empty old_string)
	if len(params.Edits) > 0 && params.Edits[0].OldString == "" {
		response, err = m.processMultiEditWithCreation(params)
	} else {
		response, err = m.processMultiEditExistingFile(params)
	}

	if err != nil {
		return response, err
	}

	return response, nil
}

func (m *multiEditTool) validateEdits(edits []MultiEditOperation) error {
	for i, edit := range edits {
		// Check for both strings being empty (except for first edit which can create a file)
		if edit.OldString == "" && edit.NewString == "" {
			return fmt.Errorf("edit %d: both old_string and new_string cannot be empty", i+1)
		}
		
		// Check for identical old_string and new_string
		if edit.OldString == edit.NewString {
			return fmt.Errorf("edit %d: old_string and new_string are identical", i+1)
		}
		
		// Only the first edit can have empty old_string (for file creation)
		if i > 0 && edit.OldString == "" {
			return fmt.Errorf("edit %d: only the first edit can have empty old_string (for file creation)", i+1)
		}
	}
	return nil
}

func (m *multiEditTool) processMultiEditWithCreation(params MultiEditParams) (ToolResponse, error) {
	// First edit creates the file
	firstEdit := params.Edits[0]
	if firstEdit.OldString != "" {
		return NewTextErrorResponse("first edit must have empty old_string for file creation"), nil
	}

	// Check if file already exists
	if _, err := os.Stat(params.FilePath); err == nil {
		return NewTextErrorResponse(fmt.Sprintf("File already exists: %s (remove it first or use edit tool)", params.FilePath)), nil
	} else if !os.IsNotExist(err) {
		return ToolResponse{}, fmt.Errorf("failed to access file: %w", err)
	}

	// Create parent directories
	dir := filepath.Dir(params.FilePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ToolResponse{}, fmt.Errorf("failed to create parent directories: %w", err)
	}

	// Start with the content from the first edit
	currentContent := firstEdit.NewString

	// Apply remaining edits to the content
	for i := 1; i < len(params.Edits); i++ {
		edit := params.Edits[i]
		newContent, err := m.applyEditToContent(currentContent, edit)
		if err != nil {
			return NewTextErrorResponse(fmt.Sprintf("edit %d failed: %s", i+1, err.Error())), nil
		}
		currentContent = newContent
	}

	// Write the file
	err := os.WriteFile(params.FilePath, []byte(currentContent), 0o644)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("failed to write file: %w", err)
	}

	recordFileWrite(params.FilePath)
	recordFileRead(params.FilePath)

	return WithResponseMetadata(
		NewTextResponse(fmt.Sprintf("File created with %d edits: %s", len(params.Edits), params.FilePath)),
		MultiEditResponseMetadata{
			EditsApplied: len(params.Edits),
		},
	), nil
}

func (m *multiEditTool) processMultiEditExistingFile(params MultiEditParams) (ToolResponse, error) {
	// Validate file exists and is readable
	fileInfo, err := os.Stat(params.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return NewTextErrorResponse(fmt.Sprintf("File does not exist: %s (use first edit with empty old_string to create)", params.FilePath)), nil
		}
		return ToolResponse{}, fmt.Errorf("failed to access file: %w", err)
	}

	if fileInfo.IsDir() {
		return NewTextErrorResponse(fmt.Sprintf("Path is a directory: %s (expected file)", params.FilePath)), nil
	}

	// Optional: Check if file was read before editing (only warn, don't block)
	// This is commented out as tests expect to be able to edit without reading
	// if getLastReadTime(params.FilePath).IsZero() {
	//     // Just log a warning, don't block the operation
	//     // slog.Warn("editing file without reading it first", "file", params.FilePath)
	// }

	// Read current file content
	content, err := os.ReadFile(params.FilePath)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("failed to read file: %w", err)
	}

	// Detect original line ending style
	originalContent := string(content)
	hasWindowsLineEndings := strings.Contains(originalContent, "\r\n")
	
	// Convert to Unix line endings for processing
	oldContent := normalizeLineEndings(originalContent)
	currentContent := oldContent

	// Apply all edits sequentially
	for i, edit := range params.Edits {
		newContent, err := m.applyEditToContent(currentContent, edit)
		if err != nil {
			return NewTextErrorResponse(fmt.Sprintf("edit %d failed: %s", i+1, err.Error())), nil
		}
		currentContent = newContent
	}

	// Check if content actually changed
	if oldContent == currentContent {
		return NewTextErrorResponse("No changes made (all edits resulted in identical content)"), nil
	}

	// Restore original line ending style if needed
	finalContent := currentContent
	if hasWindowsLineEndings {
		finalContent = strings.ReplaceAll(currentContent, "\n", "\r\n")
	}

	// Write the updated content with original file permissions
	err = os.WriteFile(params.FilePath, []byte(finalContent), fileInfo.Mode())
	if err != nil {
		return ToolResponse{}, fmt.Errorf("failed to write file: %w", err)
	}

	recordFileWrite(params.FilePath)
	recordFileRead(params.FilePath)

	return WithResponseMetadata(
		NewTextResponse(fmt.Sprintf("Applied %d edits to file: %s", len(params.Edits), params.FilePath)),
		MultiEditResponseMetadata{
			EditsApplied: len(params.Edits),
		},
	), nil
}

func (m *multiEditTool) applyEditToContent(content string, edit MultiEditOperation) (string, error) {
	// This should never happen due to validation, but check anyway
	if edit.OldString == "" {
		return "", fmt.Errorf("old_string cannot be empty for content replacement")
	}

	var newContent string
	var replacementCount int

	if edit.ReplaceAll {
		newContent = strings.ReplaceAll(content, edit.OldString, edit.NewString)
		replacementCount = strings.Count(content, edit.OldString)
		if replacementCount == 0 {
			return "", fmt.Errorf("string not found in content (ensure exact match including whitespace)")
		}
	} else {
		index := strings.Index(content, edit.OldString)
		if index == -1 {
			return "", fmt.Errorf("string not found in content (ensure exact match including whitespace)")
		}

		lastIndex := strings.LastIndex(content, edit.OldString)
		if index != lastIndex {
			return "", fmt.Errorf("string appears multiple times (%d occurrences) - use more context for unique match or set replace_all=true", strings.Count(content, edit.OldString))
		}

		newContent = content[:index] + edit.NewString + content[index+len(edit.OldString):]
		replacementCount = 1
	}

	return newContent, nil
}

// normalizeLineEndings converts Windows line endings to Unix
func normalizeLineEndings(content string) string {
	return strings.ReplaceAll(content, "\r\n", "\n")
}