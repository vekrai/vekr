package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const tavilySearchURL = "https://api.tavily.com/search"

type WebSearchTool struct {
	apiKey     string
	searchURL  string
	httpClient *http.Client
}

func NewWebSearchTool(apiKey string, httpClient *http.Client) *WebSearchTool {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &WebSearchTool{apiKey: apiKey, searchURL: tavilySearchURL, httpClient: httpClient}
}

func (t *WebSearchTool) Search(ctx context.Context, query string) (string, error) {
	body := map[string]any{"query": query, "max_results": 5}
	if t.apiKey != "" {
		body["api_key"] = t.apiKey
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("websearch: encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.searchURL, bytes.NewReader(encoded))
	if err != nil {
		return "", fmt.Errorf("websearch: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if t.apiKey == "" {
		req.Header.Set("X-Tavily-Access-Mode", "keyless")
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("websearch: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", fmt.Errorf("web search unavailable, configure a Tavily API key")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("websearch: unexpected status %d", resp.StatusCode)
	}

	var parsed struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("websearch: decode response: %w", err)
	}

	if len(parsed.Results) == 0 {
		return "no results found", nil
	}

	var sb strings.Builder
	for i, r := range parsed.Results {
		fmt.Fprintf(&sb, "%d. %s\n%s\n%s\n\n", i+1, r.Title, r.URL, r.Content)
	}
	return strings.TrimSpace(sb.String()), nil
}
