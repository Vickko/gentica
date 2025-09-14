package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type BashParams struct {
	Command string `json:"command"`
	Timeout int    `json:"timeout"`
}

type BashResponseMetadata struct {
	StartTime        int64  `json:"start_time"`
	EndTime          int64  `json:"end_time"`
	Output           string `json:"output"`
	WorkingDirectory string `json:"working_directory"`
}

type bashTool struct {
	workingDir string
}

const (
	BashToolName = "bash"

	DefaultTimeout  = 1 * 60 * 1000  // 1 minutes in milliseconds
	MaxTimeout      = 10 * 60 * 1000 // 10 minutes in milliseconds
	MaxOutputLength = 30000
	BashNoOutput    = "no output"

	bashDescription = `Execute bash commands in a shell environment.

WHEN TO USE THIS TOOL:
- Use when you need to run shell commands or scripts
- Helpful for system operations, file management, and automation
- Perfect for running build commands, tests, or other CLI tools

HOW TO USE:
- Provide the command you want to execute
- Optionally specify a timeout in milliseconds (default: 1 minute, max: 10 minutes)

FEATURES:
- Executes commands in a shell environment
- Captures stdout and stderr
- Returns exit code information
- Enforces timeout limits

LIMITATIONS:
- Some commands are blocked for security (like curl, wget)
- Cannot run interactive commands
- Output is truncated if exceeds 30000 characters

TIPS:
- Use semicolons or && to chain multiple commands
- Check exit codes for command success/failure
- Be mindful of the working directory`
)

// Simple list of banned commands for security
var bannedCommands = []string{
	// Network tools
	"curl", "wget", "nc", "netcat", "telnet", "ssh", "scp", "sftp", "ftp", "rsync", "nmap",
	// Privilege escalation
	"sudo", "su", "doas",
	// Permission and ownership changes
	"chmod", "chown", "chgrp",
	// Dangerous destructive commands
	"rm -rf /", "rm -rf /*", "dd if=/dev/zero", "dd if=/dev/random",
	// System control
	"shutdown", "reboot", "halt", "poweroff", "init",
	// Process control (dangerous variants)
	"kill -9 -1", "pkill -9", "killall -9",
	// Disk and filesystem operations
	"mkfs", "fdisk", "parted", "format",
	// Package managers (global installs)
	"apt install", "yum install", "brew install", "pip install --system",
	"npm install -g", "gem install",
}

func NewBashTool(workingDir string) BaseTool {
	return &bashTool{
		workingDir: workingDir,
	}
}

func (b *bashTool) Name() string {
	return BashToolName
}

func (b *bashTool) Info() ToolInfo {
	return ToolInfo{
		Name:        BashToolName,
		Description: bashDescription,
		Parameters: map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "The bash command to execute",
			},
			"timeout": map[string]any{
				"type":        "number",
				"description": "Optional timeout in milliseconds (max 600000)",
			},
		},
		Required: []string{"command"},
	}
}

func (b *bashTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params BashParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse("invalid parameters: " + err.Error()), nil
	}

	if params.Command == "" {
		return NewTextErrorResponse("command is required"), nil
	}

	// Check for banned commands
	cmdLower := strings.ToLower(params.Command)
	for _, banned := range bannedCommands {
		if strings.Contains(cmdLower, banned) {
			return NewTextErrorResponse(fmt.Sprintf("command '%s' is not allowed for security reasons", banned)), nil
		}
	}

	// Set timeout
	timeout := DefaultTimeout
	if params.Timeout > 0 {
		timeout = params.Timeout
		if timeout > MaxTimeout {
			timeout = MaxTimeout
		}
	}

	// Create context with timeout
	timeoutDuration := time.Duration(timeout) * time.Millisecond
	cmdCtx, cancel := context.WithTimeout(ctx, timeoutDuration)
	defer cancel()

	// Execute command
	startTime := time.Now().UnixMilli()
	
	// Use sh -c to execute the command in a shell
	cmd := exec.CommandContext(cmdCtx, "sh", "-c", params.Command)
	cmd.Dir = b.workingDir
	
	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	// Run the command
	err := cmd.Run()
	
	endTime := time.Now().UnixMilli()
	
	// Combine output
	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += stderr.String()
	}
	
	// Truncate if too long
	if len(output) > MaxOutputLength {
		output = output[:MaxOutputLength] + "\n... (output truncated)"
	}
	
	// Handle empty output
	if output == "" {
		output = BashNoOutput
	}
	
	// Check for timeout
	if cmdCtx.Err() == context.DeadlineExceeded {
		return WithResponseMetadata(
			NewTextErrorResponse(fmt.Sprintf("Command timed out after %d ms", timeout)),
			BashResponseMetadata{
				StartTime:        startTime,
				EndTime:          endTime,
				Output:           output,
				WorkingDirectory: b.workingDir,
			},
		), nil
	}
	
	// Get exit code
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return ToolResponse{}, fmt.Errorf("failed to execute command: %w", err)
		}
	}
	
	// Format result - non-zero exit code is considered an error
	if exitCode != 0 {
		result := fmt.Sprintf("%s\n\nExit code: %d", output, exitCode)
		return WithResponseMetadata(
			NewTextErrorResponse(result),
			BashResponseMetadata{
				StartTime:        startTime,
				EndTime:          endTime,
				Output:           output,
				WorkingDirectory: b.workingDir,
			},
		), nil
	}
	
	return WithResponseMetadata(
		NewTextResponse(output),
		BashResponseMetadata{
			StartTime:        startTime,
			EndTime:          endTime,
			Output:           output,
			WorkingDirectory: b.workingDir,
		},
	), nil
}