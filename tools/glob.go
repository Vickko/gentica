package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	GlobToolName    = "glob"
	globDescription = `Fast file pattern matching tool that finds files by name and pattern, returning matching paths sorted by modification time (newest first).

WHEN TO USE THIS TOOL:
- Use when you need to find files by name patterns or extensions
- Great for finding specific file types across a directory structure
- Useful for discovering files that match certain naming conventions

HOW TO USE:
- Provide a glob pattern to match against file paths
- Optionally specify a starting directory (defaults to current working directory)
- Results are sorted with most recently modified files first

GLOB PATTERN SYNTAX:
- '*' matches any sequence of non-separator characters
- '**' matches any sequence of characters, including separators
- '?' matches any single non-separator character
- '[...]' matches any character in the brackets
- '[!...]' matches any character not in the brackets

COMMON PATTERN EXAMPLES:
- '*.js' - Find all JavaScript files in the current directory
- '**/*.js' - Find all JavaScript files in any subdirectory
- 'src/**/*.{ts,tsx}' - Find all TypeScript files in the src directory
- '*.{html,css,js}' - Find all HTML, CSS, and JS files

LIMITATIONS:
- Results are limited to 100 files (newest first)
- Does not search file contents (use Grep tool for that)
- Hidden files (starting with '.') are skipped by default

WINDOWS NOTES:
- Path separators are handled automatically (both / and \ work)

TIPS:
- Patterns should use forward slashes (/) for cross-platform compatibility
- For the most useful results, combine with the Grep tool: first find files with Glob, then search their contents with Grep
- When doing iterative exploration that may require multiple rounds of searching, consider using the Agent tool instead
- Always check if results are truncated and refine your search pattern if needed`
)

type GlobParams struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path"`
}

type GlobResponseMetadata struct {
	NumberOfFiles int  `json:"number_of_files"`
	Truncated     bool `json:"truncated"`
}

type globTool struct {
	workingDir string
}

type fileInfo struct {
	path    string
	modTime int64
}

func NewGlobTool(workingDir string) BaseTool {
	return &globTool{
		workingDir: workingDir,
	}
}

func (g *globTool) Name() string {
	return GlobToolName
}

func (g *globTool) Info() ToolInfo {
	return ToolInfo{
		Name:        GlobToolName,
		Description: globDescription,
		Parameters: map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "The glob pattern to match files against",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "The directory to search in. Defaults to the current working directory.",
			},
		},
		Required: []string{"pattern"},
	}
}

func (g *globTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params GlobParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse(fmt.Sprintf("invalid parameters: %s", err)), nil
	}

	if params.Pattern == "" {
		return NewTextErrorResponse("pattern is required"), nil
	}

	searchPath := params.Path
	if searchPath == "" {
		searchPath = g.workingDir
	}

	// Make path absolute
	if !filepath.IsAbs(searchPath) {
		searchPath = filepath.Join(g.workingDir, searchPath)
	}

	files, truncated, err := globFiles(ctx, params.Pattern, searchPath, 100)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("error finding files: %w", err)
	}

	var output string
	if len(files) == 0 {
		output = "No files found"
	} else {
		output = strings.Join(files, "\n")
		if truncated {
			output += "\n\n(Results are truncated. Consider using a more specific path or pattern.)"
		}
	}

	return WithResponseMetadata(
		NewTextResponse(output),
		GlobResponseMetadata{
			NumberOfFiles: len(files),
			Truncated:     truncated,
		},
	), nil
}

func globFiles(ctx context.Context, pattern, searchPath string, limit int) ([]string, bool, error) {
	// Handle brace expansion patterns
	if strings.Contains(pattern, "{") && strings.Contains(pattern, "}") {
		return globWithBraceExpansion(ctx, pattern, searchPath, limit)
	}

	// Handle ** patterns by walking the directory tree
	if strings.Contains(pattern, "**") {
		return globWithDoublestar(ctx, pattern, searchPath, limit)
	}

	// For simple patterns, use filepath.Glob
	fullPattern := filepath.Join(searchPath, pattern)
	matches, err := filepath.Glob(fullPattern)
	if err != nil {
		return nil, false, err
	}

	// Filter out hidden files and collect file info
	var files []fileInfo
	for _, match := range matches {
		// Skip hidden files
		if isHidden(match) {
			continue
		}

		info, err := os.Stat(match)
		if err != nil {
			continue
		}

		// Skip directories
		if info.IsDir() {
			continue
		}

		files = append(files, fileInfo{
			path:    match,
			modTime: info.ModTime().Unix(),
		})
	}

	// Sort by modification time (newest first)
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime > files[j].modTime
	})

	// Convert to string slice and apply limit
	var result []string
	truncated := false
	for i, f := range files {
		if limit > 0 && i >= limit {
			truncated = true
			break
		}
		result = append(result, f.path)
	}

	return result, truncated, nil
}

