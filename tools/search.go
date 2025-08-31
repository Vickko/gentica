package tools

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// SearchInDirectory 在指定目录下搜索匹配正则表达式的内容
// dirPath: 要搜索的目录路径
// pattern: RE2规则的正则表达式
// 返回符合grep输出格式的结果: []string，格式为 "./相对路径:行号:匹配的行内容"
func SearchInDirectory(dirPath string, pattern string) ([]string, error) {
	// 编译正则表达式
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %v", err)
	}

	var results []string

	// 遍历目录
	err = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录
		if info.IsDir() {
			return nil
		}

		// 获取相对路径
		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			relPath = path
		}

		// 确保使用 ./ 前缀
		if !strings.HasPrefix(relPath, "./") {
			relPath = "./" + relPath
		}

		// 打开文件
		file, err := os.Open(path)
		if err != nil {
			// 跳过无法打开的文件（如二进制文件或权限问题）
			return nil
		}
		defer file.Close()

		// 逐行读取文件
		scanner := bufio.NewScanner(file)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()

			// 检查是否匹配正则表达式
			if regex.MatchString(line) {
				result := fmt.Sprintf("%s:%d:%s", relPath, lineNum, line)
				results = append(results, result)
			}
		}

		if err := scanner.Err(); err != nil {
			// 跳过读取错误（如二进制文件），但不返回错误
			return nil
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking directory: %v", err)
	}

	return results, nil
}