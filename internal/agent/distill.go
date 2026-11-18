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

//go:embed prompts/distill_system.txt
var distillSystemPromptFile string

var distillSystemPrompt = strings.TrimSpace(distillSystemPromptFile)

type DistillService struct {
	db     *gorm.DB
	client *ai.Client
}

func NewDistillService(db *gorm.DB, client *ai.Client) *DistillService {
	return &DistillService{db: db, client: client}
}

type distilledNote struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// Distill summarizes a chat's conversation into a standalone Note, linked
// back to the chat it came from via SourceChatID.
func (s *DistillService) Distill(ctx context.Context, chatID uint) (models.Node, error) {
	var messages []models.ChatMessage
	if err := s.db.Where("chat_id = ? AND role IN ?", chatID, []string{string(ai.RoleUser), string(ai.RoleAssistant)}).
		Order("created_at").Find(&messages).Error; err != nil {
		return models.Node{}, fmt.Errorf("agent: load chat for distill: %w", err)
	}
	if len(messages) == 0 {
		return models.Node{}, fmt.Errorf("agent: distill: chat %d has no messages to distill", chatID)
	}

	var transcript strings.Builder
	for _, m := range messages {
		if m.Content == "" {
			continue
		}
		fmt.Fprintf(&transcript, "%s: %s\n\n", m.Role, m.Content)
	}

	req := []ai.Message{
		{Role: ai.RoleSystem, Content: distillSystemPrompt},
		languageMessage(ctx, s.db),
		{Role: ai.RoleUser, Content: transcript.String()},
	}

	result, err := CallStructured[distilledNote](ctx, s.client, req)
	if err != nil {
		return models.Node{}, fmt.Errorf("agent: distill: %w", err)
	}
	if result.Title == "" || result.Content == "" {
		return models.Node{}, fmt.Errorf("agent: distill: model returned an empty title or content")
	}

	node := models.Node{
		Title:        result.Title,
		Content:      result.Content,
		SourceChatID: &chatID,
	}
	if err := s.db.Create(&node).Error; err != nil {
		return models.Node{}, fmt.Errorf("agent: distill: save note: %w", err)
	}
	return node, nil
}