func globWithDoublestar(ctx context.Context, pattern, searchPath string, limit int) ([]string, bool, error) {
	var files []fileInfo
	truncated := false

	// Convert pattern to work with filepath.Match
	// Replace ** with a placeholder, then split
	parts := strings.Split(pattern, "**")
	if len(parts) != 2 {
		// Complex pattern, fallback to simple matching
		parts = []string{"", pattern}
	}

	prefix := strings.TrimSuffix(parts[0], "/")
	suffix := strings.TrimPrefix(parts[1], "/")

	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			return nil // Continue walking despite errors
		}

		// Skip directories
		if info.IsDir() {
			// Skip hidden directories
			if isHidden(path) && path != searchPath {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip hidden files
		if isHidden(path) {
			return nil
		}

		// Check if the file matches the pattern
		relPath, _ := filepath.Rel(searchPath, path)

		// Check prefix if any
		if prefix != "" && !strings.HasPrefix(relPath, prefix) {
			return nil
		}

		// Check suffix pattern
		if suffix != "" {
			// For suffix, we need to check the end part
			matched, _ := filepath.Match(suffix, filepath.Base(path))
			if !matched {
				// Also try matching against the relative path
				if !strings.HasSuffix(relPath, suffix) && !matchPattern(relPath, suffix) {
					return nil
				}
			}
		}

		files = append(files, fileInfo{
			path:    path,
			modTime: info.ModTime().Unix(),
		})

		// Early exit if we have enough files
		if limit > 0 && len(files) > limit*2 {
			truncated = true
			return filepath.SkipAll
		}

		return nil
	})

	if err != nil && !errors.Is(err, filepath.SkipAll) {
		return nil, false, err
	}

	// Sort by modification time (newest first)
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime > files[j].modTime
	})

	// Convert to string slice and apply limit
	var result []string
	for i, f := range files {
		if limit > 0 && i >= limit {
			truncated = true
			break
		}
		result = append(result, f.path)
	}

	return result, truncated, nil
}

func matchPattern(path, pattern string) bool {
	// Simple pattern matching for common cases
	if strings.HasPrefix(pattern, "*.") {
		ext := pattern[1:] // Get extension including dot
		return strings.HasSuffix(path, ext)
	}

	matched, _ := filepath.Match(pattern, filepath.Base(path))
	return matched
}

// expandBraces expands brace patterns like "*.{js,css}" into ["*.js", "*.css"]
func expandBraces(pattern string) []string {
	start := strings.Index(pattern, "{")
	end := strings.Index(pattern, "}")
	
	if start == -1 || end == -1 || end < start {
		return []string{pattern}
	}
	
	prefix := pattern[:start]
	suffix := pattern[end+1:]
	options := strings.Split(pattern[start+1:end], ",")
	
	var patterns []string
	for _, opt := range options {
		expanded := prefix + strings.TrimSpace(opt) + suffix
		// Recursively expand if there are more braces
		if strings.Contains(expanded, "{") && strings.Contains(expanded, "}") {
			patterns = append(patterns, expandBraces(expanded)...)
		} else {
			patterns = append(patterns, expanded)
		}
	}
	
	return patterns
}

// globWithBraceExpansion handles patterns with brace expansion
func globWithBraceExpansion(ctx context.Context, pattern, searchPath string, limit int) ([]string, bool, error) {
	expandedPatterns := expandBraces(pattern)
	var allFiles []fileInfo
	fileMap := make(map[string]bool) // To avoid duplicates
	
	for _, expandedPattern := range expandedPatterns {
		files, _, err := globFiles(ctx, expandedPattern, searchPath, 0) // No limit for individual patterns
		if err != nil {
			return nil, false, err
		}
		
		for _, file := range files {
			if !fileMap[file] {
				fileMap[file] = true
				info, err := os.Stat(file)
				if err == nil && !info.IsDir() {
					allFiles = append(allFiles, fileInfo{
						path:    file,
						modTime: info.ModTime().Unix(),
					})
				}
			}
		}
	}
	
	// Sort by modification time (newest first)
	sort.Slice(allFiles, func(i, j int) bool {
		return allFiles[i].modTime > allFiles[j].modTime
	})
	
	// Convert to string slice and apply limit
	var result []string
	truncated := false
	for i, f := range allFiles {
		if limit > 0 && i >= limit {
			truncated = true
			break
		}
		result = append(result, f.path)
	}
	
	return result, truncated, nil
}
