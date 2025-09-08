package agent

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"gentica/config"
	"gentica/csync"
	"gentica/db"
	"gentica/message"
	"gentica/session"

	"github.com/charmbracelet/catwalk/pkg/catwalk"
)

var (
	testDB       *sql.DB
	testQueries  *db.Queries
	testSessions session.Service
	testMessages message.Service
	testTempDir  string
)

func init() {
	ctx := context.Background()

	// 创建临时目录用于测试数据库
	var err error
	testTempDir, err = os.MkdirTemp("", "agent-test-*")
	if err != nil {
		panic("创建临时目录失败: " + err.Error())
	}

	// 使用 db.Connect 初始化数据库
	testDB, err = db.Connect(ctx, testTempDir)
	if err != nil {
		panic("连接数据库失败: " + err.Error())
	}

	// 创建数据库查询对象
	testQueries = db.New(testDB)

	// 创建真实的服务
	testSessions = session.NewService(testQueries)
	testMessages = message.NewService(testQueries)

	// 初始化配置
	config.Init(testTempDir, testTempDir, false)

	// 确保配置已初始化
	cfg := config.Get()
	if cfg == nil {
		panic("配置未初始化")
	}

	// 设置 provider 配置 - 使用 Gemini
	providerMap := csync.NewMap[string, config.ProviderConfig]()
	providerMap.Set("gemini", config.ProviderConfig{
		ID:      "gemini",
		Name:    "Gemini",
		Type:    catwalk.TypeGemini,
		BaseURL: "https://aihubmix.com/gemini",
		APIKey:  "sk-6kgtZQDkmZDQMfCo28C360320cEf45FaAf1577Ef08F4032b",
		Models: []catwalk.Model{
			{
				ID:               "gemini-2.5-flash",
				Name:             "Gemini 2.5 Flash",
				DefaultMaxTokens: 1024,
				ContextWindow:    1000000,
				CostPer1MIn:      0.35,
				CostPer1MOut:     1.05,
			},
		},
	})
	cfg.Providers = providerMap

	// 设置 model 配置
	cfg.Models = map[config.SelectedModelType]config.SelectedModel{
		config.SelectedModelTypeLarge: {
			Provider: "gemini",
			Model:    "gemini-2.5-flash",
		},
		config.SelectedModelTypeSmall: {
			Provider: "gemini",
			Model:    "gemini-2.5-flash",
		},
	}
}

// TestSimpleAgentChat 测试简单的 agent 对话 - 使用真实服务
func TestSimpleAgentChat(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试12")
	}

	// 清理函数
	t.Cleanup(func() {
		if testDB != nil {
			testDB.Close()
		}
		if testTempDir != "" {
			os.RemoveAll(testTempDir)
		}
	})

	ctx := context.Background()

	// 创建测试 session
	testSession, err := testSessions.Create(ctx, "Test Session")
	if err != nil {
		t.Fatalf("创建 session 失败: %v", err)
	}

	// 创建最简单的 agent 配置
	agentCfg := config.Agent{
		ID:    "test-agent",
		Name:  "Test Agent",
		Model: config.SelectedModelTypeLarge,
		// 不要任何工具，纯聊天
		AllowedTools: []string{},
	}

	// 创建 agent
	agent, err := NewAgent(ctx, agentCfg, testSessions, testMessages, nil)
	if err != nil {
		t.Fatalf("创建 agent 失败: %v", err)
	}

	// 发送一个简单的聊天请求
	content := "Hello World"

	// 使用带超时的 context
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	t.Logf("开始运行 agent，sessionID: %s", testSession.ID)
	t.Logf("使用Provider: %s, Model: %s", config.Get().Models[config.SelectedModelTypeLarge].Provider, config.Get().Models[config.SelectedModelTypeLarge].Model)
	eventChan, err := agent.Run(ctx, testSession.ID, content)
	if err != nil {
		t.Fatalf("运行 agent 失败: %v", err)
	}

	if eventChan == nil {
		t.Fatal("eventChan 为 nil")
	}
	t.Log("成功获取 eventChan")

	// 处理事件
	var responseReceived bool
	var responseContent string
	eventCount := 0

	for event := range eventChan {
		eventCount++
		t.Logf("收到事件 #%d: Type=%v, Done=%v, Error=%v", eventCount, event.Type, event.Done, event.Error)

		if event.Error != nil {
			t.Fatalf("收到错误事件: %v", event.Error)
		}

		if event.Type == AgentEventTypeResponse && event.Done {
			responseReceived = true
			// 获取响应内容
			msgList, _ := testMessages.List(ctx, testSession.ID)
			for _, msg := range msgList {
				if msg.Role == message.Assistant {
					for _, part := range msg.Parts {
						if textPart, ok := part.(message.TextContent); ok {
							responseContent += textPart.Text
						}
					}
				}
			}
			t.Logf("收到响应: %s", responseContent)
		}
	}

	t.Logf("事件处理完成，共收到 %d 个事件", eventCount)

	// 检查是否因为超时而退出
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatal("等待响应超时")
	}

	if !responseReceived {
		t.Error("未收到响应事件")
	}

	if responseContent == "" {
		t.Error("响应内容为空")
	}

	t.Logf("测试成功完成，收到响应: %s", responseContent)
}
