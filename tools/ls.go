package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type LsParams struct {
	Path string `json:"path"`
	All  bool   `json:"all"` // Show hidden files (like ls -a)
}

type LsResponseMetadata struct {
	NumberOfFiles int  `json:"number_of_files"`
	Truncated     bool `json:"truncated"`
}

type lsTool struct {
	workingDir string
}

const (
	LsToolName    = "ls"
	MaxLsFiles    = 512
	lsDescription = `List directory contents, similar to Unix ls command.

WHEN TO USE THIS TOOL:
- Use when you need to see what files and directories are in a specific location
- Quick overview of directory contents without hierarchical structure
- Good for checking if specific files exist in a directory
- Use instead of tree when you only need immediate children

HOW TO USE:
- Provide a path to list (defaults to current working directory)
- Use all=true to show hidden files and directories (starting with '.')

FEATURES:
- Lists files and directories in alphabetical order
- Shows only immediate children (no recursion)
- Directories are marked with trailing /
- Hidden files (starting with '.') are hidden by default unless all=true
- One item per line for simple parsing

LIMITATIONS:
- Results are limited to 512 files
- Does not show file sizes, permissions, or dates
- No recursive listing (use tree tool for hierarchical view)
- No filtering options beyond hidden files

TIPS:
- Use tree tool for hierarchical directory structure
- Use glob tool for pattern-based file searching
- Use grep tool for searching file contents`
)

func NewLsTool(workingDir string) BaseTool {
	return &lsTool{
		workingDir: workingDir,
	}
}

func (l *lsTool) Name() string {
	return LsToolName
}

func (l *lsTool) Info() ToolInfo {
	return ToolInfo{
		Name:        LsToolName,
		Description: lsDescription,
		Parameters: map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The path to the directory to list (defaults to current working directory)",
			},
			"all": map[string]any{
				"type":        "boolean",
				"description": "Show all files including hidden ones (default: false)",
			},
		},
		Required: []string{},
	}
}

func (l *lsTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params LsParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse(fmt.Sprintf("invalid parameters: %s", err)), nil
	}

	searchPath := params.Path
	if searchPath == "" {
		searchPath = l.workingDir
	}

	// Expand home directory if needed
	if strings.HasPrefix(searchPath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			searchPath = filepath.Join(homeDir, searchPath[2:])
		}
	}

	if !filepath.IsAbs(searchPath) {
		searchPath = filepath.Join(l.workingDir, searchPath)
	}

	// Check if path exists and is a directory
	info, err := os.Stat(searchPath)
	if err != nil {
		if os.IsNotExist(err) {
			return NewTextErrorResponse(fmt.Sprintf("path does not exist: %s", searchPath)), nil
		}
		return NewTextErrorResponse(fmt.Sprintf("error accessing path: %s", err)), nil
	}

	if !info.IsDir() {
		return NewTextErrorResponse(fmt.Sprintf("not a directory: %s", searchPath)), nil
	}

	output, metadata, err := ListDirectory(searchPath, params.All, MaxLsFiles)
	if err != nil {
		return NewTextErrorResponse(fmt.Sprintf("error listing directory: %s", err)), nil
	}

	return WithResponseMetadata(
		NewTextResponse(output),
		metadata,
	), nil
}

func ListDirectory(searchPath string, showAll bool, maxFiles int) (string, LsResponseMetadata, error) {
	entries, err := os.ReadDir(searchPath)
	if err != nil {
		return "", LsResponseMetadata{}, fmt.Errorf("error reading directory: %w", err)
	}

	// Filter and collect entries
	var items []string
	truncated := false
	fileCount := 0

	for _, entry := range entries {
		name := entry.Name()

		// Skip hidden files unless showAll is true
		if !showAll && strings.HasPrefix(name, ".") {
			continue
		}

		// Check if we've reached max files
		if fileCount >= maxFiles {
			truncated = true
			break
		}

		// Add directory marker
		if entry.IsDir() {
			name = name + "/"
		}

		items = append(items, name)
		fileCount++
	}

	// Sort items alphabetically
	sort.Strings(items)

	// Build output
	var output strings.Builder

	// Add warning if truncated
	if truncated {
		output.WriteString(fmt.Sprintf("There are more than %d files in the directory. Use a more specific path or filter files.\n\n", maxFiles))
	}

	// Handle empty directory
	if len(items) == 0 {
		output.WriteString("(empty)\n")
	} else {
		// Write each item on its own line
		for _, item := range items {
			output.WriteString(item)
			output.WriteString("\n")
		}
	}

	metadata := LsResponseMetadata{
		NumberOfFiles: fileCount,
		Truncated:     truncated,
	}

	return output.String(), metadata, nil
}