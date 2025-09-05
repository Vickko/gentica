package llm

import (
	"os"
	"path/filepath"
	"strings"
)

// Long expands the ~ in a path to the home directory
func HomeLong(path string) string {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[1:])
	}
	return path
}