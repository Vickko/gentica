package agents

import (
	"context"
	"encoding/json"
	"testing"
)

func TestSearchNeedsAnalyzer(t *testing.T) {
	// 创建 SearchNeedsAnalyzer
	analyzer := NewSearchNeedsAnalyzer(g)

	// 测试用例
	testCases := []struct {
		name               string
		researchTopic      string
		expectedNeedSearch bool
		description        string
	}{
		{
			name:               "时效性技术主题",
			researchTopic:      "最新的人工智能技术发展趋势和应用案例",
			expectedNeedSearch: true,
			description:        "包含'最新'关键词，需要搜索当前信息",
		},
		{
			name:               "当前市场分析",
			researchTopic:      "2024年云计算市场份额分析",
			expectedNeedSearch: true,
			description:        "需要搜索具体的市场数据和分析",
		},
		{
			name:               "技术比较",
			researchTopic:      "React vs Vue vs Angular 框架对比",
			expectedNeedSearch: true,
			description:        "需要搜索多个框架的特性进行比较",
		},
		{
			name:               "最佳实践查询",
			researchTopic:      "微服务架构设计的最佳实践",
			expectedNeedSearch: true,
			description:        "需要搜索行业经验和案例",
		},
		{
			name:               "简单数学问题",
			researchTopic:      "1+1等于几",
			expectedNeedSearch: false,
			description:        "基础数学运算，无需搜索",
		},
		{
			name:               "基础概念定义",
			researchTopic:      "什么是变量",
			expectedNeedSearch: false,
			description:        "编程基础概念，知识稳定",
		},
		{
			name:               "哲学思考",
			researchTopic:      "人生的意义是什么",
			expectedNeedSearch: false,
			description:        "主观哲学问题，不需要搜索",
		},
		{
			name:               "无意义符号",
			researchTopic:      "@#$%^&*()",
			expectedNeedSearch: false,
			description:        "无法理解的符号组合",
		},
		{
			name:               "空白内容",
			researchTopic:      "   ",
			expectedNeedSearch: false,
			description:        "空白或无实质内容",
		},
		{
			name:               "研究性主题",
			researchTopic:      "区块链技术在供应链管理中的应用研究",
			expectedNeedSearch: true,
			description:        "需要收集多方面信息进行研究",
		},
		{
			name:               "实时信息查询",
			researchTopic:      "今天的股市行情如何",
			expectedNeedSearch: true,
			description:        "需要实时信息",
		},
		{
			name:               "历史事件",
			researchTopic:      "第二次世界大战的起因和经过",
			expectedNeedSearch: true,
			description:        "需要搜索历史资料和文献",
		},
		{
			name:               "技术教程",
			researchTopic:      "如何使用Docker部署应用",
			expectedNeedSearch: true,
			description:        "需要搜索具体的技术教程和步骤",
		},
		{
			name:               "创意写作",
			researchTopic:      "写一首关于春天的诗",
			expectedNeedSearch: false,
			description:        "创作性内容，不需要搜索",
		},
		{
			name:               "混合内容",
			researchTopic:      "分析一下Python和Java的性能差异，以及它们在2024年的流行度趋势",
			expectedNeedSearch: true,
			description:        "既有比较分析又有时效性要求",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 准备输入
			input := map[string]any{
				"research_topic": tc.researchTopic,
			}

			inputJSON, err := json.Marshal(input)
			if err != nil {
				t.Fatalf("Failed to marshal input: %v", err)
			}

			// 执行分析
			ctx := context.Background()
			result, err := analyzer.Run(ctx, string(inputJSON))
			if err != nil {
				t.Fatalf("Analysis failed: %v", err)
			}

			// 解析结果
			analysis, err := ParseSearchNeedsResult(result)
			if err != nil {
				t.Fatalf("Failed to parse analysis result: %v", err)
			}

			// 验证结果
			if analysis.Comments == "" {
				t.Error("Comments should not be empty")
			}

			// 记录结果（用于调试）
			t.Logf("Topic: %s", tc.researchTopic)
			t.Logf("Expected need_search: %v", tc.expectedNeedSearch)
			t.Logf("Actual need_search: %v", analysis.NeedSearch)
			t.Logf("Comments: %s", analysis.Comments)
			t.Logf("Test description: %s", tc.description)

			// 验证搜索需求判断
			if analysis.NeedSearch != tc.expectedNeedSearch {
				t.Errorf("Unexpected need_search value. Expected: %v, Got: %v",
					tc.expectedNeedSearch, analysis.NeedSearch)
			}
		})
	}
}

