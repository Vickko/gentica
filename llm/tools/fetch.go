package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

type FetchParams struct {
	URL     string `json:"url"`
	Format  string `json:"format"`
	Timeout int    `json:"timeout,omitempty"`
}

type FetchPermissionsParams struct {
	URL     string `json:"url"`
	Format  string `json:"format"`
	Timeout int    `json:"timeout,omitempty"`
}

type fetchTool struct {
	client     *http.Client
	workingDir string
}

const (
	FetchToolName        = "fetch"
	fetchToolDescription = `Fetches content from a URL and returns it in the specified format.

WHEN TO USE THIS TOOL:
- Use when you need to download content from a URL
- Helpful for retrieving documentation, API responses, or web content
- Useful for getting external information to assist with tasks

HOW TO USE:
- Provide the URL to fetch content from
- Specify the desired output format (text or raw)
- Optionally set a timeout for the request

FEATURES:
- Supports two output formats: text and raw
- Automatically handles HTTP redirects
- Sets reasonable timeouts to prevent hanging
- Validates input parameters before making requests

LIMITATIONS:
- Maximum response size is 5MB
- Only supports HTTP and HTTPS protocols
- Cannot handle authentication or cookies
- Some websites may block automated requests

TIPS:
- Use text format for plain text content or simple API responses
- Use raw format when you need the exact response content
- Set appropriate timeouts for potentially slow websites`
)

func NewFetchTool(workingDir string) BaseTool {
	return &fetchTool{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		workingDir: workingDir,
	}
}

func (t *fetchTool) Name() string {
	return FetchToolName
}

func (t *fetchTool) Info() ToolInfo {
	return ToolInfo{
		Name:        FetchToolName,
		Description: fetchToolDescription,
		Parameters: map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "The URL to fetch content from",
			},
			"format": map[string]any{
				"type":        "string",
				"description": "The format to return the content in (text or raw)",
				"enum":        []string{"text", "raw"},
			},
			"timeout": map[string]any{
				"type":        "number",
				"description": "Optional timeout in seconds (max 120)",
			},
		},
		Required: []string{"url", "format"},
	}
}

func (t *fetchTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params FetchParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse("Failed to parse fetch parameters: " + err.Error()), nil
	}

	if params.URL == "" {
		return NewTextErrorResponse("URL parameter is required"), nil
	}

	if params.Format == "" {
		return NewTextErrorResponse("format parameter is required"), nil
	}

	format := strings.ToLower(params.Format)
	if format != "text" && format != "raw" {
		return NewTextErrorResponse("format must be 'text' or 'raw'"), nil
	}

	if !strings.HasPrefix(params.URL, "http://") && !strings.HasPrefix(params.URL, "https://") {
		return NewTextErrorResponse("URL must start with http:// or https://"), nil
	}

	// Permission check removed - tool executes directly

	// Handle timeout with context
	requestCtx := ctx
	if params.Timeout > 0 {
		maxTimeout := 120 // 2 minutes
		if params.Timeout > maxTimeout {
			params.Timeout = maxTimeout
		}
		var cancel context.CancelFunc
		requestCtx, cancel = context.WithTimeout(ctx, time.Duration(params.Timeout)*time.Second)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(requestCtx, "GET", params.URL, nil)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "crush/1.0")

	resp, err := t.client.Do(req)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return NewTextErrorResponse(fmt.Sprintf("Request failed with status code: %d", resp.StatusCode)), nil
	}

	maxSize := int64(5 * 1024 * 1024) // 5MB
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxSize))
	if err != nil {
		return NewTextErrorResponse("Failed to read response body: " + err.Error()), nil
	}

	// Truncate if content is too large before processing
	maxContentSize := 250 * 1024 // 250KB
	wasTruncated := false
	if len(body) > maxContentSize {
		body = body[:maxContentSize]
		wasTruncated = true
	}

	content := string(body)

	isValidUtf8 := utf8.ValidString(content)
	if !isValidUtf8 {
		return NewTextErrorResponse("Response content is not valid UTF-8"), nil
	}

	// For text format, normalize line endings and remove excessive blank lines
	if format == "text" {
		// Normalize line endings and remove excessive blank lines
		lines := strings.Split(content, "\n")
		var cleanLines []string
		blankCount := 0
		for _, line := range lines {
			// Keep indentation but remove trailing spaces
			trimmedRight := strings.TrimRight(line, " \t\r")
			if trimmedRight == "" {
				blankCount++
				// Allow maximum 2 consecutive blank lines
				if blankCount <= 2 {
					cleanLines = append(cleanLines, "")
				}
			} else {
				blankCount = 0
				cleanLines = append(cleanLines, trimmedRight)
			}
		}
		content = strings.Join(cleanLines, "\n")
	}

	if wasTruncated {
		content += fmt.Sprintf("\n\n[Content truncated to %d bytes]", maxContentSize)
	}

	return NewTextResponse(content), nil
}