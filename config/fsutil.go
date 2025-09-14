package config

import (
	"os"
	"path/filepath"
)

// SearchParent searches for a file or directory in parent directories
func SearchParent(startDir, target string) (string, bool) {
	current := startDir
	
	for {
		targetPath := filepath.Join(current, target)
		if _, err := os.Stat(targetPath); err == nil {
			return targetPath, true
		}
		
		parent := filepath.Dir(current)
		if parent == current {
			// Reached root directory
			break
		}
		current = parent
	}
	
	return "", false
}

// HomeDir returns the user's home directory
func HomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to HOME environment variable
		return os.Getenv("HOME")
	}
	return home
}