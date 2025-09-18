package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// AdaptBaseToolToGenkit 将 BaseTool 转换为 Genkit AI Tool
// 从 BaseTool 接口直接提取 name, schema (parameters), 和执行函数
func AdaptBaseToolToGenkit(g *genkit.Genkit, tool BaseTool) ai.Tool {
	info := tool.Info()

	// 将 BaseTool 的 Parameters 转换为 Genkit 需要的 inputSchema
	inputSchema := map[string]any{
		"type": "object",
		"properties": info.Parameters,
		"required": info.Required,
	}

	// 使用 DefineToolWithInputSchema，因为我们的输入是动态的 JSON
	return genkit.DefineToolWithInputSchema(g, info.Name, info.Description, inputSchema,
		func(ctx *ai.ToolContext, input any) (string, error) {
			// 将 Genkit 的输入转换回 ToolCall 格式
			inputBytes, err := json.Marshal(input)
			if err != nil {
				return "", fmt.Errorf("failed to marshal input: %w", err)
			}

			toolCall := ToolCall{
				Name:  tool.Name(),
				Input: string(inputBytes),
			}

			// 调用原始 BaseTool 的 Run 方法
			// 注意：这里使用 context.Background()，实际使用时可能需要从 ToolContext 提取
			response, err := tool.Run(context.Background(), toolCall)
			if err != nil {
				// 真正的系统错误才返回 error
				return "", err
			}

			// 工具执行的错误（如文件不存在）也作为正常输出返回给 LLM
			// 让 LLM 自己决定如何处理这个错误
			if response.IsError {
				// 返回错误信息，但添加前缀让 LLM 知道这是错误
				return fmt.Sprintf("Error: %s", response.Content), nil
			}

			// 返回内容
			// 注意：这里丢失了 metadata，如果需要可以考虑其他方案
			return response.Content, nil
		},
	)
}

// BatchAdaptTools 批量转换多个 BaseTool 到 Genkit Tools
func BatchAdaptTools(g *genkit.Genkit, tools ...BaseTool) []ai.Tool {
	genkitTools := make([]ai.Tool, len(tools))
	for i, tool := range tools {
		genkitTools[i] = AdaptBaseToolToGenkit(g, tool)
	}
	return genkitTools
}

// 示例：如何使用适配器
// func RegisterExistingTools(g *genkit.Genkit, workingDir string) []ai.Tool {
//     existingTools := []BaseTool{
//         NewViewTool(workingDir),
//         NewEditTool(workingDir),
//         NewWriteTool(workingDir),
//         NewGlobTool(workingDir),
//         NewGrepTool(workingDir),
//         // ... 其他工具
//     }
//
//     return BatchAdaptTools(g, existingTools...)
// }