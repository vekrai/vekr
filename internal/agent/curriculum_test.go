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
				"message": {"role": "assistant", "content": "{\"path_title\":\"Algorithms\",\"topics\":[{\"title\":\"Arrays\",\"content\":\"Array basics\",\"prerequisites\":[]},{\"title\":\"Two pointers\",\"content\":\"Two pointer technique\",\"prerequisites\":[\"Arrays\"]}]}"},
				"finish_reason": "stop"
			}]
		}`)
	})

	preview, err := service.Preview(context.Background(), "teach me algorithms")
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if preview.PathTitle != "Algorithms" || len(preview.Topics) != 2 {
		t.Fatalf("got preview %+v, want path title Algorithms with 2 topics", preview)
	}

	path, err := service.Commit(context.Background(), preview)
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if path.Title != "Algorithms" {
		t.Errorf("got path title %q, want %q", path.Title, "Algorithms")
	}

	var topics []models.Node
	if err := db.Where("path_id = ?", path.ID).Order("id").Find(&topics).Error; err != nil {
		t.Fatalf("load topics: %v", err)
	}
	if len(topics) != 2 {
		t.Fatalf("got %d topics, want 2", len(topics))
	}
	for _, topic := range topics {
		if topic.Type != models.NodeTypeTopic {
			t.Errorf("got topic %q type %q, want %q", topic.Title, topic.Type, models.NodeTypeTopic)
		}
		if topic.Status == nil || *topic.Status != models.StatusNotStarted {
			t.Errorf("got topic %q status %v, want %q", topic.Title, topic.Status, models.StatusNotStarted)
		}
	}

	topicIDs := []uint{topics[0].ID, topics[1].ID}
	var links []models.NodeLink
	if err := db.Where("from_node_id IN ? AND to_node_id IN ?", topicIDs, topicIDs).Find(&links).Error; err != nil {
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

	promoted, err := service.PromoteToTopic(context.Background(), node.ID)
	if err != nil {
		t.Fatalf("PromoteToTopic: %v", err)
	}
	if promoted.Type != models.NodeTypeTopic {
		t.Errorf("got type %q, want %q", promoted.Type, models.NodeTypeTopic)
	}
	if promoted.Status == nil || *promoted.Status != models.StatusNotStarted {
		t.Errorf("got status %v, want %q", promoted.Status, models.StatusNotStarted)
	}
	if promoted.PathID != nil {
		t.Errorf("got PathID %v, want nil (promote does not assign a path)", promoted.PathID)
	}
}
