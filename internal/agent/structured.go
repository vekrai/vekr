package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"vekr/internal/ai"
)

const maxStructuredAttempts = 3

// CallStructured sends messages to the chat model and parses the response
// content as JSON into T. On a parse failure it feeds the error back to the
// model as a correction turn and retries, capped at maxStructuredAttempts. A
// truncated response (finish_reason "length") gives up immediately, since
// asking the same model for the same cut-off output again won't help.
func CallStructured[T any](ctx context.Context, client *ai.Client, messages []ai.Message) (T, error) {
	var result T
	history := append([]ai.Message(nil), messages...)
	var lastErr error

	for attempt := 1; attempt <= maxStructuredAttempts; attempt++ {
		resp, err := client.ChatCompletion(ctx, ai.ChatRequest{Messages: history})
		if err != nil {
			return result, fmt.Errorf("agent: structured call: %w", err)
		}

		if resp.FinishReason == "length" {
			return result, fmt.Errorf("agent: structured call: response truncated (hit max tokens) on attempt %d", attempt)
		}

		if err := json.Unmarshal([]byte(resp.Message.Content), &result); err == nil {
			return result, nil
		}
		lastErr = err

		history = append(history, resp.Message, ai.Message{
			Role:    ai.RoleUser,
			Content: fmt.Sprintf("That wasn't valid JSON matching the required schema: %v. Reply with only the corrected JSON object, nothing else.", err),
		})
	}

	return result, fmt.Errorf("agent: structured call: giving up after %d attempts: %w", maxStructuredAttempts, lastErr)
}
