package provider

import (
	"context"
	"log/slog"
	"strings"

	"google.golang.org/genai"
)

func newVertexAIClient(config ProviderConfig, opts *ProviderOptions) Provider {
	project := config.ExtraParams["project"]
	location := config.ExtraParams["location"]
	cc := &genai.ClientConfig{
		Project:  project,
		Location: location,
		Backend:  genai.BackendVertexAI,
	}
	client, err := genai.NewClient(context.Background(), cc)
	if err != nil {
		slog.Error("Failed to create VertexAI client", "error", err)
		return nil
	}

	// Check if this is an Anthropic model on Vertex AI
	if strings.Contains(config.Model, "anthropic") || strings.Contains(config.Model, "claude-sonnet") {
		return newAnthropicClient(config, opts, AnthropicClientTypeVertex)
	}
	
	return &geminiClient{
		config: config,
		opts:   opts,
		client: client,
	}
}
