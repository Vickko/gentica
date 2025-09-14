package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gentica/message"
	"gentica/tools"
)

type bedrockClient struct {
	config        ProviderConfig
	opts          *ProviderOptions
	childProvider Provider
}

func newBedrockClient(config ProviderConfig, opts *ProviderOptions) Provider {
	// Get AWS region from environment
	region := config.ExtraParams["region"]
	if region == "" {
		region = "us-east-1" // default region
	}
	if len(region) < 2 {
		return &bedrockClient{
			config:        config,
			opts:          opts,
			childProvider: nil, // Will cause an error when used
		}
	}

	// Determine which provider to use based on the model
	if strings.Contains(config.Model, "anthropic") {
		// Create Anthropic client with Bedrock configuration
		// Disable cache for Bedrock
		bedrockOpts := &ProviderOptions{
			DisableCache:  true,
			SystemMessage: opts.SystemMessage,
		}
		return &bedrockClient{
			config:        config,
			opts:          opts,
			childProvider: newAnthropicClient(config, bedrockOpts, AnthropicClientTypeBedrock),
		}
	}

	// Return client with nil childProvider if model is not supported
	return &bedrockClient{
		config:        config,
		opts:          opts,
		childProvider: nil,
	}
}

func (b *bedrockClient) Send(ctx context.Context, messages []message.Message, tools []tools.BaseTool) (*ProviderResponse, error) {
	if b.childProvider == nil {
		return nil, errors.New("unsupported model for bedrock provider")
	}
	return b.childProvider.Send(ctx, messages, tools)
}

func (b *bedrockClient) Stream(ctx context.Context, messages []message.Message, tools []tools.BaseTool) <-chan ProviderEvent {
	eventChan := make(chan ProviderEvent)

	if b.childProvider == nil {
		go func() {
			eventChan <- ProviderEvent{
				Type:  EventError,
				Error: errors.New("unsupported model for bedrock provider"),
			}
			close(eventChan)
		}()
		return eventChan
	}

	return b.childProvider.Stream(ctx, messages, tools)
}

func (b *bedrockClient) Model() Model {
	// For Bedrock, prefix the model name with region
	region := b.config.ExtraParams["region"]
	if region == "" {
		region = "us-east-1"
	}
	modelID := b.config.Model
	if len(region) >= 2 {
		regionPrefix := region[:2]
		modelID = fmt.Sprintf("%s.%s", regionPrefix, b.config.Model)
	}
	return Model{
		ID:               modelID,
		Name:             modelID,
		DefaultMaxTokens: b.config.MaxTokens,
	}
}
