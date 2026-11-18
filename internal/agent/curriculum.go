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

//go:embed prompts/curriculum_system.txt
var curriculumSystemPromptFile string

var curriculumSystemPrompt = strings.TrimSpace(curriculumSystemPromptFile)

type CurriculumService struct {
	db     *gorm.DB
	client *ai.Client
}

func NewCurriculumService(db *gorm.DB, client *ai.Client) *CurriculumService {
	return &CurriculumService{db: db, client: client}
}

type CurriculumPreview struct {
	TopicTitle   string        `json:"topic_title"`
	TopicContent string        `json:"topic_content"`
	Notes        []PreviewNote `json:"notes"`
}

type PreviewNote struct {
	Title         string   `json:"title"`
	Content       string   `json:"content"`
	Prerequisites []string `json:"prerequisites"`
}

func (s *CurriculumService) Preview(ctx context.Context, intake string) (CurriculumPreview, error) {
	req := []ai.Message{
		{Role: ai.RoleSystem, Content: curriculumSystemPrompt},
		languageMessage(ctx, s.db),
		{Role: ai.RoleUser, Content: intake},
	}

	preview, err := CallStructured[CurriculumPreview](ctx, s.client, req)
	if err != nil {
		return CurriculumPreview{}, fmt.Errorf("agent: curriculum preview: %w", err)
	}
	if preview.TopicTitle == "" || len(preview.Notes) == 0 {
		return CurriculumPreview{}, fmt.Errorf("agent: curriculum preview: model returned no topic title or no notes")
	}
	return preview, nil
}

func (s *CurriculumService) Commit(ctx context.Context, preview CurriculumPreview) (models.Topic, error) {
	var topic models.Topic

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		topic = models.Topic{Title: preview.TopicTitle, Content: preview.TopicContent, Status: models.StatusNotStarted}
		if err := tx.Create(&topic).Error; err != nil {
			return fmt.Errorf("save topic: %w", err)
		}

		notesByTitle := make(map[string]models.Node, len(preview.Notes))
		for _, note := range preview.Notes {
			topicID := topic.ID
			node := models.Node{Title: note.Title, Content: note.Content, TopicID: &topicID}
			if err := tx.Create(&node).Error; err != nil {
				return fmt.Errorf("save note %q: %w", note.Title, err)
			}
			notesByTitle[note.Title] = node
		}

		for _, note := range preview.Notes {
			node := notesByTitle[note.Title]
			for _, prereqTitle := range note.Prerequisites {
				prereq, ok := notesByTitle[prereqTitle]
				if !ok {
					continue
				}
				link := models.NodeLink{FromNodeID: prereq.ID, ToNodeID: node.ID, Relation: models.RelationPrerequisiteOf}
				if err := tx.Create(&link).Error; err != nil {
					return fmt.Errorf("save prerequisite link %q -> %q: %w", prereqTitle, note.Title, err)
				}
			}
		}

		return nil
	})
	if err != nil {
		return models.Topic{}, fmt.Errorf("agent: curriculum commit: %w", err)
	}

	return topic, nil
}

func (s *CurriculumService) PromoteToTopic(ctx context.Context, nodeID uint) (models.Topic, error) {
	var topic models.Topic

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var node models.Node
		if err := tx.First(&node, nodeID).Error; err != nil {
			return fmt.Errorf("note %d not found", nodeID)
		}
		if node.TopicID != nil {
			return tx.First(&topic, *node.TopicID).Error
		}

		topic = models.Topic{Title: node.Title, Status: models.StatusNotStarted}
		if err := tx.Create(&topic).Error; err != nil {
			return fmt.Errorf("save topic: %w", err)
		}
		return tx.Model(&models.Node{}).Where("id = ?", nodeID).Update("topic_id", topic.ID).Error
	})
	if err != nil {
		return models.Topic{}, fmt.Errorf("agent: promote to topic: %w", err)
	}
	return topic, nil
}
