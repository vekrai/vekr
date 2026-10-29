package agent

import (
	"context"
	"encoding/gob"
	"fmt"
	"strings"

	"github.com/alexedwards/scs/v2"
)

const curriculumPreviewSessionKey = "curriculum_preview"

func init() {
	// scs gob-encodes session data on commit — the concrete type held
	// under curriculumPreviewSessionKey must be registered so the decoder
	// knows what to decode it back into.
	gob.Register(CurriculumPreview{})
}

// CurriculumTools exposes curriculum generation to the chat tool loop as
// two bounded calls rather than one, preserving the preview-then-commit
// guardrail for graph-shaped output: the model must get the user's
// confirmation on the preview before a second, explicit call persists it.
// The preview is held in the chat session between the two calls, never
// written to the database until commit.
type CurriculumTools struct {
	service  *CurriculumService
	sessions *scs.SessionManager
}

func NewCurriculumTools(service *CurriculumService, sessions *scs.SessionManager) *CurriculumTools {
	return &CurriculumTools{service: service, sessions: sessions}
}

func (t *CurriculumTools) GeneratePreview(ctx context.Context, intake string) (string, error) {
	preview, err := t.service.Preview(ctx, intake)
	if err != nil {
		return "", err
	}
	t.sessions.Put(ctx, curriculumPreviewSessionKey, preview)

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
	preview, ok := t.sessions.Get(ctx, curriculumPreviewSessionKey).(CurriculumPreview)
	if !ok {
		return "", fmt.Errorf("no pending curriculum preview to commit — call generate_curriculum_preview first")
	}

	path, err := t.service.Commit(ctx, preview)
	if err != nil {
		return "", err
	}
	t.sessions.Remove(ctx, curriculumPreviewSessionKey)

	return fmt.Sprintf("Committed learning path %q (id=%d) with %d topics.", path.Title, path.ID, len(preview.Topics)), nil
}
