package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// FileInfo 表示文件信息
type FileInfo struct {
	Name    string    // 文件名
	Path    string    // 完整路径
	IsDir   bool      // 是否为目录
	Size    int64     // 文件大小（字节）
	ModTime time.Time // 修改时间
}

// ListFiles 列出指定目录下的文件和子目录
// directoryPath: 目录路径
// recursive: 是否递归遍历子目录
func ListFiles(directoryPath string, recursive bool) ([]FileInfo, error) {
	// 检查目录是否存在
	info, err := os.Stat(directoryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to access directory: %w", err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("path '%s' is not a directory", directoryPath)
	}

	var files []FileInfo

	if recursive {
		// 递归遍历
		err = filepath.Walk(directoryPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// 跳过根目录本身
			if path == directoryPath {
				return nil
			}

			files = append(files, FileInfo{
				Name:    info.Name(),
				Path:    path,
				IsDir:   info.IsDir(),
				Size:    info.Size(),
				ModTime: info.ModTime(),
			})

			return nil
		})
	} else {
		// 只列出当前目录
		entries, err := os.ReadDir(directoryPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory: %w", err)
		}

		for _, entry := range entries {
			info, err := entry.Info()
			if err != nil {
				continue // 跳过无法获取信息的条目
			}

			fullPath := filepath.Join(directoryPath, entry.Name())
			files = append(files, FileInfo{
				Name:    entry.Name(),
				Path:    fullPath,
				IsDir:   entry.IsDir(),
				Size:    info.Size(),
				ModTime: info.ModTime(),
			})
		}
	}

	return files, err
}