package agents

import (
	"context"
	"encoding/json"
	"fmt"

	"gentica/tools"
)

// AddToResultSetTool 添加高质量结果到结果集的工具
type AddToResultSetTool struct {
	storage *SearchStorage
}

// AddToResultSetParams 添加结果的参数
type AddToResultSetParams struct {
	Title string `json:"title"`
}

func NewAddToResultSetTool(storage *SearchStorage) tools.BaseTool {
	return &AddToResultSetTool{storage: storage}
}

func (t *AddToResultSetTool) Name() string {
	return "add_to_result_set"
}

func (t *AddToResultSetTool) Info() tools.ToolInfo {
	return tools.ToolInfo{
		Name:        "add_to_result_set",
		Description: "Add a high-quality search result to the final result set by its title. The result must already exist in the search results store.",
		Parameters: map[string]any{
			"title": map[string]any{
				"type":        "string",
				"description": "The title of the search result to add to the quality set",
			},
		},
		Required: []string{"title"},
	}
}

func (t *AddToResultSetTool) Run(ctx context.Context, call tools.ToolCall) (tools.ToolResponse, error) {
	var params AddToResultSetParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return tools.NewTextErrorResponse("Failed to parse parameters: " + err.Error()), nil
	}

	if params.Title == "" {
		return tools.NewTextErrorResponse("Title parameter is required"), nil
	}

	// 尝试添加到质量集
	if t.storage.AddToQualitySet(params.Title) {
		result, _ := t.storage.GetSearchResultByTitle(params.Title)
		return tools.NewTextResponse(fmt.Sprintf("Successfully added '%s' (score: %.1f) to quality result set", params.Title, result.Score)), nil
	}

	return tools.NewTextErrorResponse(fmt.Sprintf("Failed to add '%s' - either not found in search results or already in quality set", params.Title)), nil
}

// ViewResultSetTool 查看当前结果集的工具
type ViewResultSetTool struct {
	storage *SearchStorage
}

func NewViewResultSetTool(storage *SearchStorage) tools.BaseTool {
	return &ViewResultSetTool{storage: storage}
}

func (t *ViewResultSetTool) Name() string {
	return "view_result_set"
}

func (t *ViewResultSetTool) Info() tools.ToolInfo {
	return tools.ToolInfo{
		Name:        "view_result_set",
		Description: "View the current quality result set and search statistics",
		Parameters:  map[string]any{},
		Required:    []string{},
	}
}

func (t *ViewResultSetTool) Run(ctx context.Context, call tools.ToolCall) (tools.ToolResponse, error) {
	qualityResults := t.storage.GetQualityResults()
	totalSearches, searchRounds, qualityCount := t.storage.GetStats()

	response := struct {
		QualityResults []QualityResult `json:"quality_results"`
		Statistics     struct {
			TotalSearches int `json:"total_searches"`
			SearchRounds  int `json:"search_rounds"`
			QualityCount  int `json:"quality_count"`
		} `json:"statistics"`
	}{
		QualityResults: qualityResults,
		Statistics: struct {
			TotalSearches int `json:"total_searches"`
			SearchRounds  int `json:"search_rounds"`
			QualityCount  int `json:"quality_count"`
		}{
			TotalSearches: totalSearches,
			SearchRounds:  searchRounds,
			QualityCount:  qualityCount,
		},
	}

	responseJSON, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return tools.NewTextErrorResponse("Failed to format response: " + err.Error()), nil
	}

	return tools.NewTextResponse(string(responseJSON)), nil
}

// ClearResultSetTool 清空结果集的工具
type ClearResultSetTool struct {
	storage *SearchStorage
}

func NewClearResultSetTool(storage *SearchStorage) tools.BaseTool {
	return &ClearResultSetTool{storage: storage}
}

func (t *ClearResultSetTool) Name() string {
	return "clear_result_set"
}

func (t *ClearResultSetTool) Info() tools.ToolInfo {
	return tools.ToolInfo{
		Name:        "clear_result_set",
		Description: "Clear the quality result set (does not clear search results)",
		Parameters:  map[string]any{},
		Required:    []string{},
	}
}

func (t *ClearResultSetTool) Run(ctx context.Context, call tools.ToolCall) (tools.ToolResponse, error) {
	t.storage.ClearQualityResults()
	return tools.NewTextResponse("Quality result set has been cleared"), nil
}

// ViewAllResultsTool 查看所有搜索结果的工具（包括未筛选的）
type ViewAllResultsTool struct {
	storage *SearchStorage
}

func NewViewAllResultsTool(storage *SearchStorage) tools.BaseTool {
	return &ViewAllResultsTool{storage: storage}
}

func (t *ViewAllResultsTool) Name() string {
	return "view_all_results"
}

func (t *ViewAllResultsTool) Info() tools.ToolInfo {
	return tools.ToolInfo{
		Name:        "view_all_results",
		Description: "View all search results including those not added to quality set",
		Parameters:  map[string]any{},
		Required:    []string{},
	}
}

func (t *ViewAllResultsTool) Run(ctx context.Context, call tools.ToolCall) (tools.ToolResponse, error) {
	allResults := t.storage.GetAllSearchResults()

	// 转换为列表格式便于查看
	type ResultSummary struct {
		Title string  `json:"title"`
		Score float64 `json:"score"`
		Link  string  `json:"link"`
		Round int     `json:"round"`
	}

	var results []ResultSummary
	for _, result := range allResults {
		results = append(results, ResultSummary{
			Title: result.Title,
			Score: result.Score,
			Link:  result.Link,
			Round: result.SearchRound,
		})
	}

	responseJSON, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return tools.NewTextErrorResponse("Failed to format response: " + err.Error()), nil
	}

	return tools.NewTextResponse(string(responseJSON)), nil
}
