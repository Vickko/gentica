package tools

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
)

// readFile 读取指定文件的指定行范围
// path: 文件路径
// startLine: 起始行号（包含，从1开始）
// endLine: 结束行号（不包含，从1开始）
// 当 startLine=endLine=0 时，读取全部文件
func readFile(path string, startLine, endLine int) ([]string, error) {
	// 处理读取全部文件的情况
	if startLine == 0 && endLine == 0 {
		return readAllLines(path)
	}

	// 验证参数
	if startLine <= 0 {
		return nil, fmt.Errorf("startLine must be greater than 0")
	}
	if endLine <= startLine {
		return nil, fmt.Errorf("endLine must be greater than startLine")
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	currentLine := 1

	for scanner.Scan() {
		if currentLine >= startLine && currentLine < endLine {
			lines = append(lines, scanner.Text())
		}
		if currentLine >= endLine {
			break
		}
		currentLine++
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	// 检查是否startLine超出文件行数
	if currentLine <= startLine {
		return nil, fmt.Errorf("startLine %d exceeds file length", startLine)
	}

	return lines, nil
}

// readAllLines 读取文件的所有行
func readAllLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return lines, nil
}

// ReadFile 读取指定文件的指定行范围，支持显示行号
// path: 文件路径
// startLine: 起始行号（包含，从1开始）
// endLine: 结束行号（不包含，从1开始）
// showLineNumbers: 是否显示行号
// 当 startLine=endLine=0 时，读取全部文件
func ReadFile(path string, startLine, endLine int, showLineNumbers bool) ([]string, error) {
	lines, err := readFile(path, startLine, endLine)
	if err != nil {
		return nil, err
	}

	if !showLineNumbers {
		return lines, nil
	}

	// 计算行号格式
	totalLines := len(lines)
	actualStartLine := 1
	if startLine > 0 {
		actualStartLine = startLine
	}

	// 计算最大行号的宽度用于对齐
	maxLineNum := actualStartLine + totalLines - 1
	width := len(strconv.Itoa(maxLineNum))

	// 添加行号
	var numberedLines []string
	for i, line := range lines {
		lineNum := actualStartLine + i
		numberedLine := fmt.Sprintf("%*d\t%s", width, lineNum, line)
		numberedLines = append(numberedLines, numberedLine)
	}

	return numberedLines, nil
}
