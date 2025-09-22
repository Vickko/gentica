package agent

import (
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core"
)

// Logger 定义日志接口
type Logger interface {
	Logf(format string, args ...interface{})
}

// StandardLogger 使用标准 log 包的日志器
type StandardLogger struct{}

func (l *StandardLogger) Logf(format string, args ...interface{}) {
	log.Printf(format, args...)
}

// TestLogger 用于测试的日志器
type TestLogger struct {
	t *testing.T
}

func (l *TestLogger) Logf(format string, args ...interface{}) {
	l.t.Logf(format, args...)
}

// CreateConversationLogger 创建一个对话日志中间件
// 可以传入 nil 使用默认的标准日志器，或传入 *testing.T 使用测试日志器
func CreateConversationLogger(logger interface{}) func(core.StreamingFunc[*ai.ModelRequest, *ai.ModelResponse, *ai.ModelResponseChunk]) core.StreamingFunc[*ai.ModelRequest, *ai.ModelResponse, *ai.ModelResponseChunk] {
	var loggerImpl Logger
	switch v := logger.(type) {
	case *testing.T:
		loggerImpl = &TestLogger{t: v}
	case Logger:
		loggerImpl = v
	default:
		loggerImpl = &StandardLogger{}
	}

	var roundCounter int
	var lastMessageCount int

	return func(next core.StreamingFunc[*ai.ModelRequest, *ai.ModelResponse, *ai.ModelResponseChunk]) core.StreamingFunc[*ai.ModelRequest, *ai.ModelResponse, *ai.ModelResponseChunk] {
		return func(ctx context.Context, req *ai.ModelRequest, cb core.StreamCallback[*ai.ModelResponseChunk]) (*ai.ModelResponse, error) {
			roundCounter++

			// ========== 请求阶段 ==========
			loggerImpl.Logf("━━━ Round %d: Request ━━━", roundCounter)

			// 只打印新增的消息（相比上一轮）
			currentMessageCount := len(req.Messages)
			newMessages := req.Messages[lastMessageCount:]

			// 首轮打印系统提示
			if roundCounter == 1 && len(req.Messages) > 0 {
				for _, msg := range req.Messages {
					if msg.Role == ai.RoleSystem {
						loggerImpl.Logf("📋 System: %s", truncateMessage(msg.Text(), 200))
						break
					}
				}
			}

			// 打印新增消息
			for _, msg := range newMessages {
				switch msg.Role {
				case ai.RoleUser:
					loggerImpl.Logf("👤 User: %s", msg.Text())
				case ai.RoleTool:
					// 工具响应
					for _, part := range msg.Content {
						if part.IsToolResponse() {
							loggerImpl.Logf("🔧 Tool Response [%s]: %v",
								part.ToolResponse.Name,
								formatToolOutput(part.ToolResponse.Output))
						}
					}
				}
			}

			// 首轮打印可用工具
			if roundCounter == 1 && len(req.Tools) > 0 {
				loggerImpl.Logf("🛠️  Available Tools:")
				for _, tool := range req.Tools {
					loggerImpl.Logf("   • %s", tool.Name)
				}
			}

			// ========== 执行请求 ==========
			resp, err := next(ctx, req, cb)
			if err != nil {
				loggerImpl.Logf("❌ Error: %v", err)
				return nil, err
			}

			// ========== 响应阶段 ==========
			loggerImpl.Logf("━━━ Round %d: Response ━━━", roundCounter)

			if resp != nil && resp.Message != nil {
				// 检查响应内容类型
				hasToolCall := false
				hasText := false

				for _, part := range resp.Message.Content {
					if part.IsToolRequest() {
						hasToolCall = true
					}
					if part.IsText() && part.Text != "" {
						hasText = true
					}
				}

				// 根据内容类型打印（文本优先）
				if hasText {
					loggerImpl.Logf("🤖 Assistant: %s", resp.Message.Text())
				}

				if hasToolCall {
					loggerImpl.Logf("🔨 Tool Calls:")
					for _, part := range resp.Message.Content {
						if part.IsToolRequest() {
							loggerImpl.Logf("   → %s(%v)",
								part.ToolRequest.Name,
								formatToolInput(part.ToolRequest.Input))
						}
					}
				}

				// 更新消息计数（包含模型响应）
				lastMessageCount = currentMessageCount + 1
			}

			// Token 使用情况
			if resp != nil && resp.Usage != nil {
				loggerImpl.Logf("📊 Tokens: input=%d, output=%d",
					resp.Usage.InputTokens,
					resp.Usage.OutputTokens)
			}

			loggerImpl.Logf("━━━━━━━━━━━━━━━━━━━━━━\n")
			return resp, nil
		}
	}
}

// formatToolInput 格式化工具输入
func formatToolInput(input any) string {
	return fmt.Sprintf("%v", input)
}

// formatToolOutput 格式化工具输出
func formatToolOutput(output any) string {
	outputStr := fmt.Sprintf("%v", output)
	// 如果输出太长，截断显示
	if len(outputStr) > 256 {
		return outputStr[:253] + "..."
	}
	return outputStr
}

// truncateMessage 截断消息内容
func truncateMessage(msg string, maxLength int) string {
	if len(msg) <= maxLength {
		return msg
	}
	return msg[:maxLength] + "..."
}
