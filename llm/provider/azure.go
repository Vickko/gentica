package provider

import (
	// "github.com/charmbracelet/crush/internal/config"
	"gentica/llm" // for config
	"net/http"
	// "github.com/charmbracelet/crush/internal/log"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/azure"
	"github.com/openai/openai-go/option"
)

type azureClient struct {
	*openaiClient
}

type AzureClient ProviderClient

func newAzureClient(opts providerClientOptions) AzureClient {
	apiVersion := opts.extraParams["apiVersion"]
	if apiVersion == "" {
		apiVersion = "2025-01-01-preview"
	}

	reqOpts := []option.RequestOption{
		azure.WithEndpoint(opts.baseURL, apiVersion),
	}

	if llm.Get().Options.Debug {
		httpClient := &http.Client{}
		reqOpts = append(reqOpts, option.WithHTTPClient(httpClient))
	}

	reqOpts = append(reqOpts, azure.WithAPIKey(opts.apiKey))
	base := &openaiClient{
		providerOptions: opts,
		client:          openai.NewClient(reqOpts...),
	}

	return &azureClient{openaiClient: base}
}
