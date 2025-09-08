package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/catwalk/pkg/catwalk"
	// "github.com/charmbracelet/crush/internal/config"
	"gentica/config"
	"gentica/message"
	// "github.com/charmbracelet/crush/internal/message"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

func TestMain(m *testing.M) {
	// 由于 instance 是 private 的，我们需要通过 Init 来设置
	tempDir, _ := os.MkdirTemp("", "provider-test-*")
	defer os.RemoveAll(tempDir)
	
	// 创建空配置文件并初始化
	configFile := filepath.Join(tempDir, ".gentica.json")
	os.WriteFile(configFile, []byte("{}"), 0644)
	config.Init(tempDir, tempDir, false)

	os.Exit(m.Run())
}

func TestOpenAIClientStreamChoices(t *testing.T) {
	// Create a mock server that returns Server-Sent Events with empty choices
	// This simulates the 🤡 behavior when a server returns 200 instead of 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)

		emptyChoicesChunk := map[string]any{
			"id":      "chat-completion-test",
			"object":  "chat.completion.chunk",
			"created": time.Now().Unix(),
			"model":   "test-model",
			"choices": []any{}, // Empty choices array that causes panic
		}

		jsonData, _ := json.Marshal(emptyChoicesChunk)
		w.Write([]byte("data: " + string(jsonData) + "\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	// Create OpenAI client pointing to our mock server
	client := &openaiClient{
		providerOptions: providerClientOptions{
			modelType:     config.SelectedModelTypeLarge,
			apiKey:        "test-key",
			systemMessage: "test",
			model: func(config.SelectedModelType) catwalk.Model {
				return catwalk.Model{
					ID:   "test-model",
					Name: "test-model",
				}
			},
		},
		client: openai.NewClient(
			option.WithAPIKey("test-key"),
			option.WithBaseURL(server.URL),
		),
	}

	// Create test messages
	messages := []message.Message{
		{
			Role:  message.User,
			Parts: []message.ContentPart{message.TextContent{Text: "Hello"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	eventsChan := client.stream(ctx, messages, nil)

	// Collect events - this will panic without the bounds check
	for event := range eventsChan {
		t.Logf("Received event: %+v", event)
		if event.Type == EventError || event.Type == EventComplete {
			break
		}
	}
}
