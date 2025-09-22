package agents_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"gentica/ppt/agents"

	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai/openai"
	"github.com/openai/openai-go/option"
)

func ExampleNewTemplateDesignAgent() {
	// 初始化 Genkit
	oai := &openai.OpenAI{
		APIKey: os.Getenv("OPENAI_API_KEY"),
		Opts: []option.RequestOption{
			option.WithBaseURL(os.Getenv("OPENAI_BASE_URL")),
		},
	}

	g := genkit.Init(
		context.Background(),
		genkit.WithPlugins(oai),
	)

	// 创建工作目录
	workingDir := "/tmp/template_design_example"
	os.MkdirAll(workingDir, 0755)

	// 创建 TemplateDesignAgent
	templateAgent := agents.NewTemplateDesignAgent(g, workingDir)

	// 准备输入
	input := map[string]any{
		"style_description": "科技现代风格",
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		log.Fatal(err)
	}

	// 执行 Agent
	result, err := templateAgent.Run(context.Background(), string(inputJSON))
	if err != nil {
		log.Fatal(err)
	}

	// 解析结果
	designResult, err := agents.ParseTemplateDesignResult(result)
	if err != nil {
		log.Fatal(err)
	}

	// 输出结果
	fmt.Printf("Status: %s\n", designResult.Status)
	fmt.Printf("Directory: %s\n", designResult.DirectoryName)
	fmt.Printf("Summary: %s\n", designResult.Summary)

	if designResult.Status == "success" {
		fmt.Println("Generated template files:")
		for pageType, filePath := range designResult.FilePaths {
			fmt.Printf("  %s: %s\n", pageType, filePath)
		}
	}
}

func ExampleParseTemplateDesignOutput() {
	// 模拟的Agent输出
	agentOutput := `
生成的模板设计：

<html_block id="cover">
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="utf-8"/>
    <title>封面</title>
</head>
<body>
    <div class="slide-container">
        <h1>主标题</h1>
        <p>副标题</p>
    </div>
</body>
</html>
</html_block>

<html_block id="toc">
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="utf-8"/>
    <title>目录</title>
</head>
<body>
    <div class="slide-container">
        <h2>目录</h2>
        <ul>
            <li>第一章</li>
            <li>第二章</li>
        </ul>
    </div>
</body>
</html>
</html_block>

<html_block id="content">
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="utf-8"/>
    <title>内容页</title>
</head>
<body>
    <div class="slide-container">
        <h2>内容标题</h2>
        <p>内容正文</p>
    </div>
</body>
</html>
</html_block>

<html_block id="data">
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="utf-8"/>
    <title>数据页</title>
</head>
<body>
    <div class="slide-container">
        <h2>数据展示</h2>
        <table>
            <tr><td>数据1</td><td>值1</td></tr>
            <tr><td>数据2</td><td>值2</td></tr>
        </table>
    </div>
</body>
</html>
</html_block>

<html_block id="ending">
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="utf-8"/>
    <title>结尾页</title>
</head>
<body>
    <div class="slide-container">
        <h2>谢谢</h2>
        <p>感谢观看</p>
    </div>
</body>
</html>
</html_block>
`

	// 解析HTML输出
	templates, err := agents.ParseTemplateDesignOutput(agentOutput)
	if err != nil {
		log.Fatal(err)
	}

	// 检查各个页面是否存在
	fmt.Printf("Cover page exists: %v\n", len(templates.Cover) > 0)
	fmt.Printf("TOC page exists: %v\n", len(templates.TOC) > 0)
	fmt.Printf("Content page exists: %v\n", len(templates.Content) > 0)
	fmt.Printf("Data page exists: %v\n", len(templates.Data) > 0)
	fmt.Printf("Ending page exists: %v\n", len(templates.Ending) > 0)

	// Output:
	// Cover page exists: true
	// TOC page exists: true
	// Content page exists: true
	// Data page exists: true
	// Ending page exists: true
}