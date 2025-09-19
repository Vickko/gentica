package agent

import (
	"context"

	"github.com/firebase/genkit/go/ai"
)

// Agent 定义了一个可以与 LLM 对话的智能代理
// Agent 可以在初始化时预定义 prompt、指定模型、配置工具集合
// 并且能够自行管理消息历史，支持外部干预消息列表
type Agent interface {
	// 基础信息
	Name() string                // Agent 名称
	Description() string         // Agent 描述，用于作为工具时的描述
	InputSchema() map[string]any // 输入参数 schema，与 Tool 一致

	// 核心执行方法 - 默认携带上下文
	Run(ctx context.Context, input string) (string, error) // 解析 JSON input，执行任务

	// 消息管理
	GetMessages() []*ai.Message               // 获取当前消息历史
	SetMessages(messages []*ai.Message)       // 设置消息历史（覆盖）
	AddMessage(message *ai.Message)           // 添加单条消息
	ClearHistory()                             // 清空消息历史

	// 配置获取
	GetConfig() AgentConfig // 获取 Agent 配置
}

// AgentConfig 定义 Agent 的配置
type AgentConfig struct {
	Name         string           // Agent 名称
	Description  string           // Agent 描述（用于 Tool 转换）
	InputSchema  map[string]any   // 输入参数 schema
	Required     []string         // 必需参数
	SystemPrompt string           // 系统提示
	Model        string           // 指定模型
	Tools        []ai.Tool        // 工具集合
	Temperature  float32          // 温度参数
	MaxTokens    int              // 最大 token 数
	MaxRounds    int              // 最大工具调用轮数
	EnableLogging bool            // 是否启用对话日志
}