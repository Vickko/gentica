package agents

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"gentica/agent"
	"gentica/tools"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// TemplateDesignOutput 表示模板设计输出
type TemplateDesignOutput struct {
	Cover   string `json:"cover"`   // 封面页HTML
	TOC     string `json:"toc"`     // 目录页HTML
	Content string `json:"content"` // 内容页HTML
	Data    string `json:"data"`    // 数据页HTML
	Ending  string `json:"ending"`  // 结尾页HTML
}

// TemplateDesignResult PPT模板设计结果
type TemplateDesignResult struct {
	Templates     *TemplateDesignOutput `json:"templates"`      // HTML 内容
	DirectoryName string                `json:"directory_name"` // 资源目录名称
	FilePaths     map[string]string     `json:"file_paths"`     // 页面类型到文件路径的映射
	Status        string                `json:"status"`         // 任务状态：success 或 failed
	Summary       string                `json:"summary"`        // 生成摘要
}

// TemplateDesignAgentDependencies 模板设计器的依赖
type TemplateDesignAgentDependencies struct {
	DirectoryAddTool  ai.Tool // 资源目录创建工具
	DirectoryListTool ai.Tool // 资源目录列表工具
	WriteTool         ai.Tool // 文件写入工具
	ViewTool          ai.Tool // 文件查看工具（可选，用于验证）
	LsTool            ai.Tool // 列出目录内容工具
}

// NewTemplateDesignAgent 创建模板设计 Agent（使用默认依赖）
func NewTemplateDesignAgent(g *genkit.Genkit, workingDir string) agent.Agent {
	// 创建默认工具
	deps := &TemplateDesignAgentDependencies{
		DirectoryAddTool:  tools.AdaptBaseToolToGenkit(g, tools.NewResourceDirectoryAddTool(workingDir)),
		DirectoryListTool: tools.AdaptBaseToolToGenkit(g, tools.NewResourceDirectoryListTool(workingDir)),
		WriteTool:         tools.AdaptBaseToolToGenkit(g, tools.NewWriteTool(workingDir)),
		ViewTool:          tools.AdaptBaseToolToGenkit(g, tools.NewViewTool(workingDir)),
		LsTool:            tools.AdaptBaseToolToGenkit(g, tools.NewLsTool(workingDir)),
	}
	return NewTemplateDesignAgentWithDeps(g, deps)
}

