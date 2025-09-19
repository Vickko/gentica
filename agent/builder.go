package agent

import (
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// AgentBuilder 用于构建 Agent 的构建器
type AgentBuilder struct {
	g      *genkit.Genkit
	config AgentConfig
}

// NewBuilder 创建一个新的 Agent 构建器 - 必需参数通过函数参数传入
func NewBuilder(
	g *genkit.Genkit,
	name string,
	description string,
	systemPrompt string,
) *AgentBuilder {
	return &AgentBuilder{
		g: g,
		config: AgentConfig{
			Name:         name,
			Description:  description,
			SystemPrompt: systemPrompt,
			// 设置默认值
			Model:       "openai/gpt-4o-mini",
			Temperature: 0.7,
			MaxRounds:   5,
			MaxTokens:   2000,
		},
	}
}

// WithInputSchema 设置输入 schema 和必需字段
func (b *AgentBuilder) WithInputSchema(schema map[string]any, required ...string) *AgentBuilder {
	b.config.InputSchema = schema
	b.config.Required = required
	return b
}

// WithModel 设置模型
func (b *AgentBuilder) WithModel(model string) *AgentBuilder {
	b.config.Model = model
	return b
}

// WithTools 设置工具
func (b *AgentBuilder) WithTools(tools ...ai.Tool) *AgentBuilder {
	b.config.Tools = tools
	return b
}

// WithTemperature 设置温度参数
func (b *AgentBuilder) WithTemperature(temp float32) *AgentBuilder {
	b.config.Temperature = temp
	return b
}

// WithMaxTokens 设置最大 token 数
func (b *AgentBuilder) WithMaxTokens(max int) *AgentBuilder {
	b.config.MaxTokens = max
	return b
}

// WithMaxRounds 设置最大工具调用轮数
func (b *AgentBuilder) WithMaxRounds(rounds int) *AgentBuilder {
	b.config.MaxRounds = rounds
	return b
}

// WithLogging 启用对话日志
func (b *AgentBuilder) WithLogging(enable bool) *AgentBuilder {
	b.config.EnableLogging = enable
	return b
}

// Build 构建 Agent
func (b *AgentBuilder) Build() Agent {
	return &BaseAgent{
		g:        b.g,
		config:   b.config,
		messages: make([]*ai.Message, 0),
	}
}

// NewSimpleAgent 创建一个最简单的 Agent（只需名称和系统提示）
func NewSimpleAgent(g *genkit.Genkit, name, systemPrompt string) Agent {
	return NewBuilder(
		g,
		name,
		name, // description 默认同 name
		systemPrompt,
	).Build()
}

// NewToolAgent 创建一个工具型 Agent（明确输入输出格式）
func NewToolAgent(
	g *genkit.Genkit,
	name string,
	description string,
	inputSchema map[string]any,
	required []string,
	systemPrompt string,
	opts ...func(*AgentConfig),
) Agent {
	builder := NewBuilder(g, name, description, systemPrompt).
		WithInputSchema(inputSchema, required...)

	// 应用额外的配置选项
	for _, opt := range opts {
		opt(&builder.config)
	}

	return builder.Build()
}

// 配置选项函数
func WithModelOption(model string) func(*AgentConfig) {
	return func(c *AgentConfig) {
		c.Model = model
	}
}

func WithToolsOption(tools ...ai.Tool) func(*AgentConfig) {
	return func(c *AgentConfig) {
		c.Tools = tools
	}
}

func WithTemperatureOption(temp float32) func(*AgentConfig) {
	return func(c *AgentConfig) {
		c.Temperature = temp
	}
}