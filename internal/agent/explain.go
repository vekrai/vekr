package agent

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"vekr/internal/ai"
	"vekr/internal/i18n"
	"vekr/internal/models"
)

//go:embed prompts/explain_system.txt
var explainSystemPromptFile string

var explainSystemPrompt = strings.TrimSpace(explainSystemPromptFile)

type ExplainService struct {
	db     *gorm.DB
	client *ai.Client
}

func NewExplainService(db *gorm.DB, client *ai.Client) *ExplainService {
	return &ExplainService{db: db, client: client}
}

type explainReview struct {
	Feedback string `json:"feedback"`
}

func (s *ExplainService) Review(ctx context.Context, topicID uint, explanation string) (models.Node, error) {
	explanation = strings.TrimSpace(explanation)
	if explanation == "" {
		return models.Node{}, fmt.Errorf("agent: explain: explanation is empty")
	}

	var topic models.Topic
	if err := s.db.WithContext(ctx).First(&topic, topicID).Error; err != nil {
		return models.Node{}, fmt.Errorf("agent: explain: topic %d not found", topicID)
	}

	var notes []models.Node
	if err := s.db.WithContext(ctx).Where("topic_id = ?", topicID).Order("id").Find(&notes).Error; err != nil {
		return models.Node{}, fmt.Errorf("agent: explain: load notes: %w", err)
	}

	var material strings.Builder
	fmt.Fprintf(&material, "Topic: %s\n\n%s\n", topic.Title, topic.Content)
	for _, note := range notes {
		fmt.Fprintf(&material, "\n## %s\n\n%s\n", note.Title, note.Content)
	}
	fmt.Fprintf(&material, "\n---\n\nThe user's explanation:\n\n%s", explanation)

	req := []ai.Message{
		{Role: ai.RoleSystem, Content: explainSystemPrompt},
		languageMessage(ctx, s.db),
		{Role: ai.RoleUser, Content: material.String()},
	}

	review, err := CallStructured[explainReview](ctx, s.client, req)
	if err != nil {
		return models.Node{}, fmt.Errorf("agent: explain: %w", err)
	}
	if strings.TrimSpace(review.Feedback) == "" {
		return models.Node{}, fmt.Errorf("agent: explain: model returned empty feedback")
	}

	content := fmt.Sprintf("%s\n\n---\n\n### %s\n\n%s", explanation, i18n.T(ctx, "explain.note.feedback_heading"), review.Feedback)
	node := models.Node{
		Title:   i18n.TData(ctx, "explain.note.title", map[string]any{"Title": topic.Title}),
		Content: content,
		TopicID: &topicID,
	}
	if err := s.db.WithContext(ctx).Create(&node).Error; err != nil {
		return models.Node{}, fmt.Errorf("agent: explain: save note: %w", err)
	}
	return node, nil
}
