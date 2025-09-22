package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/chromedp"
)

// htmlSizeTool 验证HTML尺寸的工具
type htmlSizeTool struct {
	basePath string
}

// HtmlSizeInput HTML尺寸验证输入
type HtmlSizeInput struct {
	FilePath string `json:"file_path"` // HTML文件的相对或绝对路径
}

// HtmlSizeOutput HTML尺寸验证输出
type HtmlSizeOutput struct {
	Width      int    `json:"width"`       // 宽度像素
	Height     int    `json:"height"`      // 高度像素
	Valid      bool   `json:"valid"`       // 是否符合1280x720规范
	Message    string `json:"message"`     // 验证消息
	Suggestion string `json:"suggestion"`  // 调整建议
}

const (
	HtmlSizeToolName = "html_size"
	htmlSizeDescription = `HTML size validation tool that checks if rendered HTML meets the exact 1280x720 specification.

WHEN TO USE THIS TOOL:
- Use when you need to validate HTML page dimensions
- Essential for PPT page generation to ensure correct sizing
- Helps iterate and fix HTML layouts that don't meet specifications

HOW TO USE:
- Provide the path to the HTML file to validate
- The tool will render the HTML using a headless browser
- Returns the actual dimensions and validation result

FEATURES:
- Uses chromedp to render HTML in a real browser environment
- Requires .slide-container element to be present
- Strict validation: must be exactly 1280x720
- Provides adjustment suggestions when dimensions are incorrect

VALIDATION RULES:
- Width must be exactly 1280px
- Height must be exactly 720px (no tolerance)
- Requires .slide-container element

OUTPUT:
- Width and height in pixels
- Valid boolean indicating if dimensions meet specifications
- Message describing the current size
- Suggestion for adjustments if invalid`
)

// NewHtmlSizeTool 创建HTML尺寸验证工具
func NewHtmlSizeTool(basePath string) BaseTool {
	return &htmlSizeTool{
		basePath: basePath,
	}
}

// Name 返回工具名称
func (t *htmlSizeTool) Name() string {
	return HtmlSizeToolName
}

// Info 返回工具信息
func (t *htmlSizeTool) Info() ToolInfo {
	return ToolInfo{
		Name:        HtmlSizeToolName,
		Description: htmlSizeDescription,
		Parameters: map[string]any{
			"file_path": map[string]any{
				"type":        "string",
				"description": "The path to the HTML file to validate",
			},
		},
		Required: []string{"file_path"},
	}
}

// Run 执行HTML尺寸验证
func (t *htmlSizeTool) Run(ctx context.Context, params ToolCall) (ToolResponse, error) {
	// 解析输入参数
	var input HtmlSizeInput
	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		return ToolResponse{
			Type:    ToolResponseTypeText,
			Content: fmt.Sprintf("Failed to parse input: %v", err),
			IsError: true,
		}, nil
	}

	// 执行验证
	output, err := t.execute(input)
	if err != nil {
		return ToolResponse{
			Type:    ToolResponseTypeText,
			Content: fmt.Sprintf("Validation error: %v", err),
			IsError: true,
		}, nil
	}

	// 序列化输出
	outputJSON, err := json.Marshal(output)
	if err != nil {
		return ToolResponse{
			Type:    ToolResponseTypeText,
			Content: fmt.Sprintf("Failed to serialize output: %v", err),
			IsError: true,
		}, nil
	}

	return ToolResponse{
		Type:    ToolResponseTypeText,
		Content: string(outputJSON),
		IsError: false,
	}, nil
}

// InputSchema 返回输入模式
func (t *htmlSizeTool) InputSchema() any {
	return HtmlSizeInput{}
}

// execute 内部执行方法（保持原有逻辑）
func (t *htmlSizeTool) execute(input HtmlSizeInput) (HtmlSizeOutput, error) {
	// 解析文件路径
	filePath := input.FilePath
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(t.basePath, filePath)
	}

	// 读取HTML内容
	htmlContent, err := os.ReadFile(filePath)
	if err != nil {
		return HtmlSizeOutput{
			Valid:   false,
			Message: fmt.Sprintf("无法读取文件: %v", err),
		}, nil
	}

	// 使用chromedp验证尺寸
	width, height, err := t.checkSize(string(htmlContent))
	if err != nil {
		return HtmlSizeOutput{
			Valid:   false,
			Message: fmt.Sprintf("验证失败: %v", err),
		}, nil
	}

	// 判断是否符合规范（必须严格是1280x720）
	validWidth := width == 1280
	validHeight := height == 720
	valid := validWidth && validHeight

	output := HtmlSizeOutput{
		Width:   width,
		Height:  height,
		Valid:   valid,
		Message: fmt.Sprintf("当前尺寸: %dx%d", width, height),
	}

	// 添加调整建议
	if !valid {
		if !validWidth && !validHeight {
			output.Suggestion = fmt.Sprintf("尺寸应为1280x720，当前为%dx%d", width, height)
		} else if !validWidth {
			output.Suggestion = fmt.Sprintf("宽度应为1280px，当前为%dpx", width)
		} else if !validHeight {
			output.Suggestion = fmt.Sprintf("高度应为720px，当前为%dpx", height)
		}
	} else {
		output.Message += " - 符合规范"
	}

	return output, nil
}

// checkSize 使用chromedp检查HTML尺寸
func (t *htmlSizeTool) checkSize(htmlContent string) (int, int, error) {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// 设置超时
	ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var result []interface{}
	err := chromedp.Run(ctx,
		// 导航到空白页面
		chromedp.Navigate("about:blank"),
		// 设置页面HTML内容
		chromedp.ActionFunc(func(ctx context.Context) error {
			return chromedp.Evaluate(fmt.Sprintf(`document.documentElement.innerHTML = %q`, htmlContent), nil).Do(ctx)
		}),
		// 等待slide-container元素出现
		chromedp.WaitVisible(".slide-container"),
		// 获取尺寸
		chromedp.Evaluate(`
			const c = document.querySelector('.slide-container');
			[c.scrollWidth, c.scrollHeight];
		`, &result),
	)

	if err != nil {
		return 0, 0, err
	}

	if len(result) < 2 {
		return 0, 0, fmt.Errorf("无法获取尺寸信息")
	}

	width := int(result[0].(float64))
	height := int(result[1].(float64))

	return width, height, nil
}