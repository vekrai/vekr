package ai

type Config struct {
	Name           string
	BaseURL        string
	APIKey         string
	Model          string
	MaxTokensField string
}

func (c Config) maxTokensField() string {
	if c.MaxTokensField == "" {
		return "max_tokens"
	}
	return c.MaxTokensField
}
