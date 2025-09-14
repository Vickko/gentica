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

type LSParams struct {
	Path   string   `json:"path"`
	Ignore []string `json:"ignore"`
}

type TreeNode struct {
	Name     string      `json:"name"`
	Path     string      `json:"path"`
	Type     string      `json:"type"` // "file" or "directory"
	Children []*TreeNode `json:"children,omitempty"`
}

type LSResponseMetadata struct {
	NumberOfFiles int  `json:"number_of_files"`
	Truncated     bool `json:"truncated"`
}

type lsTool struct {
	workingDir string
}

const (
	LSToolName    = "ls"
	MaxLSFiles    = 1000
	lsDescription = `Directory listing tool that shows files and subdirectories in a tree structure, helping you explore and understand the project organization.

WHEN TO USE THIS TOOL:
- Use when you need to explore the structure of a directory
- Helpful for understanding the organization of a project
- Good first step when getting familiar with a new codebase

HOW TO USE:
- Provide a path to list (defaults to current working directory)
- Optionally specify glob patterns to ignore
- Results are displayed in a tree structure

FEATURES:
- Displays a hierarchical view of files and directories
- Automatically skips hidden files/directories (starting with '.')
- Skips common system directories like __pycache__
- Can filter out files matching specific patterns

LIMITATIONS:
- Results are limited to 1000 files
- Very large directories will be truncated
- Does not show file sizes or permissions
- Cannot recursively list all directories in a large project

WINDOWS NOTES:
- Hidden file detection uses Unix convention (files starting with '.')
- Windows-specific hidden files (with hidden attribute) are not automatically skipped
- Common Windows directories like System32, Program Files are not in default ignore list
- Path separators are handled automatically (both / and \ work)

TIPS:
- Use Glob tool for finding files by name patterns instead of browsing
- Use Grep tool for searching file contents
- Combine with other tools for more effective exploration`
)

func NewLsTool(workingDir string) BaseTool {
	return &lsTool{
		workingDir: workingDir,
	}
}

func (l *lsTool) Name() string {
	return LSToolName
}

func (l *lsTool) Info() ToolInfo {
	return ToolInfo{
		Name:        LSToolName,
		Description: lsDescription,
		Parameters: map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The path to the directory to list (defaults to current working directory)",
			},
			"ignore": map[string]any{
				"type":        "array",
				"description": "List of glob patterns to ignore",
				"items": map[string]any{
					"type": "string",
				},
			},
		},
		Required: []string{},
	}
}

func (l *lsTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params LSParams
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

	// Check if path is a directory
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

	output, fileCount, truncated, err := ListDirectoryTree(searchPath, params.Ignore)
	if err != nil {
		return NewTextErrorResponse(fmt.Sprintf("error listing directory: %s", err)), nil
	}

	return WithResponseMetadata(
		NewTextResponse(output),
		LSResponseMetadata{
			NumberOfFiles: fileCount,
			Truncated:     truncated,
		},
	), nil
}

func ListDirectoryTree(searchPath string, ignore []string) (string, int, bool, error) {
	files, truncated, err := listDirectory(searchPath, ignore, MaxLSFiles)
	if err != nil {
		return "", 0, false, fmt.Errorf("error listing directory: %w", err)
	}

	// Handle empty directory
	if len(files) == 0 {
		output := fmt.Sprintf("- %s%c\n  (empty)\n", searchPath, filepath.Separator)
		return output, 0, false, nil
	}

	tree := createFileTree(files, searchPath)
	output := printTree(tree, searchPath)

	if truncated {
		output = fmt.Sprintf("There are more than %d files in the directory. Use a more specific path or use the Glob tool to find specific files. The first %d files and directories are included below:\n\n%s", MaxLSFiles, MaxLSFiles, output)
	}

	return output, len(files), truncated, nil
}

