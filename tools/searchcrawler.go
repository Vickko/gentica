package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"gentica/clients/serper"
)

type SearchCrawlerParams struct {
	Query      string `json:"query"`
	ResultNum  int    `json:"result_num,omitempty"`
	SavePath   string `json:"save_path"`
}

type searchCrawlerTool struct {
	serperClient *serper.Client
	workingDir   string
}

const (
	SearchCrawlerToolName        = "searchCrawler"
	searchCrawlerToolDescription = `Searches for content using keywords and crawls the results to save as markdown files.

WHEN TO USE THIS TOOL:
- Use when you need to search and collect content from multiple web pages
- Helpful for research, gathering information on a topic
- Useful for creating a local knowledge base from search results
- Great for content aggregation and analysis

HOW TO USE:
- Provide a search query (keywords or phrase)
- Optionally specify the number of results (default: 8)
- Provide a filesystem path to save the crawled content
- The tool will search, crawl each result, and save as markdown

FEATURES:
- Concurrent crawling of multiple URLs for better performance
- Automatic filename sanitization using page titles
- Detailed statistics on success/failure rates
- Markdown format preservation for better readability
- Creates directory if it doesn't exist

LIMITATIONS:
- Some websites may block crawling attempts
- Rate limiting may apply depending on configuration
- Network errors may cause individual crawl failures

TIPS:
- Use specific keywords for better search results
- Check the save path has write permissions
- Monitor the statistics to identify crawling issues`
)

func NewSearchCrawlerTool(workingDir string) BaseTool {
	return &searchCrawlerTool{
		serperClient: serper.NewClient(),
		workingDir:   workingDir,
	}
}

func (t *searchCrawlerTool) Name() string {
	return SearchCrawlerToolName
}

func (t *searchCrawlerTool) Info() ToolInfo {
	return ToolInfo{
		Name:        SearchCrawlerToolName,
		Description: searchCrawlerToolDescription,
		Parameters: map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "The search keywords or phrase",
			},
			"result_num": map[string]any{
				"type":        "number",
				"description": "Number of results to fetch and crawl (default: 8)",
			},
			"save_path": map[string]any{
				"type":        "string",
				"description": "Filesystem path to save the crawled markdown files",
			},
		},
		Required: []string{"query", "save_path"},
	}
}

