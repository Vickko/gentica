package agents

import (
	"context"
	"fmt"

	"gentica/agent"
	"gentica/provider"
	"gentica/tools"
)

// Manager 存储初始化后的 Agent 实例
type Manager struct {
	QueryAnalyzer *agent.Agent
	Chat          *agent.Agent
	SearchTask    *agent.Agent
	SearchManager *agent.Agent

	// 共享存储
	Storage *SearchStorage
}

// InitializeAgents 初始化所有预定义的 Agent
func InitializeAgents(ctx context.Context, providerConfig provider.ProviderConfig) (*Manager, error) {
	// 创建共享存储
	storage := NewSearchStorage()

	// 创建查询分析 Agent (不需要工具)
	queryAnalyzer, err := agent.NewAgent(
		agent.Config{
			Name:         QueryAnalyzerAgentConfig.Name,
			SystemPrompt: QueryAnalyzerAgentConfig.SystemPrompt,
			Model:        QueryAnalyzerAgentConfig.Model,
		},
		providerConfig,
		nil, // 不需要工具
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create query analyzer: %w", err)
	}

	// 为 Chat Agent 准备工具 - 添加 Query Analyzer 作为工具
	chatTools := []tools.BaseTool{
		NewAgentTool(queryAnalyzer, "Query Analyzer"),
	}

	// 创建普通对话 Agent
	chatAgent, err := agent.NewAgent(
		agent.Config{
			Name:         ChatAgentConfig.Name,
			SystemPrompt: ChatAgentConfig.SystemPrompt,
			Model:        ChatAgentConfig.Model,
		},
		providerConfig,
		chatTools,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat agent: %w", err)
	}

	// 创建搜索任务 Agent (需要 searchcrawler 工具)
	searchTaskTools := []tools.BaseTool{
		tools.NewSearchCrawlerTool(), // 使用 tools 包提供的 SearchCrawler
	}

	fmt.Printf("DEBUG: Creating SearchTask agent with config: %+v\n", providerConfig)
	searchTask, err := agent.NewAgent(
		agent.Config{
			Name:         SearchTaskAgentConfig.Name,
			SystemPrompt: SearchTaskAgentConfig.SystemPrompt,
			Model:        SearchTaskAgentConfig.Model,
		},
		providerConfig,
		searchTaskTools,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create search task agent: %w", err)
	}
	fmt.Printf("DEBUG: SearchTask agent created: %v (is nil: %v)\n", searchTask, searchTask == nil)

	// 为 SearchManager 准备工具
	searchManagerTools := []tools.BaseTool{
		NewAddToResultSetTool(storage),
		NewViewResultSetTool(storage),
		NewViewAllResultsTool(storage),
		NewClearResultSetTool(storage),
		NewAgentTool(queryAnalyzer, "Query Analyzer"),
		NewAgentTool(searchTask, "Search Task"),
	}

	// 创建搜索管理器 Agent
	searchManager, err := agent.NewAgent(
		agent.Config{
			Name:         SearchManagerAgentConfig.Name,
			SystemPrompt: SearchManagerAgentConfig.SystemPrompt,
			Model:        SearchManagerAgentConfig.Model,
		},
		providerConfig,
		searchManagerTools,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create search manager: %w", err)
	}

	return &Manager{
		QueryAnalyzer: queryAnalyzer,
		Chat:          chatAgent,
		SearchTask:    searchTask,
		SearchManager: searchManager,
		Storage:       storage,
	}, nil
}
