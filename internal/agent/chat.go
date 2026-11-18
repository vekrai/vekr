package agent

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"vekr/internal/ai"
	"vekr/internal/live"
	"vekr/internal/models"
)

const backgroundReplyTimeout = 5 * time.Minute

//go:embed prompts/chat_system.txt
var chatSystemPromptFile string

var chatSystemPrompt = strings.TrimSpace(chatSystemPromptFile)

type ChatService struct {
	db     *gorm.DB
	client *ai.Client
	tools  *ToolRegistry
	hub    *live.Hub

	mu       sync.Mutex
	inFlight map[uint]bool
}

func NewChatService(db *gorm.DB, client *ai.Client, tools *ToolRegistry, hub *live.Hub) *ChatService {
	return &ChatService{db: db, client: client, tools: tools, hub: hub, inFlight: make(map[uint]bool)}
}

type chatIDKey struct{}

func withChatID(ctx context.Context, chatID uint) context.Context {
	return context.WithValue(ctx, chatIDKey{}, chatID)
}

func chatIDFromContext(ctx context.Context) (uint, bool) {
	chatID, ok := ctx.Value(chatIDKey{}).(uint)
	return chatID, ok
}

func (s *ChatService) Busy(chatID uint) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.inFlight[chatID]
}

func (s *ChatService) LatestMessageID(chatID uint) (uint, error) {
	var id uint
	err := s.db.Model(&models.ChatMessage{}).Where("chat_id = ?", chatID).Select("COALESCE(MAX(id), 0)").Scan(&id).Error
	if err != nil {
		return 0, fmt.Errorf("agent: load latest message id: %w", err)
	}
	return id, nil
}

func (s *ChatService) SendMessageInBackground(chatID uint, content string) (bool, error) {
	s.mu.Lock()
	if s.inFlight[chatID] {
		s.mu.Unlock()
		return false, nil
	}
	s.inFlight[chatID] = true
	s.mu.Unlock()

	if _, err := s.AddUserMessage(chatID, content); err != nil {
		s.finishReply(chatID)
		return false, err
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), backgroundReplyTimeout)
		defer cancel()

		if _, err := s.Reply(ctx, chatID); err != nil {
			log.Printf("chat %d: background reply failed: %v", chatID, err)
			failure := models.ChatMessage{ChatID: chatID, Role: string(ai.RoleAssistant), Content: "Sorry, something went wrong while answering: " + err.Error()}
			if err := s.db.Create(&failure).Error; err != nil {
				log.Printf("chat %d: save failure message: %v", chatID, err)
			}
		}
		s.finishReply(chatID)
	}()
	return true, nil
}

func (s *ChatService) finishReply(chatID uint) {
	s.mu.Lock()
	delete(s.inFlight, chatID)
	s.mu.Unlock()
	if s.hub != nil {
		s.hub.Publish(live.ChatStream(chatID), live.RefreshAction())
	}
}

func (s *ChatService) CurrentChat() (models.Chat, error) {
	var chat models.Chat
	err := s.db.Where("node_id IS NULL").Order("created_at desc").First(&chat).Error
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

func (s *ChatService) StartChat() (models.Chat, error) {
	current, err := s.CurrentChat()
	if err != nil {
		return models.Chat{}, err
	}
	var count int64
	if err := s.db.Model(&models.ChatMessage{}).Where("chat_id = ?", current.ID).Count(&count).Error; err != nil {
		return models.Chat{}, fmt.Errorf("agent: count chat messages: %w", err)
	}
	if count == 0 {
		return current, nil
	}

	chat := models.Chat{}
	if err := s.db.Create(&chat).Error; err != nil {
		return models.Chat{}, fmt.Errorf("agent: create chat: %w", err)
	}
	return chat, nil
}

func (s *ChatService) RecentChats(limit int) ([]models.ChatSummary, error) {
	var chats []models.ChatSummary
	err := s.db.Raw(`
		SELECT c.id AS chat_id,
		       (SELECT m.content FROM chat_messages m WHERE m.chat_id = c.id AND m.role = 'user' ORDER BY m.id LIMIT 1) AS title,
		       MAX(cm.created_at) AS last_at
		FROM chats c
		JOIN chat_messages cm ON cm.chat_id = c.id
		WHERE c.node_id IS NULL
		GROUP BY c.id
		ORDER BY last_at DESC
		LIMIT ?
	`, limit).Scan(&chats).Error
	if err != nil {
		return nil, fmt.Errorf("agent: load recent chats: %w", err)
	}
	return chats, nil
}

func (s *ChatService) FindChat(chatID uint) (models.Chat, error) {
	var chat models.Chat
	if err := s.db.First(&chat, chatID).Error; err != nil {
		return models.Chat{}, fmt.Errorf("agent: load chat %d: %w", chatID, err)
	}
	return chat, nil
}

func (s *ChatService) StartNodeChat(nodeID uint) (models.Chat, error) {
	chat := models.Chat{NodeID: &nodeID}
	if err := s.db.Create(&chat).Error; err != nil {
		return models.Chat{}, fmt.Errorf("agent: create chat for node %d: %w", nodeID, err)
	}
	return chat, nil
}

func (s *ChatService) pageContext(ctx context.Context, chat models.Chat) (string, error) {
	if chat.NodeID == nil {
		return "", nil
	}
	var node models.Node
	if err := s.db.WithContext(ctx).First(&node, *chat.NodeID).Error; err != nil {
		return "", fmt.Errorf("agent: load chat node %d: %w", *chat.NodeID, err)
	}
	return fmt.Sprintf("The user is chatting from the page of node_id %d, a %s titled %q. \"This note\" or \"this topic\" means this node — call read_note with node_id %d if you need its content.", node.ID, node.Type, node.Title, node.ID), nil
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
func buildMessages(history []models.ChatMessage, pageContext string) []ai.Message {
	messages := make([]ai.Message, 0, len(history)+2)
	messages = append(messages, ai.Message{Role: ai.RoleSystem, Content: chatSystemPrompt})
	if pageContext != "" {
		messages = append(messages, ai.Message{Role: ai.RoleSystem, Content: pageContext})
	}

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
	if _, err := s.AddUserMessage(chatID, content); err != nil {
		return models.ChatMessage{}, err
	}
	return s.Reply(ctx, chatID)
}

func (s *ChatService) AddUserMessage(chatID uint, content string) (models.ChatMessage, error) {
	userMessage := models.ChatMessage{ChatID: chatID, Role: string(ai.RoleUser), Content: content}
	if err := s.db.Create(&userMessage).Error; err != nil {
		return models.ChatMessage{}, fmt.Errorf("agent: save user message: %w", err)
	}
	return userMessage, nil
}

func (s *ChatService) Reply(ctx context.Context, chatID uint) (models.ChatMessage, error) {
	ctx = withChatID(ctx, chatID)
	chat, err := s.FindChat(chatID)
	if err != nil {
		return models.ChatMessage{}, err
	}
	pageContext, err := s.pageContext(ctx, chat)
	if err != nil {
		return models.ChatMessage{}, err
	}

	for step := 0; step < maxToolSteps; step++ {
		history, err := s.History(chatID)
		if err != nil {
			return models.ChatMessage{}, err
		}

		req := ai.ChatRequest{Messages: buildMessages(history, pageContext)}
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
