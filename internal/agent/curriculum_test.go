package agent

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"gorm.io/gorm"

	"vekr/internal/ai"
	"vekr/internal/models"
)

func newCurriculumService(t *testing.T, db *gorm.DB, handler http.HandlerFunc) *CurriculumService {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client := ai.NewClient(ai.Config{BaseURL: server.URL, Model: "test"}, nil)
	return NewCurriculumService(db, client)
}

func TestCurriculumPreviewAndCommit(t *testing.T) {
	db := testDB(t)

	service := newCurriculumService(t, db, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"id": "resp-1",
			"choices": [{
				"message": {"role": "assistant", "content": "{\"topic_title\":\"Algorithms\",\"topic_content\":\"Core algorithms\",\"notes\":[{\"title\":\"Arrays\",\"content\":\"Array basics\",\"prerequisites\":[]},{\"title\":\"Two pointers\",\"content\":\"Two pointer technique\",\"prerequisites\":[\"Arrays\"]}]}"},
				"finish_reason": "stop"
			}]
		}`)
	})

	preview, err := service.Preview(context.Background(), "teach me algorithms")
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if preview.TopicTitle != "Algorithms" || len(preview.Notes) != 2 {
		t.Fatalf("got preview %+v, want topic title Algorithms with 2 notes", preview)
	}

	topic, err := service.Commit(context.Background(), preview)
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if topic.Title != "Algorithms" || topic.Content != "Core algorithms" {
		t.Errorf("got topic %q / %q, want Algorithms / Core algorithms", topic.Title, topic.Content)
	}
	if topic.Status != models.StatusNotStarted {
		t.Errorf("got topic status %q, want %q", topic.Status, models.StatusNotStarted)
	}

	var notes []models.Node
	if err := db.Where("topic_id = ?", topic.ID).Order("id").Find(&notes).Error; err != nil {
		t.Fatalf("load notes: %v", err)
	}
	if len(notes) != 2 {
		t.Fatalf("got %d notes, want 2", len(notes))
	}

	noteIDs := []uint{notes[0].ID, notes[1].ID}
	var links []models.NodeLink
	if err := db.Where("from_node_id IN ? AND to_node_id IN ?", noteIDs, noteIDs).Find(&links).Error; err != nil {
		t.Fatalf("load node links: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("got %d node_links, want 1 (Arrays -> Two pointers)", len(links))
	}
	if links[0].Relation != models.RelationPrerequisiteOf {
		t.Errorf("got relation %q, want %q", links[0].Relation, models.RelationPrerequisiteOf)
	}
}

func TestPromoteToTopic(t *testing.T) {
	db := testDB(t)
	service := NewCurriculumService(db, nil)

	node := models.Node{Title: "Plain note", Content: "just a note"}
	if err := db.Create(&node).Error; err != nil {
		t.Fatalf("create node: %v", err)
	}

	topic, err := service.PromoteToTopic(context.Background(), node.ID)
	if err != nil {
		t.Fatalf("PromoteToTopic: %v", err)
	}
	if topic.Title != "Plain note" || topic.Status != models.StatusNotStarted {
		t.Errorf("got topic %q status %q, want Plain note / %q", topic.Title, topic.Status, models.StatusNotStarted)
	}

	var reloaded models.Node
	if err := db.First(&reloaded, node.ID).Error; err != nil {
		t.Fatalf("reload node: %v", err)
	}
	if reloaded.TopicID == nil || *reloaded.TopicID != topic.ID {
		t.Errorf("got note TopicID %v, want %d", reloaded.TopicID, topic.ID)
	}

	again, err := service.PromoteToTopic(context.Background(), node.ID)
	if err != nil {
		t.Fatalf("PromoteToTopic (again): %v", err)
	}
	if again.ID != topic.ID {
		t.Errorf("got topic %d on second promote, want the existing topic %d", again.ID, topic.ID)
	}
}
