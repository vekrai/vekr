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

func newPracticeService(t *testing.T, db *gorm.DB, handler http.HandlerFunc) *PracticeService {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client := ai.NewClient(ai.Config{BaseURL: server.URL, Model: "test"}, nil)
	return NewPracticeService(db, client)
}

func TestPracticeGenerateItemsCreatesItemsAndConceptLinks(t *testing.T) {
	db := testDB(t)

	topic := models.Node{Title: "Binary search", Content: "Search a sorted array in O(log n).", Type: models.NodeTypeTopic}
	if err := db.Create(&topic).Error; err != nil {
		t.Fatalf("create topic: %v", err)
	}

	service := newPracticeService(t, db, func(w http.ResponseWriter, r *http.Request) {
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

	var concepts []models.Concept
	if err := db.Where("name IN ?", []string{"binary-search", "pointers"}).Find(&concepts).Error; err != nil {
		t.Fatalf("load concepts: %v", err)
	}
	if len(concepts) != 2 {
		t.Fatalf("got %d concepts, want 2 (deduplicated)", len(concepts))
	}

	var nodeConceptCount int64
	if err := db.Model(&models.NodeConcept{}).Where("node_id = ?", topic.ID).Count(&nodeConceptCount).Error; err != nil {
		t.Fatalf("count node_concepts: %v", err)
	}
	if nodeConceptCount != 2 {
		t.Errorf("got %d node_concepts, want 2", nodeConceptCount)
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

	topic := models.Node{Title: "Topic A", Content: "content", Type: models.NodeTypeTopic}
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

	topic := models.Node{Title: "Topic", Content: "content", Type: models.NodeTypeTopic}
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
