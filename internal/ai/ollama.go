package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type ollamaProvider struct {
	baseURL string
	model   string
	client  *http.Client
}

func newOllamaProvider(cfg Config) (*ollamaProvider, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	model := cfg.Model
	if model == "" {
		model = "llama3.2"
	}
	return &ollamaProvider{
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{},
	}, nil
}

func (p *ollamaProvider) Name() string { return "ollama" }

func (p *ollamaProvider) Complete(ctx context.Context, messages []Message) (string, error) {
	body := map[string]any{
		"model":    p.model,
		"messages": messages,
		"stream":   false,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	endpoint := p.baseURL + "/api/chat"
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

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
		return "", fmt.Errorf("ollama error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}
	if result.Message.Content == "" {
		return "", fmt.Errorf("empty content in response")
	}
	return result.Message.Content, nil
}
