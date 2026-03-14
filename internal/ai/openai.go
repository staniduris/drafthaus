package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type openaiProvider struct {
	apiKey string
	model  string
	client *http.Client
}

func newOpenAIProvider(cfg Config) (*openaiProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key required")
	}
	model := cfg.Model
	if model == "" {
		model = "gpt-4o-mini"
	}
	return &openaiProvider{
		apiKey: cfg.APIKey,
		model:  model,
		client: &http.Client{},
	}, nil
}

func (p *openaiProvider) Name() string { return "openai" }

func (p *openaiProvider) Complete(ctx context.Context, messages []Message) (string, error) {
	body := map[string]any{
		"model":    p.model,
		"messages": messages,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

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
		return "", fmt.Errorf("openai error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}
	return result.Choices[0].Message.Content, nil
}
