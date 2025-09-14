package provider

// ProviderConfig holds configuration for a provider
type ProviderConfig struct {
	// The provider type, e.g. "openai", "anthropic", etc.
	Type Type `json:"type,omitempty"`
	// The provider's API endpoint.
	BaseURL string `json:"base_url,omitempty"`
	// The provider's API key.
	APIKey string `json:"api_key,omitempty"`
	// The model to use
	Model string `json:"model,omitempty"`
	// Max tokens for the model
	MaxTokens int64 `json:"max_tokens,omitempty"`
	// Custom system prompt prefix.
	SystemPromptPrefix string `json:"system_prompt_prefix,omitempty"`
	// Extra headers to send with each request to the provider.
	ExtraHeaders map[string]string `json:"extra_headers,omitempty"`
	// Extra body
	ExtraBody map[string]any `json:"extra_body,omitempty"`
	// Used to pass extra parameters to the provider.
	ExtraParams map[string]string `json:"extra_params,omitempty"`
	// Reasoning effort for OpenAI models
	ReasoningEffort string `json:"reasoning_effort,omitempty"`
	// Whether the model should think (for Anthropic)
	Think bool `json:"think,omitempty"`
}