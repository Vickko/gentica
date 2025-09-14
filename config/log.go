package config

import (
	"log/slog"
	"os"
	"path/filepath"
)

// SetupLog configures the logging system
func SetupLog(logPath string, debug bool) error {
	// Ensure log directory exists
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}
	
	// Open or create log file
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	
	// Configure log level
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}
	
	// Create handler with options
	opts := &slog.HandlerOptions{
		Level: level,
	}
	
	// Create text handler writing to file
	handler := slog.NewTextHandler(logFile, opts)
	
	// Set as default logger
	slog.SetDefault(slog.New(handler))
	
	return nil
}