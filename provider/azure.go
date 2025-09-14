package provider

import (
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/azure"
	"github.com/openai/openai-go/option"
)

type azureClient struct {
	*openaiClient
}

func newAzureClient(config ProviderConfig, opts *ProviderOptions) Provider {
	apiVersion := config.ExtraParams["apiVersion"]
	if apiVersion == "" {
		apiVersion = "2025-01-01-preview"
	}

	reqOpts := []option.RequestOption{
		azure.WithEndpoint(config.BaseURL, apiVersion),
	}

	reqOpts = append(reqOpts, azure.WithAPIKey(config.APIKey))
	base := &openaiClient{
		config: config,
		opts:   opts,
		client: openai.NewClient(reqOpts...),
	}

	return &azureClient{openaiClient: base}
}