func TestParseSearchNeedsResult(t *testing.T) {
	testCases := []struct {
		name      string
		input     string
		expectErr bool
	}{
		{
			name: "Valid JSON",
			input: `{
				"comments": "这是一个时效性很强的主题，需要搜索最新信息。",
				"need_search": true
			}`,
			expectErr: false,
		},
		{
			name: "JSON with extra text",
			input: `Here is the analysis result:
			{
				"comments": "这是一个基础概念问题，不需要搜索。",
				"need_search": false
			}
			That's all.`,
			expectErr: false,
		},
		{
			name: "Missing comments field",
			input: `{
				"need_search": true
			}`,
			expectErr: true,
		},
		{
			name:      "Invalid JSON",
			input:     "This is not JSON",
			expectErr: true,
		},
		{
			name: "Empty comments",
			input: `{
				"comments": "",
				"need_search": false
			}`,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ParseSearchNeedsResult(tc.input)
			if tc.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result == nil {
					t.Error("Result should not be nil")
				} else if result.Comments == "" {
					t.Error("Comments should not be empty")
				}
			}
		})
	}
}

// TestSearchNeedsAnalyzerEdgeCases 测试边缘情况
func TestSearchNeedsAnalyzerEdgeCases(t *testing.T) {
	analyzer := NewSearchNeedsAnalyzer(g)

	edgeCases := []struct {
		name          string
		researchTopic string
		shouldWork    bool
	}{
		{
			name:          "非常长的研究主题",
			researchTopic: "探讨在云原生环境下，如何设计和实现一个高可用、可扩展、安全的微服务架构，包括服务发现、负载均衡、熔断器、API网关、分布式追踪、日志聚合、监控告警、容器编排、持续集成持续部署、服务网格、零信任安全模型等方面的最佳实践，以及如何处理分布式事务、数据一致性、缓存策略、消息队列、事件驱动架构等技术挑战。",
			shouldWork:    true,
		},
		{
			name:          "包含特殊字符",
			researchTopic: "C++ vs C# vs F# 性能对比",
			shouldWork:    true,
		},
		{
			name:          "多语言混合",
			researchTopic: "研究AI人工智能在healthcare医疗领域的application应用",
			shouldWork:    true,
		},
		{
			name:          "纯数字",
			researchTopic: "2024 2025 2026",
			shouldWork:    true,
		},
		{
			name:          "表情符号",
			researchTopic: "🚀 🔥 💡",
			shouldWork:    true,
		},
		{
			name:          "代码片段作为主题",
			researchTopic: "function add(a, b) { return a + b; }",
			shouldWork:    true,
		},
		{
			name:          "URL作为研究主题",
			researchTopic: "https://github.com/trending",
			shouldWork:    true,
		},
		{
			name:          "SQL查询作为主题",
			researchTopic: "SELECT * FROM users WHERE age > 18",
			shouldWork:    true,
		},
	}

	for _, ec := range edgeCases {
		t.Run(ec.name, func(t *testing.T) {
			input := map[string]any{
				"research_topic": ec.researchTopic,
			}

			inputJSON, err := json.Marshal(input)
			if err != nil {
				t.Fatalf("Failed to marshal input: %v", err)
			}

			ctx := context.Background()
			result, err := analyzer.Run(ctx, string(inputJSON))

			if ec.shouldWork {
				if err != nil {
					t.Fatalf("Analysis should work but failed: %v", err)
				}

				analysis, err := ParseSearchNeedsResult(result)
				if err != nil {
					t.Fatalf("Failed to parse result: %v", err)
				}

				// 验证返回了有效的结果
				if analysis.Comments == "" {
					t.Error("Comments should not be empty for edge case")
				}

				t.Logf("Edge case topic: %s", ec.researchTopic)
				t.Logf("Need search: %v", analysis.NeedSearch)
				t.Logf("Comments: %s", analysis.Comments)
			} else {
				if err == nil {
					t.Error("Expected failure but got success")
				}
			}
		})
	}
}

// TestSearchNeedsAnalyzerConsistency 测试结果一致性
func TestSearchNeedsAnalyzerConsistency(t *testing.T) {
	// 由于温度设置为0.3，相同输入应该产生相似的结果
	analyzer := NewSearchNeedsAnalyzer(g)

	topic := "区块链技术的应用前景"
	input := map[string]any{
		"research_topic": topic,
	}

	inputJSON, _ := json.Marshal(input)
	ctx := context.Background()

	// 运行多次，检查结果一致性
	results := make([]bool, 3)
	for i := 0; i < 3; i++ {
		result, err := analyzer.Run(ctx, string(inputJSON))
		if err != nil {
			t.Fatalf("Run %d failed: %v", i+1, err)
		}

		analysis, err := ParseSearchNeedsResult(result)
		if err != nil {
			t.Fatalf("Failed to parse result on run %d: %v", i+1, err)
		}

		results[i] = analysis.NeedSearch
		t.Logf("Run %d - need_search: %v", i+1, analysis.NeedSearch)
	}

	// 检查结果是否一致
	for i := 1; i < len(results); i++ {
		if results[i] != results[0] {
			t.Logf("Warning: Inconsistent results detected, but this might be acceptable due to model variance")
			// 不作为错误，只是警告
		}
	}
}
