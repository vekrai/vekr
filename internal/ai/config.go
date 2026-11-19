package ai

import (
	"os"
	"strconv"
)

type Config struct {
	Name           string
	BaseURL        string
	APIKey         string
	Model          string
	MaxTokensField string
	ContextWindow  int
}

func (c Config) maxTokensField() string {
	if c.MaxTokensField == "" {
		return "max_tokens"
	}
	return c.MaxTokensField
}

func ConfigFromEnv() Config {
	baseURL := os.Getenv("AI_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11434/v1"
	}
	model := os.Getenv("AI_MODEL")
	if model == "" {
		model = "qwen"
	}

	contextWindow, _ := strconv.Atoi(os.Getenv("AI_CONTEXT_WINDOW"))

	return Config{
		Name:           "local-qwen",
		BaseURL:        baseURL,
		APIKey:         os.Getenv("AI_API_KEY"),
		Model:          model,
		MaxTokensField: os.Getenv("AI_MAX_TOKENS_FIELD"),
		ContextWindow:  contextWindow,
	}
}
