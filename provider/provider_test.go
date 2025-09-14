package provider

import (
	"testing"
)

// TestNewProvider tests the creation of providers with simplified config
func TestNewProvider(t *testing.T) {
	tests := []struct {
		name   string
		config ProviderConfig
		opts   *ProviderOptions
		wantErr bool
	}{
		{
			name: "OpenAI provider",
			config: ProviderConfig{
				Type:      TypeOpenAI,
				APIKey:    "test-key",
				Model:     "gpt-4",
				MaxTokens: 4096,
			},
			opts: &ProviderOptions{
				SystemMessage: "You are a helpful assistant",
			},
			wantErr: false,
		},
		{
			name: "Anthropic provider",
			config: ProviderConfig{
				Type:      TypeAnthropic,
				APIKey:    "test-key",
				Model:     "claude-3-opus",
				MaxTokens: 4096,
			},
			opts: &ProviderOptions{
				SystemMessage: "You are a helpful assistant",
			},
			wantErr: false,
		},
		{
			name: "Gemini provider",
			config: ProviderConfig{
				Type:      TypeGemini,
				APIKey:    "test-key",
				Model:     "gemini-pro",
				MaxTokens: 4096,
			},
			opts: &ProviderOptions{
				SystemMessage: "You are a helpful assistant",
			},
			wantErr: false,
		},
		{
			name: "Unsupported provider",
			config: ProviderConfig{
				Type: "unsupported",
			},
			opts: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewProvider(tt.config, tt.opts)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewProvider() expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("NewProvider() unexpected error: %v", err)
				}
				if provider == nil {
					t.Errorf("NewProvider() returned nil provider")
				} else {
					// Verify model is set correctly
					model := provider.Model()
					if model.ID != tt.config.Model {
						t.Errorf("Expected model ID %s, got %s", tt.config.Model, model.ID)
					}
				}
			}
		})
	}
}