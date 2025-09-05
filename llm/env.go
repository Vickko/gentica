package llm

import (
	"os"
	"strings"
)

// Env provides environment variable operations
type Env struct{}

// New creates a new Env instance
func NewEnv() *Env {
	return &Env{}
}

// NewEnvironmentVariableResolver creates a resolver for environment variables
func NewEnvironmentVariableResolver(env *Env) *EnvironmentVariableResolver {
	return &EnvironmentVariableResolver{env: env}
}

// EnvironmentVariableResolver resolves environment variables
type EnvironmentVariableResolver struct {
	env *Env
}

// ResolveValue resolves a value that may contain environment variables
func (r *EnvironmentVariableResolver) ResolveValue(value string) (string, error) {
	if strings.HasPrefix(value, "$") {
		envVar := value[1:]
		return os.Getenv(envVar), nil
	}
	return value, nil
}