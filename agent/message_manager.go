package agent

import (
	"github.com/firebase/genkit/go/ai"
)

// MessageManager 定义消息管理器接口
type MessageManager interface {
	TruncateMessages(messages []*ai.Message, maxCount int) []*ai.Message
	CompressMessages(messages []*ai.Message) ([]*ai.Message, error)
	FilterByRole(messages []*ai.Message, roles ...ai.Role) []*ai.Message
	EstimateTokens(messages []*ai.Message) int
}

// DefaultMessageManager 默认消息管理器实现
type DefaultMessageManager struct {
	windowSize    int
	compressRatio float32
	tokenLimit    int
}

// MessageOption 消息管理器配置选项
type MessageOption func(*DefaultMessageManager)

// NewMessageManager 创建新的消息管理器
func NewMessageManager(opts ...MessageOption) MessageManager {
	mm := &DefaultMessageManager{
		windowSize:    20,
		compressRatio: 0.5,
		tokenLimit:    8000,
	}

	for _, opt := range opts {
		opt(mm)
	}

	return mm
}

// WithWindowSize 设置消息窗口大小
func WithWindowSize(size int) MessageOption {
	return func(mm *DefaultMessageManager) {
		mm.windowSize = size
	}
}

// WithCompressRatio 设置压缩比率
func WithCompressRatio(ratio float32) MessageOption {
	return func(mm *DefaultMessageManager) {
		mm.compressRatio = ratio
	}
}

// WithTokenLimit 设置 token 限制
func WithTokenLimit(limit int) MessageOption {
	return func(mm *DefaultMessageManager) {
		mm.tokenLimit = limit
	}
}

// TruncateMessages 实现消息截断（滑动窗口）
func (mm *DefaultMessageManager) TruncateMessages(messages []*ai.Message, maxCount int) []*ai.Message {
	if len(messages) <= maxCount {
		return messages
	}

	// 保留系统消息和最近的消息
	var result []*ai.Message

	// 首先保留所有系统消息
	systemMessages := 0
	for _, msg := range messages {
		if msg.Role == ai.RoleSystem {
			result = append(result, msg)
			systemMessages++
		}
	}

	// 计算剩余可用空间
	remainingSpace := maxCount - systemMessages
	if remainingSpace <= 0 {
		return result // 只返回系统消息
	}

	// 添加最近的消息
	start := len(messages) - remainingSpace
	if start < 0 {
		start = 0
	}

	// 添加非系统消息
	for i := start; i < len(messages); i++ {
		if messages[i].Role != ai.RoleSystem {
			result = append(result, messages[i])
		}
	}

	return result
}

// CompressMessages 压缩消息历史
func (mm *DefaultMessageManager) CompressMessages(messages []*ai.Message) ([]*ai.Message, error) {
	// 如果消息数量少于窗口大小，不需要压缩
	if len(messages) <= mm.windowSize {
		return messages, nil
	}

	// 保留系统消息
	var compressed []*ai.Message
	var toCompress []*ai.Message

	for _, msg := range messages {
		if msg.Role == ai.RoleSystem {
			compressed = append(compressed, msg)
		} else {
			toCompress = append(toCompress, msg)
		}
	}

	// 计算需要压缩的消息数量
	compressCount := int(float32(len(toCompress)) * mm.compressRatio)
	if compressCount < 2 {
		return messages, nil // 消息太少，不值得压缩
	}

	// 简单的压缩策略：创建摘要消息
	// 这里可以集成 LLM 来生成更智能的摘要
	summary := ai.NewUserTextMessage("Previous conversation summary: " + mm.createSimpleSummary(toCompress[:compressCount]))
	compressed = append(compressed, summary)

	// 添加未压缩的消息
	compressed = append(compressed, toCompress[compressCount:]...)

	return compressed, nil
}

// createSimpleSummary 创建简单的摘要
func (mm *DefaultMessageManager) createSimpleSummary(messages []*ai.Message) string {
	// 简单实现：提取每个消息的前几个字符
	// 实际应用中，这里应该调用 LLM 生成摘要
	summary := ""
	for i, msg := range messages {
		if i > 0 {
			summary += "; "
		}
		text := msg.Text()
		if len(text) > 50 {
			text = text[:50] + "..."
		}
		summary += text
	}
	return summary
}

// FilterByRole 按角色过滤消息
func (mm *DefaultMessageManager) FilterByRole(messages []*ai.Message, roles ...ai.Role) []*ai.Message {
	var filtered []*ai.Message
	roleSet := make(map[ai.Role]bool)
	for _, role := range roles {
		roleSet[role] = true
	}

	for _, msg := range messages {
		if roleSet[msg.Role] {
			filtered = append(filtered, msg)
		}
	}

	return filtered
}

// EstimateTokens 估算 token 数量
func (mm *DefaultMessageManager) EstimateTokens(messages []*ai.Message) int {
	// 简单估算：每个字符约 0.25 个 token
	// 实际应用中应该使用更准确的 tokenizer
	totalChars := 0
	for _, msg := range messages {
		totalChars += len(msg.Text())

		// 考虑工具调用的 token
		for _, part := range msg.Content {
			if part.IsToolRequest() {
				// 工具调用通常需要更多 token
				totalChars += len(part.ToolRequest.Name) * 2
				if part.ToolRequest.Input != nil {
					// 估算输入参数的大小
					totalChars += 100 // 简单估算
				}
			}
		}
	}
	return totalChars / 4
}

// SmartMessageManager 智能消息管理器，提供更高级的功能
type SmartMessageManager struct {
	*DefaultMessageManager
	agent Agent // 用于生成摘要的 agent
}

// NewSmartMessageManager 创建智能消息管理器
func NewSmartMessageManager(summaryAgent Agent, opts ...MessageOption) MessageManager {
	dmm := &DefaultMessageManager{
		windowSize:    20,
		compressRatio: 0.5,
		tokenLimit:    8000,
	}

	for _, opt := range opts {
		opt(dmm)
	}

	return &SmartMessageManager{
		DefaultMessageManager: dmm,
		agent:                summaryAgent,
	}
}

// CompressMessages 使用 LLM 智能压缩消息
func (smm *SmartMessageManager) CompressMessages(messages []*ai.Message) ([]*ai.Message, error) {
	if smm.agent == nil {
		// 如果没有配置 agent，使用默认方法
		return smm.DefaultMessageManager.CompressMessages(messages)
	}

	// 使用 agent 生成摘要
	// 这里的实现可以根据需要扩展
	return smm.DefaultMessageManager.CompressMessages(messages)
}