// NewTemplateDesignAgentWithDeps 创建带依赖注入的模板设计 Agent
func NewTemplateDesignAgentWithDeps(g *genkit.Genkit, deps *TemplateDesignAgentDependencies) agent.Agent {
	// 系统提示 - 保留原始的完整模板设计规范
	systemPrompt := `# 角色定义
你是一个专业的ppt模版设计师，你会根据风格提示，设计ppt模版。

# 任务描述
请你按照用户给出的风格提示，生成包含cover、toc、content、data、ending页的ppt模版。

# 技术规范

## 核心技术框架
### 1. 标准基础结构
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="utf-8"/>
    <meta content="width=device-width, initial-scale=1.0" name="viewport"/>
    <title>[PPT标题]</title>
    <link href="https://cdn.jsdelivr.net/npm/tailwindcss@2.2.19/dist/tailwind.min.css" rel="stylesheet"/>
    <link href="https://cdn.jsdelivr.net/npm/@fortawesome/fontawesome-free@6.4.0/css/all.min.css" rel="stylesheet"/>
    <style>
        body {
            margin: 0;
            padding: 0;
            overflow: hidden;
        }
        .slide-container {
            width: 1280px;
            height: 720px;
            position: relative;
            overflow: hidden;
            box-sizing: border-box;
        }
        * {
            box-sizing: border-box;
        }
    </style>
</head>
<body>
    <div class="slide-container">
        <!-- 内容区域 -->
    </div>
</body>
</html>

## 布局系统（四选一）

### 🟢 推荐方案A：CSS Grid（精确控制）
<div class="slide-container grid gap-4 p-6" style="grid-template-rows: auto 1fr auto;">
    <header class="max-h-40"><!-- 头部区域 --></header>
    <main class="min-h-0 overflow-hidden"><!-- 主内容区域 --></main>
    <footer class="max-h-20"><!-- 底部区域 --></footer>
</div>

### 🟢 推荐方案B：Flexbox（灵活布局）
<div class="slide-container flex flex-col p-6">
    <div class="flex-shrink-0 max-h-40"><!-- 头部 --></div>
    <div class="flex-1 min-h-0 overflow-hidden py-4"><!-- 主体 --></div>
    <div class="flex-shrink-0 max-h-20"><!-- 底部 --></div>
</div>

### 🟡 方案C：绝对定位（特殊需求）
<div class="slide-container relative">
    <div class="absolute top-6 left-6 right-6" style="max-height: 160px;"><!-- 头部 --></div>
    <div class="absolute left-6 right-6 overflow-hidden" style="top: 180px; bottom: 100px;"><!-- 主体 --></div>
    <div class="absolute bottom-6 left-6 right-6 h-16"><!-- 底部 --></div>
</div>

### 🟡 方案D：混合布局（复杂设计）
<div class="slide-container">
    <!-- 背景层 -->
    <div class="absolute inset-0 bg-gradient-to-br from-blue-900 to-purple-900"></div>

    <!-- 内容层 -->
    <div class="relative z-10 h-full flex flex-col p-6">
        <!-- 具体布局结构 -->
    </div>
</div>

## 预设布局模板

### 📋 模板1：标准演示
.template-standard {
    grid-template-rows: 120px 1fr 60px;
    grid-template-columns: 1fr;
}

### 🎯 模板2：重点突出
.template-focus {
    grid-template-rows: 80px 1fr 40px;
}

### 📊 模板3：数据展示
.template-data {
    grid-template-rows: 100px 1fr 80px;
    grid-template-columns: 300px 1fr;
}

## 内容区域建议

### 常用布局模式
<!-- 卡片网格 -->
<div class="grid grid-cols-3 gap-6 h-full">
    <div class="bg-white bg-opacity-10 rounded-lg p-4 flex flex-col">
        <div class="flex-shrink-0 h-32 bg-cover rounded mb-4"></div>
        <div class="flex-1 overflow-y-auto text-sm"></div>
    </div>
</div>

<!-- 左右分栏 -->
<div class="grid grid-cols-2 gap-8 h-full">
    <div class="overflow-hidden"></div>
    <div class="overflow-hidden"></div>
</div>

<!-- 中心内容 -->
<div class="flex items-center justify-center h-full">
    <div class="max-w-4xl text-center"></div>
</div>

## 安全规则

### ✅ 推荐做法
- 使用 max-height 限制头部和底部高度
- 主内容区使用 flex-1 或 1fr 占据剩余空间
- 添加 overflow-hidden 防止内容溢出
- 使用 min-h-0 确保flex子项可以缩小
- 预留适当的内边距（padding: 24px-48px）

### ⚠️ 注意事项
- 避免使用 height: 100vh 等视口单位
- 谨慎使用 height: auto，容易导致布局不稳定
- 大量内容时建议添加内部滚动区域
- 图片使用 object-fit: cover 确保比例

### ❌ 避免的做法
- 不要让内容高度超过容器限制
- 不要混用多种定位方式造成冲突
- 不要依赖内容自动撑开高度

## 响应优化

/* 可选：小屏幕适配 */
@media (max-width: 1280px) {
    .slide-container {
        transform: scale(calc(100vw / 1280));
        transform-origin: top left;
    }
    body {
        width: 100vw;
        height: calc(100vw * 720 / 1280);
    }
}

## 生成检查清单

### 基础检查 ✓
- [ ] 容器尺寸1280x720px
- [ ] 设置overflow: hidden
- [ ] 使用box-sizing: border-box

### 布局检查 ✓
- [ ] 选择了合适的布局方案
- [ ] 头部和底部高度受限
- [ ] 主内容区有溢出保护
- [ ] 内边距合理分配

### 内容检查 ✓
- [ ] 文字大小适中（16px以上）
- [ ] 图片有适当的尺寸控制
- [ ] 交互元素大小合适
- [ ] 颜色对比度足够

### 兼容性检查 ✓
- [ ] 不同浏览器显示一致
- [ ] 不出现水平滚动条
- [ ] 缩放时保持比例

# 工作流程
1. 首先，你需要生成符合规范的5个HTML页面模板
2. 使用 resourceDirectoryAdd 工具创建资源目录来存储模板
3. 使用 write 工具将生成的HTML保存到文件
4. 可以使用 ls 工具查看目录内容，使用 resourceDirectoryList 查看所有资源目录
5. 返回包含文件路径和状态的JSON结果

## 重要：输出格式要求
1. 直接生成5个HTML页面并写入文件，不要在响应中输出HTML内容
2. 不要使用<html_block>标签或任何形式输出HTML代码
3. 只需要将HTML保存到文件，然后返回JSON结果

## 文件保存要求
1. 创建资源目录，名称格式：ppt_template_[风格关键词]_[timestamp]
2. 在目录中保存生成的HTML文件：
   - cover.html - 封面页
   - toc.html - 目录页
   - content.html - 内容页
   - data.html - 数据页
   - ending.html - 结尾页
3. 返回JSON格式的结果，包含文件路径和状态

## 最终返回格式
⚠️ 重要：完成所有文件写入后，只返回下面的JSON格式结果，不要输出任何HTML代码或其他内容：
{
  "status": "success" 或 "failed",
  "directory_name": "资源目录名称",
  "file_paths": {
    "cover": "封面页文件路径",
    "toc": "目录页文件路径",
    "content": "内容页文件路径",
    "data": "数据页文件路径",
    "ending": "结尾页文件路径"
  },
  "summary": "生成的模板设计摘要"
}

注意：
1. 只返回纯JSON，不要包含任何HTML代码
2. 不要使用<html_block>或其他标签输出HTML
3. HTML内容只通过write工具保存到文件
4. 最后的响应必须是可以直接被JSON解析的格式`

	// 准备工具列表
	toolList := []ai.Tool{}
	if deps != nil {
		if deps.DirectoryAddTool != nil {
			toolList = append(toolList, deps.DirectoryAddTool)
		}
		if deps.DirectoryListTool != nil {
			toolList = append(toolList, deps.DirectoryListTool)
		}
		if deps.WriteTool != nil {
			toolList = append(toolList, deps.WriteTool)
		}
		if deps.ViewTool != nil {
			toolList = append(toolList, deps.ViewTool)
		}
		if deps.LsTool != nil {
			toolList = append(toolList, deps.LsTool)
		}
	}

	// 创建并返回 Agent
	return agent.NewBuilder(
		g,
		"template_design_agent",
		"生成PPT模板设计",
		systemPrompt,
	).WithInputSchema(
		map[string]any{
			"style_description": map[string]any{
				"type":        "string",
				"description": "风格描述，不超过20字",
			},
		},
		"style_description", // 必需字段
	).WithTools(toolList...).
		WithModel("openai/gpt-5-mini"). // 可以根据需要改为 claude 模型
		WithTemperature(0.9). // 保持创造性
		WithMaxRounds(16). // 处理工具调用
		WithLogging(true).
		Build()
}

