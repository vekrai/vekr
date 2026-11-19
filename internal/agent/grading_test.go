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

func newGradingService(t *testing.T, db *gorm.DB, handler http.HandlerFunc) *GradingService {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client := ai.NewClient(ai.Config{BaseURL: server.URL, Model: "test"}, nil)
	return NewGradingService(db, client)
}

func seedPracticeItem(t *testing.T, db *gorm.DB, itemType string, answerKey *string, conceptNames ...string) models.PracticeItem {
	t.Helper()

	topic := models.Topic{Title: "Topic", Content: "content"}
	if err := db.Create(&topic).Error; err != nil {
		t.Fatalf("create topic: %v", err)
	}

	item := models.PracticeItem{TopicID: topic.ID, Prompt: "prompt", Type: itemType, AnswerKey: answerKey}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("create practice item: %v", err)
	}

	for _, name := range conceptNames {
		concept := models.Concept{Name: name}
		if err := db.Create(&concept).Error; err != nil {
			t.Fatalf("create concept: %v", err)
		}
		if err := db.Exec(`INSERT INTO practice_item_concepts (practice_item_id, concept_id) VALUES (?, ?)`, item.ID, concept.ID).Error; err != nil {
			t.Fatalf("link item to concept: %v", err)
		}
	}

	return item
}

func TestGradeMCQExactMatchNoModelCall(t *testing.T) {
	db := testDB(t)

	answerKey := "paris"
	item := seedPracticeItem(t, db, models.PracticeItemTypeMCQ, &answerKey, "geography")

	calledModel := false
	service := newGradingService(t, db, func(w http.ResponseWriter, r *http.Request) {
		calledModel = true
	})

	attempt, err := service.Grade(context.Background(), item.ID, "  Paris  ")
	if err != nil {
		t.Fatalf("Grade: %v", err)
	}
	if calledModel {
		t.Error("expected no model call for exact-match grading")
	}
	if attempt.Passed == nil || !*attempt.Passed {
		t.Errorf("got passed %v, want true (case/whitespace-insensitive match)", attempt.Passed)
	}
}

func TestGradeMCQWrongAnswerExplainsExpectedChoice(t *testing.T) {
	db := testDB(t)

	answerKey := "B"
	item := seedPracticeItem(t, db, models.PracticeItemTypeMCQ, &answerKey)
	if err := db.Model(&item).Update("prompt", "Choose one.\nA) I have seen it last night.\nB) I saw it last night.").Error; err != nil {
		t.Fatalf("update prompt: %v", err)
	}

	service := newGradingService(t, db, func(w http.ResponseWriter, r *http.Request) {})

	attempt, err := service.Grade(context.Background(), item.ID, "a")
	if err != nil {
		t.Fatalf("Grade: %v", err)
	}
	if attempt.Passed == nil || *attempt.Passed {
		t.Fatalf("got passed %v, want false", attempt.Passed)
	}
	want := "Correct answer: B) I saw it last night."
	if attempt.AgentFeedback != want {
		t.Errorf("got feedback %q, want %q", attempt.AgentFeedback, want)
	}
}

func TestGradeOpenItemCallsModelAndLogsMistakeOnFail(t *testing.T) {
	db := testDB(t)

	item := seedPracticeItem(t, db, models.PracticeItemTypeOpen, nil, "recursion")

	service := newGradingService(t, db, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"id": "resp",
			"choices": [{
				"message": {"role": "assistant", "content": "{\"passed\":false,\"feedback\":\"missing base case\",\"flagged\":false,\"error_type\":\"knowledge_gap\"}"},
				"finish_reason": "stop"
			}]
		}`)
	})

	attempt, err := service.Grade(context.Background(), item.ID, "my answer")
	if err != nil {
		t.Fatalf("Grade: %v", err)
	}
	if attempt.Passed == nil || *attempt.Passed {
		t.Errorf("got passed %v, want false", attempt.Passed)
	}
	if attempt.AgentFeedback != "missing base case" {
		t.Errorf("got feedback %q, want %q", attempt.AgentFeedback, "missing base case")
	}

	var mistakes []models.MistakeLog
	if err := db.Where("attempt_id = ?", attempt.ID).Find(&mistakes).Error; err != nil {
		t.Fatalf("load mistake_log: %v", err)
	}
	if len(mistakes) != 1 || mistakes[0].ErrorType != models.ErrorTypeKnowledgeGap {
		t.Fatalf("got mistakes %+v, want 1 entry with error_type knowledge_gap", mistakes)
	}

	var weakness models.ConceptWeakness
	if err := db.First(&weakness, "concept_id = ?", mistakes[0].ConceptID).Error; err != nil {
		t.Fatalf("load concept_weakness: %v", err)
	}
	if weakness.MissCount != 1 || weakness.AttemptCount != 1 {
		t.Errorf("got weakness %+v, want MissCount=1 AttemptCount=1", weakness)
	}
	if weakness.LastMissedAt == nil {
		t.Error("got nil LastMissedAt, want it set")
	}
}

func TestGradePassingButFlaggedStillLogsMistake(t *testing.T) {
	db := testDB(t)

	item := seedPracticeItem(t, db, models.PracticeItemTypeCode, nil, "boundary-pointers")

	service := newGradingService(t, db, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"id": "resp",
			"choices": [{
				"message": {"role": "assistant", "content": "{\"passed\":true,\"feedback\":\"works but fragile on empty input\",\"flagged\":true,\"error_type\":\"careless_slip\"}"},
				"finish_reason": "stop"
			}]
		}`)
	})

	attempt, err := service.Grade(context.Background(), item.ID, "code submission")
	if err != nil {
		t.Fatalf("Grade: %v", err)
	}
	if attempt.Passed == nil || !*attempt.Passed {
		t.Errorf("got passed %v, want true", attempt.Passed)
	}

	var mistakeCount int64
	if err := db.Model(&models.MistakeLog{}).Where("attempt_id = ?", attempt.ID).Count(&mistakeCount).Error; err != nil {
		t.Fatalf("count mistake_log: %v", err)
	}
	if mistakeCount != 1 {
		t.Errorf("got %d mistake_log rows, want 1 — a pass can still be flagged", mistakeCount)
	}
}

