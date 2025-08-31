package tools

import (
	"bufio"
	"fmt"
	"os"
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
