package agents

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// g 变量现在在 research_collector_test.go 中定义和初始化

func TestArticleEvaluator(t *testing.T) {

	// 创建临时测试文件
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_article.txt")
	testContent := `# Go语言并发编程最佳实践

发布日期：2024年9月

## 介绍
Go语言的并发模型是其最强大的特性之一。本文将深入探讨Go语言并发编程的最佳实践，包括goroutine的使用、channel通信、同步原语等核心概念。

## Goroutine基础
Goroutine是Go语言中的轻量级线程。与传统的操作系统线程相比，goroutine的创建和销毁开销极小，可以轻松创建成千上万个goroutine。

### 创建Goroutine
使用go关键字可以启动一个新的goroutine：
go func() {
    // 并发执行的代码
}()

## Channel通信
Channel是Go语言中goroutine之间通信的主要方式。遵循"不要通过共享内存来通信，而应该通过通信来共享内存"的原则。

### 无缓冲Channel
ch := make(chan int)
无缓冲channel提供同步通信，发送和接收操作会阻塞直到另一端准备好。

### 有缓冲Channel
ch := make(chan int, 10)
有缓冲channel允许异步通信，只有在缓冲区满时发送才会阻塞。

## 同步原语
除了channel，Go还提供了传统的同步原语：
- sync.Mutex: 互斥锁
- sync.RWMutex: 读写锁
- sync.WaitGroup: 等待组
- sync.Once: 单次执行

## 最佳实践总结
1. 优先使用channel进行goroutine间通信
2. 避免goroutine泄漏，确保所有goroutine都能正常退出
3. 合理使用context进行超时控制和取消操作
4. 使用sync/atomic包进行原子操作
5. 避免过度使用goroutine，考虑使用worker pool模式

## 结论
掌握Go语言的并发编程是成为高级Go开发者的必经之路。通过合理使用goroutine和channel，可以编写出高效、可维护的并发程序。`

	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// 创建 ArticleEvaluator
	evaluator := NewArticleEvaluator(g, tempDir)

	// 测试用例
	testCases := []struct {
		name          string
		filePath      string
		researchTopic string
		expectSuccess bool
	}{
		{
			name:          "Go并发编程相关文章",
			filePath:      testFile,
			researchTopic: "Go语言并发编程",
			expectSuccess: true,
		},
		{
			name:          "不太相关的主题",
			filePath:      testFile,
			researchTopic: "Python机器学习",
			expectSuccess: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 准备输入
			input := map[string]any{
				"file_path":      tc.filePath,
				"research_topic": tc.researchTopic,
			}

			inputJSON, err := json.Marshal(input)
			if err != nil {
				t.Fatalf("Failed to marshal input: %v", err)
			}

			// 执行评估
			ctx := context.Background()
			result, err := evaluator.Run(ctx, string(inputJSON))
			if err != nil {
				if tc.expectSuccess {
					t.Fatalf("Evaluation failed unexpectedly: %v", err)
				}
				return
			}

			if !tc.expectSuccess {
				t.Fatal("Expected failure but got success")
			}

			// 解析结果
			evaluation, err := ParseEvaluationResult(result)
			if err != nil {
				t.Fatalf("Failed to parse evaluation result: %v", err)
			}

			// 验证结果
			if evaluation.Keyword == "" {
				t.Error("Keyword should not be empty")
			}

			if evaluation.Status != "completed" {
				t.Errorf("Status should be 'completed', got: %s", evaluation.Status)
			}

			// 验证评分
			score := evaluation.Result
			if score.Title == "" {
				t.Error("Title should not be empty")
			}
			if score.Score < 0 || score.Score > 100 {
				t.Errorf("Total score out of range: %d", score.Score)
			}
			if score.Relevance < 0 || score.Relevance > 40 {
				t.Errorf("Relevance score out of range: %d", score.Relevance)
			}
			if score.Quality < 0 || score.Quality > 30 {
				t.Errorf("Quality score out of range: %d", score.Quality)
			}
			if score.Timeliness < 0 || score.Timeliness > 30 {
				t.Errorf("Timeliness score out of range: %d", score.Timeliness)
			}
			if score.Comments == "" {
				t.Error("Comments should not be empty")
			}

			// 验证总分等于各维度之和
			expectedTotal := score.Relevance + score.Quality + score.Timeliness
			if score.Score != expectedTotal {
				t.Errorf("Total score (%d) doesn't match sum of dimensions (%d)", score.Score, expectedTotal)
			}

			// 打印评估结果（用于调试）
			t.Logf("Evaluation for '%s':", tc.researchTopic)
			t.Logf("  Keyword: %s", evaluation.Keyword)
			t.Logf("  Article: %s", score.Title)
			t.Logf("    Total Score: %d", score.Score)
			t.Logf("    Relevance: %d/40", score.Relevance)
			t.Logf("    Quality: %d/30", score.Quality)
			t.Logf("    Timeliness: %d/30", score.Timeliness)
			t.Logf("    Comments: %s", score.Comments)
		})
	}
}

func TestParseEvaluationResult(t *testing.T) {
	testCases := []struct {
		name      string
		input     string
		expectErr bool
	}{
		{
			name: "Valid JSON",
			input: `{
				"keyword": "Go并发",
				"result": {
					"title": "Go并发编程",
					"score": 85,
					"relevance": 38,
					"quality": 25,
					"timeliness": 22,
					"comments": "文章直接聚焦Go语言并发编程核心概念，内容深入且实用"
				},
				"status": "completed"
			}`,
			expectErr: false,
		},
		{
			name: "JSON with extra text",
			input: `Here is the evaluation result:
			{
				"keyword": "Go并发",
				"result": {
					"title": "Go并发编程",
					"score": 85,
					"relevance": 38,
					"quality": 25,
					"timeliness": 22,
					"comments": "文章直接聚焦Go语言并发编程核心概念，内容深入且实用"
				},
				"status": "completed"
			}
			That's all.`,
			expectErr: false,
		},
		{
			name:      "Invalid JSON",
			input:     "This is not JSON",
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ParseEvaluationResult(tc.input)
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
				} else if result.Keyword == "" {
					t.Error("Keyword should not be empty")
				}
			}
		})
	}
}

