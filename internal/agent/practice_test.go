package agent

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gorm.io/gorm"

	"vekr/internal/ai"
	"vekr/internal/models"
)

func newPracticeService(t *testing.T, db *gorm.DB, handler http.HandlerFunc) *PracticeService {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client := ai.NewClient(ai.Config{BaseURL: server.URL, Model: "test"}, nil)
	return NewPracticeService(db, client)
}

func TestPracticeGenerateItemsCreatesItemsAndConceptLinks(t *testing.T) {
	db := testDB(t)

	topic := models.Topic{Title: "Binary search", Content: "Search a sorted array in O(log n)."}
	if err := db.Create(&topic).Error; err != nil {
		t.Fatalf("create topic: %v", err)
	}
	note := models.Node{Title: "Loop invariant", Content: "lo <= target <= hi", TopicID: &topic.ID}
	if err := db.Create(&note).Error; err != nil {
		t.Fatalf("create note: %v", err)
	}

	var body string
	service := newPracticeService(t, db, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body = string(raw)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"id": "resp-1",
			"choices": [{
				"message": {"role": "assistant", "content": "{\"items\":[{\"prompt\":\"What is the time complexity?\",\"answer_key\":\"O(log n)\",\"type\":\"flashcard\",\"concepts\":[\"binary-search\"]},{\"prompt\":\"Implement it\",\"answer_key\":\"\",\"type\":\"code\",\"concepts\":[\"binary-search\",\"pointers\"]}]}"},
				"finish_reason": "stop"
			}]
		}`)
	})

	items, err := service.GenerateItems(context.Background(), topic.ID, 2)
	if err != nil {
		t.Fatalf("GenerateItems: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	if !strings.Contains(body, "Loop invariant") {
		t.Errorf("request body missing the topic's note content:\n%s", body)
	}

	var concepts []models.Concept
	if err := db.Where("name IN ?", []string{"binary-search", "pointers"}).Find(&concepts).Error; err != nil {
		t.Fatalf("load concepts: %v", err)
	}
	if len(concepts) != 2 {
		t.Fatalf("got %d concepts, want 2 (deduplicated)", len(concepts))
	}

	for _, c := range concepts {
		if c.TopicID == nil || *c.TopicID != topic.ID {
			t.Errorf("got concept %q TopicID %v, want %d", c.Name, c.TopicID, topic.ID)
		}
	}
	for _, item := range items {
		if item.TopicID != topic.ID {
			t.Errorf("got item TopicID %d, want %d", item.TopicID, topic.ID)
		}
	}

	var itemConceptCount int64
	if err := db.Model(&models.PracticeItemConcept{}).Where("practice_item_id = ?", items[1].ID).Count(&itemConceptCount).Error; err != nil {
		t.Fatalf("count practice_item_concepts: %v", err)
	}
	if itemConceptCount != 2 {
		t.Errorf("got %d practice_item_concepts for the code item, want 2", itemConceptCount)
	}
}

func TestPracticeGenerateItemsReusesExistingConcept(t *testing.T) {
	db := testDB(t)

	topic := models.Topic{Title: "Topic A", Content: "content"}
	if err := db.Create(&topic).Error; err != nil {
		t.Fatalf("create topic: %v", err)
	}

	callCount := 0
	service := newPracticeService(t, db, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"id": "resp",
			"choices": [{
				"message": {"role": "assistant", "content": "{\"items\":[{\"prompt\":\"Q\",\"answer_key\":\"A\",\"type\":\"flashcard\",\"concepts\":[\"shared-concept\"]}]}"},
				"finish_reason": "stop"
			}]
		}`)
	})

	if _, err := service.GenerateItems(context.Background(), topic.ID, 1); err != nil {
		t.Fatalf("GenerateItems (first): %v", err)
	}
	if _, err := service.GenerateItems(context.Background(), topic.ID, 1); err != nil {
		t.Fatalf("GenerateItems (second): %v", err)
	}

	var count int64
	if err := db.Model(&models.Concept{}).Where("name = ?", "shared-concept").Count(&count).Error; err != nil {
		t.Fatalf("count concepts: %v", err)
	}
	if count != 1 {
		t.Errorf("got %d concept rows for the same name, want 1 (reused, not duplicated)", count)
	}
}

func TestPracticeGenerateItemsSkipsUnknownType(t *testing.T) {
	db := testDB(t)

	topic := models.Topic{Title: "Topic", Content: "content"}
	if err := db.Create(&topic).Error; err != nil {
		t.Fatalf("create topic: %v", err)
	}

	service := newPracticeService(t, db, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"id": "resp",
			"choices": [{
				"message": {"role": "assistant", "content": "{\"items\":[{\"prompt\":\"Bad\",\"answer_key\":\"x\",\"type\":\"multiple_choice\",\"concepts\":[]},{\"prompt\":\"Good\",\"answer_key\":\"B\",\"type\":\"mcq\",\"concepts\":[]}]}"},
				"finish_reason": "stop"
			}]
		}`)
	})

	items, err := service.GenerateItems(context.Background(), topic.ID, 2)
	if err != nil {
		t.Fatalf("GenerateItems: %v", err)
	}
	if len(items) != 1 || items[0].Prompt != "Good" {
		t.Errorf("got items %+v, want only the mcq item — unknown types should be skipped, not fail the batch", items)
	}
}
