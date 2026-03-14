package ai

import (
	"context"
	"fmt"
)

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`    // "system", "user", "assistant"
	Content string `json:"content"`
}

// Provider is the interface for AI model backends.
type Provider interface {
	// Complete sends messages and returns the assistant's response.
	Complete(ctx context.Context, messages []Message) (string, error)
	// Name returns the provider name for logging.
	Name() string
}

// Config holds AI provider configuration.
type Config struct {
	Provider string // "openai", "anthropic", "ollama", "none"
	APIKey   string
	Model    string
	BaseURL  string // for Ollama or custom endpoints
}

// NewProvider creates a Provider from config.
func NewProvider(cfg Config) (Provider, error) {
	switch cfg.Provider {
	case "openai":
		return newOpenAIProvider(cfg)
	case "anthropic":
		return newAnthropicProvider(cfg)
	case "ollama":
		return newOllamaProvider(cfg)
	case "none", "":
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown AI provider: %s", cfg.Provider)
	}
}
