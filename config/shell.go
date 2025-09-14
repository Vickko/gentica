package config

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// ShellExecutor provides shell command execution
type ShellExecutor struct {
	env []string
}

// ShellOptions contains options for shell creation
type ShellOptions struct {
	Env []string
}

// NewShellExecutor creates a new ShellExecutor
func NewShellExecutor(opts *ShellOptions) *ShellExecutor {
	s := &ShellExecutor{}
	if opts != nil {
		s.env = opts.Env
	}
	return s
}

// Exec executes a shell command and returns stdout, stderr, and error
func (s *ShellExecutor) Exec(ctx context.Context, command string) (string, string, error) {
	var cmd *exec.Cmd
	
	// Determine shell based on OS
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}
	
	if s.env != nil {
		cmd.Env = s.env
	}
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	
	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err
}

// SimpleExec is a helper that executes a command and returns stdout or error
func SimpleExec(command string) (string, error) {
	ctx := context.Background()
	shell := NewShellExecutor(nil)
	stdout, stderr, err := shell.Exec(ctx, command)
	if err != nil {
		if stderr != "" {
			return "", fmt.Errorf("%w: %s", err, stderr)
		}
		return "", err
	}
	return stdout, nil
}