func (t *searchCrawlerTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params SearchCrawlerParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse("Failed to parse search crawler parameters: " + err.Error()), nil
	}

	// Validate parameters
	if params.Query == "" {
		return NewTextErrorResponse("query parameter is required"), nil
	}

	if params.SavePath == "" {
		return NewTextErrorResponse("save_path parameter is required"), nil
	}

	// Set default result number
	if params.ResultNum <= 0 {
		params.ResultNum = 8
	}

	// Convert relative path to absolute path
	savePath := params.SavePath
	if !filepath.IsAbs(savePath) {
		savePath = filepath.Join(t.workingDir, savePath)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(savePath, 0755); err != nil {
		return NewTextErrorResponse(fmt.Sprintf("Failed to create directory %s: %v", savePath, err)), nil
	}

	// Perform search
	searchReq := &serper.SearchRequest{
		Query: params.Query,
		Num:   params.ResultNum,
	}

	searchResp, err := t.serperClient.Search(ctx, searchReq)
	if err != nil {
		return NewTextErrorResponse(fmt.Sprintf("Search failed: %v", err)), nil
	}

	// Extract URLs and titles from organic results
	type crawlTarget struct {
		URL   string
		Title string
	}

	targets := make([]crawlTarget, 0, len(searchResp.Organic))
	for _, result := range searchResp.Organic {
		if result.Link != "" {
			targets = append(targets, crawlTarget{
				URL:   result.Link,
				Title: result.Title,
			})
			if len(targets) >= params.ResultNum {
				break
			}
		}
	}

	if len(targets) == 0 {
		return NewTextErrorResponse("No search results found"), nil
	}

	// Crawl URLs concurrently
	type crawlResult struct {
		target crawlTarget
		err    error
	}

	results := make(chan crawlResult, len(targets))
	var wg sync.WaitGroup

	for _, target := range targets {
		wg.Add(1)
		go func(tgt crawlTarget) {
			defer wg.Done()

			// Crawl the webpage
			webpageResp, err := t.serperClient.ScrapeWebpageSimple(ctx, tgt.URL)
			if err != nil {
				results <- crawlResult{target: tgt, err: err}
				return
			}

			// Save markdown content
			content := webpageResp.Markdown
			if content == "" {
				content = webpageResp.Text
			}

			if content == "" {
				results <- crawlResult{target: tgt, err: fmt.Errorf("no content retrieved")}
				return
			}

			// Generate filename
			filename := sanitizeFilename(tgt.Title)
			if filename == "" {
				filename = sanitizeFilename(tgt.URL)
			}
			if filename == "" {
				filename = fmt.Sprintf("page_%d", len(results)+1)
			}
			filename = filename + ".md"

			filepath := filepath.Join(savePath, filename)

			// Write content to file
			fileContent := fmt.Sprintf("# %s\n\nSource: %s\n\n---\n\n%s",
				tgt.Title, tgt.URL, content)

			err = os.WriteFile(filepath, []byte(fileContent), 0644)
			if err != nil {
				results <- crawlResult{target: tgt, err: fmt.Errorf("failed to save file: %v", err)}
				return
			}

			results <- crawlResult{target: tgt, err: nil}
		}(target)
	}

	// Wait for all crawls to complete
	wg.Wait()
	close(results)

	// Collect results
	var successCount, failureCount int
	var errors []string
	var savedResults []string

	for result := range results {
		if result.err != nil {
			failureCount++
			errors = append(errors, fmt.Sprintf("%s: %v", result.target.URL, result.err))
		} else {
			successCount++
			savedResults = append(savedResults, fmt.Sprintf("- %s", result.target.Title))
		}
	}

	// Build response message
	var response strings.Builder

	totalResults := len(targets)

	if successCount == 0 && failureCount > 0 {
		// All crawls failed
		response.WriteString(fmt.Sprintf("[WARNING] %d个结果爬取全部失败，报错信息：\n", failureCount))
		for _, errMsg := range errors {
			response.WriteString(fmt.Sprintf("  - %s\n", errMsg))
		}
		response.WriteString("\n搜索结果概览：\n")
	} else {
		// At least some crawls succeeded
		response.WriteString(fmt.Sprintf("已得到%d个搜索结果，其中%d个结果正文已保存至指定路径，",
			totalResults, successCount))

		if failureCount > 0 {
			response.WriteString(fmt.Sprintf("%d个失败，报错信息：\n", failureCount))
			for _, errMsg := range errors {
				response.WriteString(fmt.Sprintf("  - %s\n", errMsg))
			}
			response.WriteString("\n")
		} else {
			response.WriteString("0个失败。\n\n")
		}

		response.WriteString("搜索结果概览：\n")
	}

	// Add search results overview
	for i, target := range targets {
		if i >= params.ResultNum {
			break
		}
		response.WriteString(fmt.Sprintf("%d. %s\n   URL: %s\n", i+1, target.Title, target.URL))
	}

	if successCount > 0 {
		response.WriteString(fmt.Sprintf("\n已成功保存的文件：\n%s", strings.Join(savedResults, "\n")))
	}

	return NewTextResponse(response.String()), nil
}

// sanitizeFilename removes invalid characters from filename
func sanitizeFilename(name string) string {
	// Remove HTML tags if any
	re := regexp.MustCompile(`<[^>]+>`)
	name = re.ReplaceAllString(name, "")

	// Replace invalid filename characters
	invalidChars := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	name = invalidChars.ReplaceAllString(name, "_")

	// Trim spaces and dots
	name = strings.TrimSpace(name)
	name = strings.Trim(name, ".")

	// Trim underscores that might be left from replacing invalid chars
	name = strings.Trim(name, "_")

	// Limit length
	if len(name) > 200 {
		name = name[:200]
	}

	// If empty after sanitization, return empty
	if name == "" {
		return ""
	}

	return name
}