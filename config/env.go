package config

import (
	"os"
	"strings"
)

// Env provides environment variable management
type Env struct {
	vars map[string]string
}

// NewEnv creates a new Env from current environment
func NewEnv() *Env {
	return &Env{
		vars: envToMap(os.Environ()),
	}
}

// NewEnvFromMap creates a new Env from a map
func NewEnvFromMap(vars map[string]string) *Env {
	return &Env{
		vars: vars,
	}
}

// Get retrieves an environment variable
func (e *Env) Get(key string) string {
	if e.vars != nil {
		return e.vars[key]
	}
	return os.Getenv(key)
}

// Set sets an environment variable
func (e *Env) Set(key, value string) {
	if e.vars == nil {
		e.vars = make(map[string]string)
	}
	e.vars[key] = value
}

// Env returns the environment as a slice of "key=value" strings
func (e *Env) Env() []string {
	if e.vars == nil {
		return os.Environ()
	}
	var env []string
	for k, v := range e.vars {
		env = append(env, k+"="+v)
	}
	return env
}

// envToMap converts environment slice to map
func envToMap(environ []string) map[string]string {
	m := make(map[string]string)
	for _, e := range environ {
		if idx := strings.Index(e, "="); idx > 0 {
			m[e[:idx]] = e[idx+1:]
		}
	}
	return m
}