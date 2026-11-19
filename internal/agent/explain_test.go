package agent

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vekr/internal/ai"
	"vekr/internal/models"
)

func TestExplainReviewSavesNoteUnderTopic(t *testing.T) {
	db := testDB(t)

	topic := models.Topic{Title: "Since vs. For", Content: "since + start point, for + duration"}
	if err := db.Create(&topic).Error; err != nil {
		t.Fatalf("create topic: %v", err)
	}

	var sentPrompt string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		sentPrompt = string(body)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices": [{"message": {"role": "assistant", "content": "{\"feedback\":\"Right about since.\"}"}, "finish_reason": "stop"}]}`)
	}))
	t.Cleanup(server.Close)
	service := NewExplainService(db, ai.NewClient(ai.Config{BaseURL: server.URL, Model: "test"}, nil))

	node, err := service.Review(context.Background(), topic.ID, "since is for a starting point")
	if err != nil {
		t.Fatalf("Review: %v", err)
	}

	if !strings.Contains(sentPrompt, "since is for a starting point") {
		t.Errorf("want the explanation sent to the model, got %s", sentPrompt)
	}
	if node.TopicID == nil || *node.TopicID != topic.ID {
		t.Errorf("got topic id %v, want %d", node.TopicID, topic.ID)
	}
	if !strings.Contains(node.Title, "Since vs. For") {
		t.Errorf("got title %q, want it to name the topic", node.Title)
	}
	if !strings.Contains(node.Content, "since is for a starting point") || !strings.Contains(node.Content, "Right about since.") {
		t.Errorf("got content %q, want explanation and feedback", node.Content)
	}
}

func TestExplainReviewRejectsEmptyExplanation(t *testing.T) {
	db := testDB(t)
	service := NewExplainService(db, ai.NewClient(ai.Config{BaseURL: "http://unused", Model: "test"}, nil))

	if _, err := service.Review(context.Background(), 1, "   "); err == nil {
		t.Error("want an error for an empty explanation")
	}
}
