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

//go:embed prompts/chat_system.txt
var chatSystemPromptFile string

var chatSystemPrompt = strings.TrimSpace(chatSystemPromptFile)

type ChatService struct {
	db     *gorm.DB
	client *ai.Client
}

func NewChatService(db *gorm.DB, client *ai.Client) *ChatService {
	return &ChatService{db: db, client: client}
}

func (s *ChatService) CurrentChat() (models.Chat, error) {
	var chat models.Chat
	err := s.db.Order("created_at desc").First(&chat).Error
	if err == nil {
		return chat, nil
	}
	if err != gorm.ErrRecordNotFound {
		return models.Chat{}, fmt.Errorf("agent: load current chat: %w", err)
	}

	chat = models.Chat{}
	if err := s.db.Create(&chat).Error; err != nil {
		return models.Chat{}, fmt.Errorf("agent: create chat: %w", err)
	}
	return chat, nil
}

func (s *ChatService) History(chatID uint) ([]models.ChatMessage, error) {
	var messages []models.ChatMessage
	if err := s.db.Where("chat_id = ?", chatID).Order("created_at").Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("agent: load chat history: %w", err)
	}
	return messages, nil
}

func (s *ChatService) SendMessage(ctx context.Context, chatID uint, content string) (models.ChatMessage, error) {
	userMessage := models.ChatMessage{ChatID: chatID, Role: string(ai.RoleUser), Content: content}
	if err := s.db.Create(&userMessage).Error; err != nil {
		return models.ChatMessage{}, fmt.Errorf("agent: save user message: %w", err)
	}

	history, err := s.History(chatID)
	if err != nil {
		return models.ChatMessage{}, err
	}

	messages := make([]ai.Message, 0, len(history)+1)
	messages = append(messages, ai.Message{Role: ai.RoleSystem, Content: chatSystemPrompt})
	for _, m := range history {
		messages = append(messages, ai.Message{Role: ai.Role(m.Role), Content: m.Content})
	}

	resp, err := s.client.ChatCompletion(ctx, ai.ChatRequest{Messages: messages})
	if err != nil {
		return models.ChatMessage{}, fmt.Errorf("agent: chat completion: %w", err)
	}

	assistantMessage := models.ChatMessage{ChatID: chatID, Role: string(ai.RoleAssistant), Content: resp.Message.Content}
	if err := s.db.Create(&assistantMessage).Error; err != nil {
		return models.ChatMessage{}, fmt.Errorf("agent: save assistant message: %w", err)
	}

	return assistantMessage, nil
}
