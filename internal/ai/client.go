package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const logBodyLimit = 4096

func truncateForLog(b []byte) string {
	if len(b) <= logBodyLimit {
		return string(b)
	}
	return string(b[:logBodyLimit]) + "...(truncated)"
}

type Client struct {
	config     Config
	httpClient *http.Client
}

func NewClient(config Config, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 2 * time.Minute}
	}
	return &Client{config: config, httpClient: httpClient}
}

func (c *Client) payload(req ChatRequest, stream bool) map[string]any {
	payload := map[string]any{
		"model":    c.config.Model,
		"messages": req.Messages,
		"stream":   stream,
	}
	if len(req.Tools) > 0 {
		payload["tools"] = req.Tools
	}
	if req.Temperature != nil {
		payload["temperature"] = *req.Temperature
	}
	if req.MaxTokens > 0 {
		payload[c.config.maxTokensField()] = req.MaxTokens
	}
	return payload
}

func (c *Client) newRequest(ctx context.Context, body map[string]any) (*http.Request, error) {
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("ai: encode request: %w", err)
	}
	log.Printf("ai: request model=%s url=%s body=%s", c.config.Model, c.config.BaseURL+"/chat/completions", truncateForLog(encoded))

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.BaseURL+"/chat/completions", bytes.NewReader(encoded))
	if err != nil {
		return nil, fmt.Errorf("ai: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}
	return httpReq, nil
}

func readErrorBody(resp *http.Response) string {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	return string(body)
}

type wireResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Message      Message `json:"message"`
		FinishReason string  `json:"finish_reason"`
	} `json:"choices"`
	Usage Usage `json:"usage"`
}

func (c *Client) ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	httpReq, err := c.newRequest(ctx, c.payload(req, false))
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ai: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ai: unexpected status %d: %s", resp.StatusCode, readErrorBody(resp))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ai: read response: %w", err)
	}
	log.Printf("ai: response status=%d body=%s", resp.StatusCode, truncateForLog(respBody))

	var wire wireResponse
	if err := json.Unmarshal(respBody, &wire); err != nil {
		return nil, fmt.Errorf("ai: decode response: %w", err)
	}
	if len(wire.Choices) == 0 {
		return nil, fmt.Errorf("ai: response had no choices")
	}

	choice := wire.Choices[0]
	return &ChatResponse{
		ID:           wire.ID,
		Model:        wire.Model,
		Message:      choice.Message,
		FinishReason: choice.FinishReason,
		Usage:        wire.Usage,
	}, nil
}
