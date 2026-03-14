package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type anthropicProvider struct {
	apiKey string
	model  string
	client *http.Client
}

func newAnthropicProvider(cfg Config) (*anthropicProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("Anthropic API key required")
	}
	model := cfg.Model
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}
	return &anthropicProvider{
		apiKey: cfg.APIKey,
		model:  model,
		client: &http.Client{},
	}, nil
}

func (p *anthropicProvider) Name() string { return "anthropic" }

func (p *anthropicProvider) Complete(ctx context.Context, messages []Message) (string, error) {
	// Separate system message from the messages array.
	systemPrompt := ""
	var chatMessages []Message
	for _, m := range messages {
		if m.Role == "system" {
			systemPrompt = m.Content
		} else {
			chatMessages = append(chatMessages, m)
		}
	}

	body := map[string]any{
		"model":      p.model,
		"max_tokens": 4096,
		"messages":   chatMessages,
	}
	if systemPrompt != "" {
		body["system"] = systemPrompt
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")
	// OAuth tokens (sk-ant-oat01-) use Bearer auth; standard API keys use x-api-key
	if strings.HasPrefix(p.apiKey, "sk-ant-oat01-") {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	} else {
		req.Header.Set("x-api-key", p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("api call: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("anthropic error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}
	for _, block := range result.Content {
		if block.Type == "text" {
			return block.Text, nil
		}
	}
	return "", fmt.Errorf("no text content in response")
}
