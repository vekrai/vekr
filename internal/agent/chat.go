package agent

import (
	"context"
	_ "embed"
	"encoding/json"
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
	tools  *ToolRegistry
}

func NewChatService(db *gorm.DB, client *ai.Client, tools *ToolRegistry) *ChatService {
	return &ChatService{db: db, client: client, tools: tools}
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

// buildMessages converts stored history into the wire format, prepending the
// system prompt and synthesizing a "no result" tool reply for any assistant
// tool call that never got a matching result (e.g. the process was cut short
// mid-loop) so the replayed history is never broken.
func buildMessages(history []models.ChatMessage) []ai.Message {
	messages := make([]ai.Message, 0, len(history)+1)
	messages = append(messages, ai.Message{Role: ai.RoleSystem, Content: chatSystemPrompt})

	var pendingToolCallIDs []string

	for _, m := range history {
		switch m.Role {
		case string(ai.RoleAssistant):
			msg := ai.Message{Role: ai.RoleAssistant, Content: m.Content}
			if m.ToolCalls != "" {
				var calls []ai.ToolCall
				if err := json.Unmarshal([]byte(m.ToolCalls), &calls); err == nil {
					msg.ToolCalls = calls
					for _, c := range calls {
						pendingToolCallIDs = append(pendingToolCallIDs, c.ID)
					}
				}
			}
			messages = append(messages, msg)
		case string(ai.RoleTool):
			pendingToolCallIDs = removeString(pendingToolCallIDs, m.ToolCallID)
			messages = append(messages, ai.Message{
				Role:       ai.RoleTool,
				Content:    m.Content,
				ToolCallID: m.ToolCallID,
				Name:       m.Name,
			})
		default:
			messages = append(messages, ai.Message{Role: ai.Role(m.Role), Content: m.Content})
		}
	}

	for _, id := range pendingToolCallIDs {
		messages = append(messages, ai.Message{Role: ai.RoleTool, ToolCallID: id, Content: "no result"})
	}

	return messages
}

func removeString(items []string, target string) []string {
	out := items[:0]
	for _, item := range items {
		if item != target {
			out = append(out, item)
		}
	}
	return out
}

func (s *ChatService) SendMessage(ctx context.Context, chatID uint, content string) (models.ChatMessage, error) {
	userMessage := models.ChatMessage{ChatID: chatID, Role: string(ai.RoleUser), Content: content}
	if err := s.db.Create(&userMessage).Error; err != nil {
		return models.ChatMessage{}, fmt.Errorf("agent: save user message: %w", err)
	}

	for step := 0; step < maxToolSteps; step++ {
		history, err := s.History(chatID)
		if err != nil {
			return models.ChatMessage{}, err
		}

		req := ai.ChatRequest{Messages: buildMessages(history)}
		if s.tools != nil {
			req.Tools = s.tools.Definitions()
		}

		resp, err := s.client.ChatCompletion(ctx, req)
		if err != nil {
			return models.ChatMessage{}, fmt.Errorf("agent: chat completion: %w", err)
		}

		if len(resp.Message.ToolCalls) == 0 {
			assistantMessage := models.ChatMessage{ChatID: chatID, Role: string(ai.RoleAssistant), Content: resp.Message.Content}
			if err := s.db.Create(&assistantMessage).Error; err != nil {
				return models.ChatMessage{}, fmt.Errorf("agent: save assistant message: %w", err)
			}
			return assistantMessage, nil
		}

		toolCallsJSON, err := json.Marshal(resp.Message.ToolCalls)
		if err != nil {
			return models.ChatMessage{}, fmt.Errorf("agent: encode tool calls: %w", err)
		}
		assistantMessage := models.ChatMessage{
			ChatID:    chatID,
			Role:      string(ai.RoleAssistant),
			Content:   resp.Message.Content,
			ToolCalls: string(toolCallsJSON),
		}
		if err := s.db.Create(&assistantMessage).Error; err != nil {
			return models.ChatMessage{}, fmt.Errorf("agent: save assistant tool-call message: %w", err)
		}

		for _, call := range resp.Message.ToolCalls {
			result, err := s.tools.Dispatch(ctx, call.Function.Name, json.RawMessage(call.Function.Arguments))
			if err != nil {
				result = "error: " + err.Error()
			}
			toolMessage := models.ChatMessage{
				ChatID:     chatID,
				Role:       string(ai.RoleTool),
				Content:    result,
				ToolCallID: call.ID,
				Name:       call.Function.Name,
			}
			if err := s.db.Create(&toolMessage).Error; err != nil {
				return models.ChatMessage{}, fmt.Errorf("agent: save tool result message: %w", err)
			}
		}
	}

	giveUpMessage := models.ChatMessage{
		ChatID:  chatID,
		Role:    string(ai.RoleAssistant),
		Content: "I wasn't able to finish that within the allowed number of tool calls.",
	}
	if err := s.db.Create(&giveUpMessage).Error; err != nil {
		return models.ChatMessage{}, fmt.Errorf("agent: save give-up message: %w", err)
	}
	return giveUpMessage, nil
}
