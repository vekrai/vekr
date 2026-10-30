package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// CurriculumTools exposes curriculum generation to the chat tool loop as
// two bounded calls rather than one, preserving the preview-then-commit
// guardrail for graph-shaped output: the model must get the user's
// confirmation on the preview before a second, explicit call persists it.
// The preview is held in memory per chat between the two calls, never
// written to the database until commit.
type CurriculumTools struct {
	service *CurriculumService

	mu       sync.Mutex
	previews map[uint]CurriculumPreview
}

func NewCurriculumTools(service *CurriculumService) *CurriculumTools {
	return &CurriculumTools{service: service, previews: make(map[uint]CurriculumPreview)}
}

func (t *CurriculumTools) GeneratePreview(ctx context.Context, intake string) (string, error) {
	chatID, ok := chatIDFromContext(ctx)
	if !ok {
		return "", fmt.Errorf("curriculum preview needs a chat to hold it")
	}

	preview, err := t.service.Preview(ctx, intake)
	if err != nil {
		return "", err
	}
	t.mu.Lock()
	t.previews[chatID] = preview
	t.mu.Unlock()

	var sb strings.Builder
	fmt.Fprintf(&sb, "Proposed path: %s\n\n", preview.PathTitle)
	for _, topic := range preview.Topics {
		fmt.Fprintf(&sb, "- %s", topic.Title)
		if len(topic.Prerequisites) > 0 {
			fmt.Fprintf(&sb, " (requires: %s)", strings.Join(topic.Prerequisites, ", "))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\nNothing has been saved yet. Show this to the user and only call commit_curriculum_preview after they explicitly confirm.")
	return sb.String(), nil
}

func (t *CurriculumTools) CommitPreview(ctx context.Context) (string, error) {
	chatID, _ := chatIDFromContext(ctx)
	t.mu.Lock()
	preview, ok := t.previews[chatID]
	t.mu.Unlock()
	if !ok {
		return "", fmt.Errorf("no pending curriculum preview to commit — call generate_curriculum_preview first")
	}

	path, err := t.service.Commit(ctx, preview)
	if err != nil {
		return "", err
	}
	t.mu.Lock()
	delete(t.previews, chatID)
	t.mu.Unlock()

	return fmt.Sprintf("Committed learning path %q (id=%d) with %d topics.", path.Title, path.ID, len(preview.Topics)), nil
}
