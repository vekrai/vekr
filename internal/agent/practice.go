package agent

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"vekr/internal/ai"
	"vekr/internal/models"
)

//go:embed prompts/practice_system.txt
var practiceSystemPromptFile string

var practiceSystemPrompt = strings.TrimSpace(practiceSystemPromptFile)

type PracticeService struct {
	db     *gorm.DB
	client *ai.Client
}

func NewPracticeService(db *gorm.DB, client *ai.Client) *PracticeService {
	return &PracticeService{db: db, client: client}
}

type generatedItems struct {
	Items []generatedItem `json:"items"`
}

type generatedItem struct {
	Prompt    string   `json:"prompt"`
	AnswerKey string   `json:"answer_key"`
	Type      string   `json:"type"`
	Concepts  []string `json:"concepts"`
}

// GenerateItems creates count practice items for a topic, picking a
// type (mcq/flashcard/open/code) per item contextually — one generic
// generator, no per-domain templates. Commits directly: practice items are
// cheap to regenerate or delete, so there's no preview step.
func (s *PracticeService) GenerateItems(ctx context.Context, topicID uint, count int) ([]models.PracticeItem, error) {
	var topic models.Topic
	if err := s.db.WithContext(ctx).First(&topic, topicID).Error; err != nil {
		return nil, fmt.Errorf("agent: generate practice items: topic %d not found", topicID)
	}

	var notes []models.Node
	if err := s.db.WithContext(ctx).Where("topic_id = ?", topicID).Order("id").Find(&notes).Error; err != nil {
		return nil, fmt.Errorf("agent: generate practice items: load notes: %w", err)
	}

	var material strings.Builder
	fmt.Fprintf(&material, "Topic: %s\n\n%s\n", topic.Title, topic.Content)
	for _, note := range notes {
		fmt.Fprintf(&material, "\n## %s\n\n%s\n", note.Title, note.Content)
	}
	fmt.Fprintf(&material, "\nGenerate %d practice items.", count)
	prompt := material.String()

	req := []ai.Message{
		{Role: ai.RoleSystem, Content: practiceSystemPrompt},
		{Role: ai.RoleUser, Content: prompt},
	}

	generated, err := CallStructured[generatedItems](ctx, s.client, req)
	if err != nil {
		return nil, fmt.Errorf("agent: generate practice items: %w", err)
	}
	if len(generated.Items) == 0 {
		return nil, fmt.Errorf("agent: generate practice items: model returned no items")
	}

	var items []models.PracticeItem
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, gi := range generated.Items {
			if gi.Prompt == "" {
				continue
			}
			switch gi.Type {
			case models.PracticeItemTypeMCQ, models.PracticeItemTypeFlashcard, models.PracticeItemTypeOpen, models.PracticeItemTypeCode:
			default:
				continue
			}
			item := models.PracticeItem{TopicID: topicID, Prompt: gi.Prompt, Type: gi.Type}
			if gi.AnswerKey != "" {
				item.AnswerKey = &gi.AnswerKey
			}
			if err := tx.Create(&item).Error; err != nil {
				return fmt.Errorf("save practice item: %w", err)
			}

			for _, name := range gi.Concepts {
				concept, err := findOrCreateConcept(tx, topicID, name)
				if err != nil {
					return err
				}
				if err := tx.Exec(`INSERT INTO practice_item_concepts (practice_item_id, concept_id) VALUES (?, ?) ON CONFLICT DO NOTHING`, item.ID, concept.ID).Error; err != nil {
					return fmt.Errorf("link practice item to concept %q: %w", name, err)
				}
			}

			items = append(items, item)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("agent: generate practice items: %w", err)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("agent: generate practice items: no valid items in model output")
	}

	return items, nil
}

func findOrCreateConcept(tx *gorm.DB, topicID uint, name string) (models.Concept, error) {
	var concept models.Concept
	query := tx.Where("name = ? AND topic_id = ?", name, topicID)

	err := query.First(&concept).Error
	if err == nil {
		return concept, nil
	}
	if err != gorm.ErrRecordNotFound {
		return models.Concept{}, fmt.Errorf("find concept %q: %w", name, err)
	}

	concept = models.Concept{TopicID: &topicID, Name: name}
	if err := tx.Create(&concept).Error; err != nil {
		return models.Concept{}, fmt.Errorf("create concept %q: %w", name, err)
	}
	return concept, nil
}