// listDirectory lists files in a directory, applying ignore patterns
func listDirectory(rootPath string, ignorePatterns []string, maxFiles int) ([]string, bool, error) {
	var results []string
	truncated := false
	fileCount := 0

	// Default ignore patterns
	defaultIgnore := []string{
		".*",           // Hidden files
		"__pycache__",  // Python cache
		"node_modules", // Node modules
		".git",         // Git directory
	}

	allIgnorePatterns := append(defaultIgnore, ignorePatterns...)

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files/dirs with errors
		}

		// Skip the root directory itself
		if path == rootPath {
			return nil
		}

		// Check ignore patterns
		relPath, err := filepath.Rel(rootPath, path)
		if err != nil {
			relPath = path
		}
		baseName := filepath.Base(path)
		
		for _, pattern := range allIgnorePatterns {
			var matched bool
			
			// Handle recursive patterns like "test/**"
			if strings.Contains(pattern, "/**") {
				prefix := strings.TrimSuffix(pattern, "/**")
				if strings.HasPrefix(relPath, prefix+string(filepath.Separator)) || relPath == prefix {
					matched = true
				}
			} else if strings.Contains(pattern, "**") {
				// Handle general ** patterns
				matched, _ = filepath.Match(pattern, relPath)
				if !matched {
					matched, _ = filepath.Match(pattern, baseName)
				}
			} else {
				// Simple pattern matching on base name
				matched, _ = filepath.Match(pattern, baseName)
			}
			
			if matched {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// Check if we've reached max files
		if fileCount >= maxFiles {
			truncated = true
			return filepath.SkipAll
		}

		fileCount++
		results = append(results, path)
		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return nil, false, err
	}

	// Sort the results
	sort.Strings(results)

	return results, truncated, nil
}

func createFileTree(sortedPaths []string, rootPath string) []*TreeNode {
	root := []*TreeNode{}
	pathMap := make(map[string]*TreeNode)

	for _, path := range sortedPaths {
		relativePath := strings.TrimPrefix(path, rootPath)
		parts := strings.Split(relativePath, string(filepath.Separator))
		currentPath := ""
		var parentPath string

		var cleanParts []string
		for _, part := range parts {
			if part != "" {
				cleanParts = append(cleanParts, part)
			}
		}
		parts = cleanParts

		if len(parts) == 0 {
			continue
		}

		for i, part := range parts {
			if currentPath == "" {
				currentPath = part
			} else {
				currentPath = filepath.Join(currentPath, part)
			}

			if _, exists := pathMap[currentPath]; exists {
				parentPath = currentPath
				continue
			}

			// Check if it's a directory by checking if it exists as a directory
			fullPath := filepath.Join(rootPath, currentPath)
			info, err := os.Stat(fullPath)
			isDir := err == nil && info.IsDir()
			
			nodeType := "file"
			if isDir {
				nodeType = "directory"
			}
			
			newNode := &TreeNode{
				Name:     part,
				Path:     currentPath,
				Type:     nodeType,
				Children: []*TreeNode{},
			}

			pathMap[currentPath] = newNode

			if i > 0 && parentPath != "" {
				if parent, ok := pathMap[parentPath]; ok {
					parent.Children = append(parent.Children, newNode)
				}
			} else {
				root = append(root, newNode)
			}

			parentPath = currentPath
		}
	}

	return root
}

func printTree(tree []*TreeNode, rootPath string) string {
	var result strings.Builder

	result.WriteString("- ")
	result.WriteString(rootPath)
	if len(rootPath) > 0 && rootPath[len(rootPath)-1] != filepath.Separator {
		result.WriteByte(filepath.Separator)
	}
	result.WriteByte('\n')

	for _, node := range tree {
		printNode(&result, node, 1)
	}

	return result.String()
}

func printNode(builder *strings.Builder, node *TreeNode, level int) {
	indent := strings.Repeat("  ", level)

	nodeName := node.Name
	if node.Type == "directory" {
		nodeName = nodeName + string(filepath.Separator)
	}

	fmt.Fprintf(builder, "%s- %s\n", indent, nodeName)

	if node.Type == "directory" && len(node.Children) > 0 {
		for _, child := range node.Children {
			printNode(builder, child, level+1)
		}
	}
}