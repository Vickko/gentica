package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchInDirectory(t *testing.T) {
	// 创建临时测试目录
	tempDir, err := os.MkdirTemp("", "search_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 测试用例结构
	testFiles := map[string]string{
		"file1.txt":          "This is a test file\nError: Something went wrong\nNormal line here",
		"subdir/file2.py":    "print(\"Hello World\")\nprint(\"Warning: This is a test.\")\ndef main():",
		"logs/system.log":    "INFO: System started\nCRITICAL ERROR: Database connection failed.\nINFO: Recovery started",
		"empty.txt":          "",
		"single_line.txt":    "Single line with error message",
		"no_match.txt":       "This file has no matches\nJust normal content\nNothing special",
		"special_chars.txt":  "Line with [brackets] and (parentheses)\nRegex test: a*b+c?\nEnd of file",
	}

	// 创建测试文件和目录结构
	for filePath, content := range testFiles {
		fullPath := filepath.Join(tempDir, filePath)
		dir := filepath.Dir(fullPath)
		
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
		
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file %s: %v", fullPath, err)
		}
	}

	tests := []struct {
		name        string
		pattern     string
		expected    []string
		expectError bool
	}{
		{
			name:    "Simple string match",
			pattern: "Error",
			expected: []string{
				"./file1.txt:2:Error: Something went wrong",
			},
		},
		{
			name:    "Case insensitive with regex flag",
			pattern: "(?i)error",
			expected: []string{
				"./file1.txt:2:Error: Something went wrong",
				"./logs/system.log:2:CRITICAL ERROR: Database connection failed.",
				"./single_line.txt:1:Single line with error message",
			},
		},
		{
			name:    "Case sensitive match",
			pattern: "error",
			expected: []string{
				"./single_line.txt:1:Single line with error message",
			},
		},
		{
			name:    "Regex pattern with word boundary",
			pattern: `\bprint\b`,
			expected: []string{
				"./subdir/file2.py:1:print(\"Hello World\")",
				"./subdir/file2.py:2:print(\"Warning: This is a test.\")",
			},
		},
		{
			name:    "Regex with special characters",
			pattern: `\[.*\]`,
			expected: []string{
				"./special_chars.txt:1:Line with [brackets] and (parentheses)",
			},
		},
		{
			name:    "Regex quantifiers",
			pattern: `a\*b\+c\?`,
			expected: []string{
				"./special_chars.txt:2:Regex test: a*b+c?",
			},
		},
		{
			name:    "Pattern with line anchors",
			pattern: `^INFO:`,
			expected: []string{
				"./logs/system.log:1:INFO: System started",
				"./logs/system.log:3:INFO: Recovery started",
			},
		},
		{
			name:     "No matches",
			pattern:  "nonexistent",
			expected: []string{},
		},
		{
			name:        "Invalid regex pattern",
			pattern:     "[unclosed",
			expectError: true,
		},
		{
			name:    "Match in subdirectory",
			pattern: "Warning",
			expected: []string{
				"./subdir/file2.py:2:print(\"Warning: This is a test.\")",
			},
		},
		{
			name:    "Multiple matches in same file",
			pattern: "INFO",
			expected: []string{
				"./logs/system.log:1:INFO: System started",
				"./logs/system.log:3:INFO: Recovery started",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := SearchInDirectory(tempDir, tt.pattern)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(results) != len(tt.expected) {
				t.Errorf("Expected %d results, got %d.\nExpected: %v\nActual: %v", 
					len(tt.expected), len(results), tt.expected, results)
				return
			}

			// 将结果转换为map以便比较（因为文件遍历顺序可能不确定）
			resultMap := make(map[string]bool)
			for _, result := range results {
				resultMap[result] = true
			}

			for _, expected := range tt.expected {
				if !resultMap[expected] {
					t.Errorf("Expected result not found: %s\nActual results: %v", expected, results)
				}
			}
		})
	}
}

func TestSearchInDirectoryEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		dirPath     string
		pattern     string
		expectError bool
	}{
		{
			name:        "Nonexistent directory",
			dirPath:     "/nonexistent/directory",
			pattern:     "test",
			expectError: true,
		},
		{
			name:        "Empty pattern (should be valid)",
			dirPath:     ".",
			pattern:     "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := SearchInDirectory(tt.dirPath, tt.pattern)
			
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestSearchInDirectoryLargeFile(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "search_large_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建大文件
	largeFileName := filepath.Join(tempDir, "large.txt")
	file, err := os.Create(largeFileName)
	if err != nil {
		t.Fatalf("Failed to create large file: %v", err)
	}

	// 写入大量数据
	lines := []string{
		"Line 1: Normal content",
		"Line 2: ERROR occurred here",
		"Line 3: More normal content",
	}
	
	for i := 0; i < 1000; i++ {
		for j, line := range lines {
			lineNum := i*len(lines) + j + 1
			content := strings.Replace(line, "Line", fmt.Sprintf("Line %d", lineNum), 1)
			if _, err := file.WriteString(content + "\n"); err != nil {
				t.Fatalf("Failed to write to large file: %v", err)
			}
		}
	}
	file.Close()

	// 测试搜索大文件
	results, err := SearchInDirectory(tempDir, "ERROR")
	if err != nil {
		t.Fatalf("Error searching large file: %v", err)
	}

	// 应该找到1000个匹配项
	if len(results) != 1000 {
		t.Errorf("Expected 1000 matches, got %d", len(results))
	}

	// 验证第一个和最后一个结果的格式
	if len(results) > 0 {
		first := results[0]
		if !strings.HasPrefix(first, "./large.txt:2:Line 2") {
			t.Errorf("Unexpected first result format: %s", first)
		}
	}
}

func TestSearchInDirectoryBinaryFile(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "search_binary_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建二进制文件（包含null字节）
	binaryFile := filepath.Join(tempDir, "binary.bin")
	binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 'E', 'r', 'r', 'o', 'r', 0x00}
	if err := os.WriteFile(binaryFile, binaryData, 0644); err != nil {
		t.Fatalf("Failed to write binary file: %v", err)
	}

	// 创建正常文本文件作为对比
	textFile := filepath.Join(tempDir, "text.txt")
	if err := os.WriteFile(textFile, []byte("Error in text file"), 0644); err != nil {
		t.Fatalf("Failed to write text file: %v", err)
	}

	// 搜索应该能处理二进制文件而不崩溃
	results, err := SearchInDirectory(tempDir, "Error")
	if err != nil {
		t.Fatalf("Error searching directory with binary file: %v", err)
	}

	// 应该至少找到文本文件中的匹配项
	found := false
	for _, result := range results {
		if strings.Contains(result, "text.txt") {
			found = true
			break
		}
	}
	
	if !found {
		t.Errorf("Should find match in text file, results: %v", results)
	}
}