package provider

// Type represents the provider type (e.g., OpenAI, Anthropic, etc.)
type Type string

// Provider type constants
const (
	TypeAnthropic Type = "anthropic"
	TypeOpenAI    Type = "openai"
	TypeGemini    Type = "gemini"
	TypeBedrock   Type = "bedrock"
	TypeAzure     Type = "azure"
	TypeVertexAI  Type = "vertexai"
)

// Model represents a language model with its metadata
type Model struct {
	// The model identifier
	ID string `json:"id,omitempty"`
	// Human-readable model name
	Name string `json:"name,omitempty"`
	// Default maximum tokens for the model
	DefaultMaxTokens int64 `json:"default_max_tokens,omitempty"`
	// Context window size in tokens
	ContextWindow int `json:"context_window,omitempty"`
	// Cost per 1 million input tokens
	CostPer1MIn float64 `json:"cost_per_1m_in,omitempty"`
	// Cost per 1 million output tokens
	CostPer1MOut float64 `json:"cost_per_1m_out,omitempty"`
}