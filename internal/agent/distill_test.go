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

func newDistillService(t *testing.T, db *gorm.DB, handler http.HandlerFunc) *DistillService {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client := ai.NewClient(ai.Config{BaseURL: server.URL, Model: "test"}, nil)
	return NewDistillService(db, client)
}

func TestDistillCreatesNoteLinkedToChat(t *testing.T) {
	db := testDB(t)

	chat := models.Chat{}
	if err := db.Create(&chat).Error; err != nil {
		t.Fatalf("create chat: %v", err)
	}
	messages := []models.ChatMessage{
		{ChatID: chat.ID, Role: "user", Content: "explain go generics"},
		{ChatID: chat.ID, Role: "assistant", Content: "generics let you write functions over any type"},
	}
	for i := range messages {
		if err := db.Create(&messages[i]).Error; err != nil {
			t.Fatalf("create chat message: %v", err)
		}
	}

	service := newDistillService(t, db, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"id": "resp-1",
			"choices": [{
				"message": {"role": "assistant", "content": "{\"title\":\"Go generics\",\"content\":\"Generics let you write functions over any type.\"}"},
				"finish_reason": "stop"
			}]
		}`)
	})

	node, err := service.Distill(context.Background(), chat.ID)
	if err != nil {
		t.Fatalf("Distill: %v", err)
	}
	if node.Title != "Go generics" {
		t.Errorf("got title %q, want %q", node.Title, "Go generics")
	}
	if node.SourceChatID == nil || *node.SourceChatID != chat.ID {
		t.Errorf("got SourceChatID %v, want %d", node.SourceChatID, chat.ID)
	}

	var saved models.Node
	if err := db.First(&saved, node.ID).Error; err != nil {
		t.Fatalf("load saved node: %v", err)
	}
	if saved.Content != "Generics let you write functions over any type." {
		t.Errorf("got saved content %q, want the distilled body", saved.Content)
	}
}

func TestDistillErrorsOnEmptyChat(t *testing.T) {
	db := testDB(t)

	chat := models.Chat{}
	if err := db.Create(&chat).Error; err != nil {
		t.Fatalf("create chat: %v", err)
	}

	service := newDistillService(t, db, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("model should not be called for a chat with no messages")
	})

	if _, err := service.Distill(context.Background(), chat.ID); err == nil {
		t.Fatal("expected an error for a chat with no messages")
	}
}
