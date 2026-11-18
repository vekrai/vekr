package agent

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"vekr/internal/ai"
	"vekr/internal/models"
)

//go:embed prompts/grading_system.txt
var gradingSystemPromptFile string

var gradingSystemPrompt = strings.TrimSpace(gradingSystemPromptFile)

type GradingService struct {
	db     *gorm.DB
	client *ai.Client
}

func NewGradingService(db *gorm.DB, client *ai.Client) *GradingService {
	return &GradingService{db: db, client: client}
}

type gradeResult struct {
	Passed    bool   `json:"passed"`
	Feedback  string `json:"feedback"`
	Flagged   bool   `json:"flagged"`
	ErrorType string `json:"error_type"`
}

// Grade records an Attempt for a practice item. mcq and flashcard items
// are graded by exact match, no model call needed; open and code items are
// judged by the LLM (no execution sandbox — judgment only). A passing
// attempt can still come back Flagged if the review spots a weak pattern
// (e.g. fragile boundary logic that happens to look correct), which still
// produces a MistakeLog entry and rolls into ConceptWeakness. Single-row
// output, commits directly — no preview step.
func (s *GradingService) Grade(ctx context.Context, practiceItemID uint, submission string) (models.Attempt, error) {
	var item models.PracticeItem
	if err := s.db.WithContext(ctx).First(&item, practiceItemID).Error; err != nil {
		return models.Attempt{}, fmt.Errorf("agent: grade: practice item %d not found", practiceItemID)
	}

	var result gradeResult
	switch item.Type {
	case models.PracticeItemTypeMCQ, models.PracticeItemTypeFlashcard:
		expected := ""
		if item.AnswerKey != nil {
			expected = *item.AnswerKey
		}
		result = gradeResult{Passed: strings.EqualFold(strings.TrimSpace(submission), strings.TrimSpace(expected))}
	default:
		var err error
		result, err = s.judge(ctx, item, submission)
		if err != nil {
			return models.Attempt{}, fmt.Errorf("agent: grade: %w", err)
		}
	}

	passed := result.Passed
	attempt := models.Attempt{
		PracticeItemID: practiceItemID,
		Submission:     submission,
		Passed:         &passed,
		AgentFeedback:  result.Feedback,
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&attempt).Error; err != nil {
			return fmt.Errorf("save attempt: %w", err)
		}

		if !result.Passed || result.Flagged {
			errorType := result.ErrorType
			if errorType == "" {
				errorType = models.ErrorTypeKnowledgeGap
			}

			var conceptIDs []uint
			if err := tx.Table("practice_item_concepts").Where("practice_item_id = ?", practiceItemID).Pluck("concept_id", &conceptIDs).Error; err != nil {
				return fmt.Errorf("load concepts for item: %w", err)
			}

			for _, conceptID := range conceptIDs {
				mistake := models.MistakeLog{AttemptID: attempt.ID, ConceptID: conceptID, ErrorType: errorType}
				if err := tx.Create(&mistake).Error; err != nil {
					return fmt.Errorf("save mistake log: %w", err)
				}
				if err := recomputeConceptWeakness(tx, conceptID); err != nil {
					return err
				}
			}
		}

		return updateTopicStatus(tx, item.TopicID)
	})
	if err != nil {
		return models.Attempt{}, fmt.Errorf("agent: grade: %w", err)
	}

	return attempt, nil
}

func (s *GradingService) judge(ctx context.Context, item models.PracticeItem, submission string) (gradeResult, error) {
	answerKey := ""
	if item.AnswerKey != nil {
		answerKey = *item.AnswerKey
	}

	prompt := fmt.Sprintf("Prompt:\n%s\n\nReference answer (may be blank):\n%s\n\nSubmission:\n%s", item.Prompt, answerKey, submission)
	req := []ai.Message{
		{Role: ai.RoleSystem, Content: gradingSystemPrompt},
		languageMessage(ctx, s.db),
		{Role: ai.RoleUser, Content: prompt},
	}

	return CallStructured[gradeResult](ctx, s.client, req)
}

func updateTopicStatus(tx *gorm.DB, topicID uint) error {
	var topic models.Topic
	if err := tx.First(&topic, topicID).Error; err != nil {
		return fmt.Errorf("load topic %d: %w", topicID, err)
	}
	if topic.Status == models.StatusMastered {
		return nil
	}

	var row struct {
		Total  int64
		Passed int64
	}
	err := tx.Raw(`
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (
				WHERE latest.passed
				  AND NOT EXISTS (SELECT 1 FROM mistake_log m WHERE m.attempt_id = latest.id)
			) AS passed
		FROM practice_items pi
		LEFT JOIN LATERAL (
			SELECT a.id, a.passed
			  FROM attempts a
			 WHERE a.practice_item_id = pi.id
			 ORDER BY a.created_at DESC, a.id DESC
			 LIMIT 1
		) latest ON true
		WHERE pi.topic_id = ?
	`, topicID).Scan(&row).Error
	if err != nil {
		return fmt.Errorf("load topic %d progress: %w", topicID, err)
	}

	status := models.StatusPracticing
	if row.Total > 0 && row.Passed == row.Total {
		status = models.StatusMastered
	}
	if err := tx.Model(&models.Topic{}).Where("id = ?", topicID).Update("status", status).Error; err != nil {
		return fmt.Errorf("update topic %d status: %w", topicID, err)
	}
	return nil
}

// recomputeConceptWeakness recomputes the rollup for one concept from
// scratch — aggregate queries over mistake_log/attempts, rather than
// incrementing counters in place — so it can never drift from the
// underlying rows.
func recomputeConceptWeakness(tx *gorm.DB, conceptID uint) error {
	var row struct {
		MissCount    int64
		AttemptCount int64
		LastMissedAt *time.Time
	}

	err := tx.Raw(`
		SELECT
			(SELECT COUNT(*) FROM mistake_log WHERE concept_id = ?) AS miss_count,
			(SELECT COUNT(DISTINCT a.id)
			   FROM attempts a
			   JOIN practice_item_concepts pic ON pic.practice_item_id = a.practice_item_id
			  WHERE pic.concept_id = ?) AS attempt_count,
			(SELECT MAX(created_at) FROM mistake_log WHERE concept_id = ?) AS last_missed_at
	`, conceptID, conceptID, conceptID).Scan(&row).Error
	if err != nil {
		return fmt.Errorf("recompute concept_weakness: %w", err)
	}

	err = tx.Exec(`
		INSERT INTO concept_weakness (concept_id, miss_count, attempt_count, last_missed_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (concept_id) DO UPDATE SET
			miss_count = EXCLUDED.miss_count,
			attempt_count = EXCLUDED.attempt_count,
			last_missed_at = EXCLUDED.last_missed_at
	`, conceptID, row.MissCount, row.AttemptCount, row.LastMissedAt).Error
	if err != nil {
		return fmt.Errorf("upsert concept_weakness: %w", err)
	}
	return nil
}