func TestGradePassingNotFlaggedLogsNoMistake(t *testing.T) {
	db := testDB(t)

	item := seedPracticeItem(t, db, models.PracticeItemTypeOpen, nil, "concept")

	service := newGradingService(t, db, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"id": "resp",
			"choices": [{
				"message": {"role": "assistant", "content": "{\"passed\":true,\"feedback\":\"good\",\"flagged\":false,\"error_type\":\"\"}"},
				"finish_reason": "stop"
			}]
		}`)
	})

	attempt, err := service.Grade(context.Background(), item.ID, "answer")
	if err != nil {
		t.Fatalf("Grade: %v", err)
	}

	var mistakeCount int64
	if err := db.Model(&models.MistakeLog{}).Where("attempt_id = ?", attempt.ID).Count(&mistakeCount).Error; err != nil {
		t.Fatalf("count mistake_log: %v", err)
	}
	if mistakeCount != 0 {
		t.Errorf("got %d mistake_log rows, want 0 for an unflagged pass", mistakeCount)
	}
}

func topicStatus(t *testing.T, db *gorm.DB, topicID uint) string {
	t.Helper()
	var topic models.Topic
	if err := db.First(&topic, topicID).Error; err != nil {
		t.Fatalf("load topic: %v", err)
	}
	return topic.Status
}

func TestGradeMastersTopicOnceEveryItemPasses(t *testing.T) {
	db := testDB(t)

	topic := models.Topic{Title: "Capitals", Content: "content", Status: models.StatusNotStarted}
	if err := db.Create(&topic).Error; err != nil {
		t.Fatalf("create topic: %v", err)
	}
	keyA, keyB := "A", "C"
	first := models.PracticeItem{TopicID: topic.ID, Prompt: "q1", Type: models.PracticeItemTypeMCQ, AnswerKey: &keyA}
	second := models.PracticeItem{TopicID: topic.ID, Prompt: "q2", Type: models.PracticeItemTypeMCQ, AnswerKey: &keyB}
	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}
	if err := db.Create(&second).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}

	service := newGradingService(t, db, func(w http.ResponseWriter, r *http.Request) {
		t.Error("unexpected model call for mcq grading")
	})
	ctx := context.Background()

	if _, err := service.Grade(ctx, first.ID, "a"); err != nil {
		t.Fatalf("Grade: %v", err)
	}
	if got := topicStatus(t, db, topic.ID); got != models.StatusPracticing {
		t.Fatalf("after one of two passes: got status %q, want %q", got, models.StatusPracticing)
	}

	if _, err := service.Grade(ctx, second.ID, "B"); err != nil {
		t.Fatalf("Grade: %v", err)
	}
	if got := topicStatus(t, db, topic.ID); got != models.StatusPracticing {
		t.Fatalf("after a wrong answer: got status %q, want %q", got, models.StatusPracticing)
	}

	if _, err := service.Grade(ctx, second.ID, "c"); err != nil {
		t.Fatalf("Grade: %v", err)
	}
	if got := topicStatus(t, db, topic.ID); got != models.StatusMastered {
		t.Fatalf("after every item's latest attempt passed: got status %q, want %q", got, models.StatusMastered)
	}

	if _, err := service.Grade(ctx, first.ID, "wrong"); err != nil {
		t.Fatalf("Grade: %v", err)
	}
	if got := topicStatus(t, db, topic.ID); got != models.StatusMastered {
		t.Errorf("mastered should be terminal: got status %q after a later miss", got)
	}
}

func TestGradeFlaggedPassDoesNotMasterTopic(t *testing.T) {
	db := testDB(t)

	item := seedPracticeItem(t, db, models.PracticeItemTypeCode, nil, "boundary-pointers")

	service := newGradingService(t, db, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"id": "resp",
			"choices": [{
				"message": {"role": "assistant", "content": "{\"passed\":true,\"feedback\":\"fragile\",\"flagged\":true,\"error_type\":\"careless_slip\"}"},
				"finish_reason": "stop"
			}]
		}`)
	})

	if _, err := service.Grade(context.Background(), item.ID, "code"); err != nil {
		t.Fatalf("Grade: %v", err)
	}
	if got := topicStatus(t, db, item.TopicID); got != models.StatusPracticing {
		t.Errorf("got status %q, want %q — a flagged pass should not count toward mastery", got, models.StatusPracticing)
	}
}
