package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Service struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

func NewService(baseURL, apiKey, model string) *Service {
	if model == "" {
		model = "gpt-4o-mini"
	}
	return &Service{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		client:  &http.Client{Timeout: 20 * time.Second},
	}
}

func (s *Service) Enabled() bool {
	return s.apiKey != ""
}

type chatCompletionRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

// SuggestDescription asks an OpenAI-compatible LLM for a product description.
// Returns a fallback description when no API key is configured.
func (s *Service) SuggestDescription(ctx context.Context, code string) (string, error) {
	if !s.Enabled() {
		return s.fallback(code), nil
	}

	base := strings.TrimSuffix(s.baseURL, "/")
	if base == "" {
		base = "https://api.openai.com/v1"
	}

	payload := chatCompletionRequest{
		Model: s.model,
		Messages: []chatMessage{
			{
				Role: "system",
				Content: "Você é um especialista em catálogo de produtos. " +
					"Responda APENAS com a descrição do produto, sem aspas e sem pontuação extra. " +
					"Máximo 80 caracteres.",
			},
			{
				Role:    "user",
				Content: fmt.Sprintf("Sugira uma descrição comercial para o produto de código %q.", code),
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return s.fallback(code), err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return s.fallback(code), err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return s.fallback(code), err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return s.fallback(code), fmt.Errorf("AI provider returned %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}

	var result chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return s.fallback(code), err
	}
	if len(result.Choices) == 0 {
		return s.fallback(code), fmt.Errorf("AI provider returned no choices")
	}

	desc := strings.TrimSpace(result.Choices[0].Message.Content)
	if desc == "" {
		return s.fallback(code), nil
	}
	return desc, nil
}

func (s *Service) fallback(code string) string {
	return fmt.Sprintf("Produto %s (descrição gerada offline)", code)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}