package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// OpenAIClient adalah client HTTP ke OpenAI Chat Completions.
type OpenAIClient struct {
	endpoint string
	key      string
	model    string
	http     *http.Client
}

// NewOpenAIClient membuat OpenAIClient.
func NewOpenAIClient(endpoint, key, model string) *OpenAIClient {
	return &OpenAIClient{endpoint: endpoint, key: key, model: model, http: &http.Client{Timeout: 120 * time.Second}}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

// Chat mengirim prompt ke Chat Completions dan mengembalikan isi balasan pertama.
func (c *OpenAIClient) Chat(ctx context.Context, system, user string) (string, error) {
	body, _ := json.Marshal(chatRequest{
		Model:    c.model,
		Messages: []chatMessage{{Role: "system", Content: system}, {Role: "user", Content: user}},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.key)
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("openai returned status %d", resp.StatusCode)
	}
	var out chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", errors.New("openai returned no choices")
	}
	return out.Choices[0].Message.Content, nil
}
