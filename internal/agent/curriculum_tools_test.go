package agent

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexedwards/scs/v2"
	"gorm.io/gorm"

	"vekr/internal/ai"
	"vekr/internal/models"
)

func newCurriculumTools(t *testing.T, db *gorm.DB, handler http.HandlerFunc) (*CurriculumTools, context.Context) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client := ai.NewClient(ai.Config{BaseURL: server.URL, Model: "test"}, nil)
	service := NewCurriculumService(db, client)
	sessions := scs.New()

	ctx, err := sessions.Load(context.Background(), "")
	if err != nil {
		t.Fatalf("load session: %v", err)
	}

	return NewCurriculumTools(service, sessions), ctx
}

func TestCurriculumToolsGenerateThenCommit(t *testing.T) {
	db := testDB(t)

	tools, ctx := newCurriculumTools(t, db, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"id": "resp-1",
			"choices": [{
				"message": {"role": "assistant", "content": "{\"path_title\":\"Binary search\",\"topics\":[{\"title\":\"Sorted arrays\",\"content\":\"basics\",\"prerequisites\":[]}]}"},
				"finish_reason": "stop"
			}]
		}`)
	})

	summary, err := tools.GeneratePreview(ctx, "teach me binary search")
	if err != nil {
		t.Fatalf("GeneratePreview: %v", err)
	}
	if !strings.Contains(summary, "Binary search") || !strings.Contains(summary, "Sorted arrays") {
		t.Errorf("got summary %q, want it to mention the path and topic", summary)
	}
	if !strings.Contains(summary, "confirm") {
		t.Errorf("got summary %q, want it to tell the model to wait for confirmation", summary)
	}

	result, err := tools.CommitPreview(ctx)
	if err != nil {
		t.Fatalf("CommitPreview: %v", err)
	}
	if !strings.Contains(result, "Binary search") {
		t.Errorf("got commit result %q, want it to mention the path title", result)
	}

	var path models.Path
	if err := db.Where("title = ?", "Binary search").First(&path).Error; err != nil {
		t.Fatalf("expected the path to be persisted: %v", err)
	}

	if _, err := tools.CommitPreview(ctx); err == nil {
		t.Fatal("expected a second commit with no pending preview to fail")
	}
}

func TestCurriculumToolsCommitWithoutPreviewFails(t *testing.T) {
	db := testDB(t)

	tools, ctx := newCurriculumTools(t, db, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("model should not be called")
	})

	if _, err := tools.CommitPreview(ctx); err == nil {
		t.Fatal("expected an error when no preview has been generated yet")
	}
}
