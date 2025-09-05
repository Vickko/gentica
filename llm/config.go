package llm

import (
	"github.com/charmbracelet/catwalk/pkg/catwalk"
)

// SelectedModelType represents the model type selection
type SelectedModelType string

const (
	SelectedModelTypeSmall SelectedModelType = "small"
	SelectedModelTypeLarge SelectedModelType = "large"
)

// ProviderConfig represents provider configuration
type ProviderConfig struct {
	ID                 string
	Type               catwalk.Type
	APIKey             string
	BaseURL            string
	ExtraHeaders       map[string]string
	ExtraBody          map[string]any
	ExtraParams        map[string]string
	SystemPromptPrefix string
}

// ModelConfig represents model configuration
type ModelConfig struct {
	Model            catwalk.Model
	Provider         ProviderConfig
	MaxTokens        int64
	Think            bool
	ReasoningEffort  string
}

// Options represents global options
type Options struct {
	Debug        bool
	ContextPaths []string
}

// LSPConfig represents LSP configuration
type LSPConfig struct {
	Disabled bool
}

// Agent represents agent configuration
type Agent struct {
	ID           string
	Name         string
	Model        SelectedModelType
	MaxRetries   int
	AllowedTools []string
}

// MCPType represents the MCP connection type
type MCPType string

const (
	MCPStdio MCPType = "stdio"
	MCPHttp  MCPType = "http"
	MCPSse   MCPType = "sse"
)

// MCPConfig represents MCP server configuration
type MCPConfig struct {
	Type             MCPType
	Command          string
	Args             []string
	URL              string
	Timeout          int
	Disabled         bool
	ResolvedEnv      []string
	ResolvedHeaders  map[string]string
}

// Config represents the main configuration
type Config struct {
	Models     map[SelectedModelType]ModelConfig
	Options    Options
	LSP        map[string]LSPConfig
	MCP        map[string]MCPConfig
	Agents     map[string]Agent
	workingDir string
}

var globalConfig *Config

// Get returns the global config instance
func Get() *Config {
	if globalConfig == nil {
		// Initialize with empty config
		globalConfig = &Config{
			Models: make(map[SelectedModelType]ModelConfig),
			Options: Options{
				Debug: false,
			},
			LSP:        make(map[string]LSPConfig),
			MCP:        make(map[string]MCPConfig),
			Agents:     make(map[string]Agent),
			workingDir: ".",
		}
	}
	return globalConfig
}

// GetModelByType returns the model for the given type
func (c *Config) GetModelByType(modelType SelectedModelType) catwalk.Model {
	if model, ok := c.Models[modelType]; ok {
		return model.Model
	}
	// Return a default empty model if not found
	return catwalk.Model{}
}

// Resolve resolves a string value (placeholder for now)
func (c *Config) Resolve(value string) (string, error) {
	// Simple implementation - just return the value as-is
	return value, nil
}

// WorkingDir returns the working directory
func (c *Config) WorkingDir() string {
	return c.workingDir
}

// GetProviderForModel returns the provider config for a model type
func (c *Config) GetProviderForModel(modelType SelectedModelType) ProviderConfig {
	if model, ok := c.Models[modelType]; ok {
		return model.Provider
	}
	// Return a default empty provider if not found
	return ProviderConfig{}
}