package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"gentica/message"
	"gentica/ppt/agents"
	"gentica/provider"
)

func main() {
	ctx := context.Background()

	// 配置 Provider
	providerConfig := provider.ProviderConfig{
		Type:      provider.TypeGemini,
		BaseURL:   "https://aihubmix.com/gemini",
		APIKey:    "sk-6kgtZQDkmZDQMfCo28C360320cEf45FaAf1577Ef08F4032b",
		Model:     "gemini-2.5-pro",
		MaxTokens: 4096,
	}

	// 初始化所有 Agents
	fmt.Println("🚀 初始化 PPT Agent 系统...")
	agentManager, err := agents.InitializeAgents(ctx, providerConfig)
	if err != nil {
		log.Fatalf("初始化 Agents 失败: %v", err)
	}
	fmt.Println("✅ Agents 初始化成功")

	// 测试 1: 简单的查询分析

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("📝 测试 1: 查询分析 Agent")
	fmt.Println(strings.Repeat("=", 80))

	testQuery := "今天的新闻？"
	fmt.Printf("查询: %s\n", testQuery)

	messages := []message.Message{
		{
			Role: message.User,
			Parts: []message.ContentPart{
				message.TextContent{Text: testQuery},
			},
		},
	}

	response, err := agentManager.QueryAnalyzer.Run(ctx, messages)
	if err != nil {
		log.Printf("❌ 查询分析失败: %v", err)
	} else {
		fmt.Println("分析结果:")
		for _, part := range response.Parts {
			if textPart, ok := part.(message.TextContent); ok {
				fmt.Println(textPart.Text)
			}
		}
	}

	// 测试 2: 搜索任务
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("🔍 测试 2: 搜索任务 Agent")
	fmt.Println(strings.Repeat("=", 80))

	searchQuery := "大语言模型最新进展"
	fmt.Printf("搜索关键词: %s\n", searchQuery)

	searchPrompt := fmt.Sprintf(`请搜索关键词'%s'，研究主题是：了解大语言模型的最新技术进展`, searchQuery)
	searchMessages := []message.Message{
		{
			Role: message.User,
			Parts: []message.ContentPart{
				message.TextContent{Text: searchPrompt},
			},
		},
	}

	searchResponse, err := agentManager.SearchTask.Run(ctx, searchMessages)
	if err != nil {
		log.Printf("❌ 搜索任务失败: %v", err)
	} else {
		fmt.Println("搜索结果:")
		for _, part := range searchResponse.Parts {
			if textPart, ok := part.(message.TextContent); ok {
				// 尝试解析JSON显示
				var result map[string]interface{}
				if err := json.Unmarshal([]byte(textPart.Text), &result); err == nil {
					if searchResults, ok := result["search_results"].([]interface{}); ok {
						fmt.Printf("找到 %d 个结果\n", len(searchResults))
						for i, r := range searchResults {
							if res, ok := r.(map[string]interface{}); ok {
								fmt.Printf("%d. %s (总分: %.1f)\n",
									i+1,
									res["title"],
									res["total_score"])
							}
						}
					}
				} else {
					fmt.Println(textPart.Text)
				}
			}
		}
	}
	fmt.Println("123123123123")
	fmt.Println(searchResponse)
	fmt.Println("123123123123")

	// 测试 3: 完整的搜索管理流程
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("🎯 测试 3: 搜索管理器 - 完整流程")
	fmt.Println(strings.Repeat("=", 80))

	researchTopic := "人工智能在教育领域的应用"
	fmt.Printf("研究主题: %s\n", researchTopic)
	fmt.Println("⏳ 开始搜索和评估流程...")

	// 设置超时
	searchCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	managerMessages := []message.Message{
		{
			Role: message.User,
			Parts: []message.ContentPart{
				message.TextContent{Text: researchTopic},
			},
		},
	}

	startTime := time.Now()
	managerResponse, err := agentManager.SearchManager.Run(searchCtx, managerMessages)
	elapsed := time.Since(startTime)

	if err != nil {
		log.Printf("❌ 搜索管理器执行失败: %v", err)
	} else {
		fmt.Printf("⏱️  处理时间: %.2f秒\n", elapsed.Seconds())
		fmt.Println("\n最终结果:")

		for _, part := range managerResponse.Parts {
			if textPart, ok := part.(message.TextContent); ok {
				// 尝试解析JSON
				var result map[string]interface{}
				if err := json.Unmarshal([]byte(textPart.Text), &result); err == nil {
					// 格式化输出
					if needSearch, ok := result["need_search"].(bool); ok {
						fmt.Printf("需要搜索: %v\n", needSearch)
					}
					if status, ok := result["status"].(string); ok {
						fmt.Printf("状态: %s\n", status)
					}
					if totalSearches, ok := result["total_searches"].(float64); ok {
						fmt.Printf("总搜索次数: %d\n", int(totalSearches))
					}
					if rounds, ok := result["rounds"].(float64); ok {
						fmt.Printf("搜索轮数: %d\n", int(rounds))
					}

					// 显示高质量结果
					if resultSet, ok := result["result_set"].([]interface{}); ok {
						fmt.Printf("\n📚 搜集到的高质量资料 (%d条):\n", len(resultSet))
						fmt.Println(strings.Repeat("-", 40))

						for idx, item := range resultSet {
							if resultMap, ok := item.(map[string]interface{}); ok {
								fmt.Printf("%d. %s\n", idx+1, resultMap["title"])
								if score, ok := resultMap["score"].(float64); ok {
									fmt.Printf("   评分: %.1f/100\n", score)
								}
							}
						}
					}

					// 显示总结
					if summary, ok := result["agent_summary"].(string); ok {
						fmt.Printf("\n📝 总结:\n%s\n", summary)
					}
				} else {
					// 原始输出
					fmt.Println(textPart.Text)
				}
			}
		}
	}

	// 显示存储统计
	totalSearches, searchRounds, qualityCount := agentManager.Storage.GetStats()
	fmt.Printf("\n💾 存储统计:\n")
	fmt.Printf("  总搜索数: %d\n", totalSearches)
	fmt.Printf("  搜索轮数: %d\n", searchRounds)
	fmt.Printf("  质量结果数: %d\n", qualityCount)

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("✅ 所有测试完成!")
}
