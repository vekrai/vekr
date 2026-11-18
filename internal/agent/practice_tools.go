package agent

import (
	"context"
	"fmt"
	"strings"
)

// PracticeTools exposes practice-item generation to the chat tool loop.
// Unlike curriculum generation, this is a single-row-per-item operation
// with no graph to undo, so it commits directly — no preview/confirm step.
type PracticeTools struct {
	service *PracticeService
}

func NewPracticeTools(service *PracticeService) *PracticeTools {
	return &PracticeTools{service: service}
}

func (t *PracticeTools) GenerateItems(ctx context.Context, topicID uint, count int) (string, error) {
	items, err := t.service.GenerateItems(ctx, topicID, count)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Created %d practice item(s) for this topic:\n", len(items))
	for _, item := range items {
		fmt.Fprintf(&sb, "- [%s] %s\n", item.Type, item.Prompt)
	}
	sb.WriteString("\nThe user can answer them on the topic's practice page.")
	return sb.String(), nil
}