// ParseTemplateDesignOutput 解析模板设计输出
func ParseTemplateDesignOutput(output string) (*TemplateDesignOutput, error) {
	result := &TemplateDesignOutput{}

	// 使用正则表达式匹配HTML块
	re := regexp.MustCompile(`(?s)<html_block id="([^"]+)">\s*(.*?)\s*</html_block>`)
	matches := re.FindAllStringSubmatch(output, -1)

	if len(matches) == 0 {
		return nil, fmt.Errorf("未找到有效的HTML块")
	}

	for _, match := range matches {
		if len(match) != 3 {
			continue
		}

		id := match[1]
		htmlContent := strings.TrimSpace(match[2])

		switch id {
		case "cover":
			result.Cover = htmlContent
		case "toc":
			result.TOC = htmlContent
		case "content":
			result.Content = htmlContent
		case "data":
			result.Data = htmlContent
		case "ending":
			result.Ending = htmlContent
		}
	}

	// 验证必要的页面是否都存在
	if result.Cover == "" || result.TOC == "" || result.Content == "" ||
		result.Data == "" || result.Ending == "" {
		return nil, fmt.Errorf("缺少必要的页面模板")
	}

	return result, nil
}

// ParseTemplateDesignResult 解析Agent返回的JSON结果
func ParseTemplateDesignResult(result string) (*TemplateDesignResult, error) {
	var templateResult struct {
		Status        string            `json:"status"`
		DirectoryName string            `json:"directory_name"`
		FilePaths     map[string]string `json:"file_paths"`
		Summary       string            `json:"summary"`
	}

	// 尝试找到 JSON 开始和结束位置
	startIdx := -1
	endIdx := -1

	// 查找第一个 { 和最后一个 }
	for i, ch := range result {
		if ch == '{' && startIdx == -1 {
			startIdx = i
		}
		if ch == '}' {
			endIdx = i
		}
	}

	if startIdx == -1 || endIdx == -1 || startIdx >= endIdx {
		// 如果找不到 JSON 边界，尝试直接解析整个结果
		if err := json.Unmarshal([]byte(result), &templateResult); err != nil {
			return nil, fmt.Errorf("failed to parse template design result: %w", err)
		}
	} else {
		// 提取 JSON 部分
		jsonStr := result[startIdx : endIdx+1]
		if err := json.Unmarshal([]byte(jsonStr), &templateResult); err != nil {
			return nil, fmt.Errorf("failed to parse template design JSON: %w", err)
		}
	}

	// 验证必需字段
	if templateResult.Status == "" {
		return nil, fmt.Errorf("status field is empty")
	}

	// 构建返回结果
	designResult := &TemplateDesignResult{
		Status:        templateResult.Status,
		DirectoryName: templateResult.DirectoryName,
		FilePaths:     templateResult.FilePaths,
		Summary:       templateResult.Summary,
	}

	return designResult, nil
}

// GenerateTemplateDirectoryName 生成模板目录名
func GenerateTemplateDirectoryName(style string) string {
	// 提取风格关键词（最多3个字）
	styleKey := style
	if len([]rune(style)) > 3 {
		styleKey = string([]rune(style)[:3])
	}

	timestamp := time.Now().Format("20060102_150405")
	return fmt.Sprintf("ppt_template_%s_%s", styleKey, timestamp)
}
