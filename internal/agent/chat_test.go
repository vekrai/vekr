package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"

	"vekr/internal/ai"
	"vekr/internal/live"
	"vekr/internal/models"
)

func newChatService(t *testing.T, db *gorm.DB, handler http.HandlerFunc) *ChatService {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client := ai.NewClient(ai.Config{BaseURL: server.URL, Model: "test"}, nil)
	knowledge := NewKnowledgeTools(db)
	curriculum := NewCurriculumTools(NewCurriculumService(db, client))
	practice := NewPracticeTools(NewPracticeService(db, client))
	tools := NewToolRegistry(NewWebSearchTool("", nil), NewWebFetchTool(), knowledge, curriculum, practice)
	return NewChatService(db, client, tools, nil)
}

func TestChatServiceDispatchesToolCallThenAnswers(t *testing.T) {
	db := testDB(t)

	node := models.Node{Title: "Go generics", Content: "Generics let you write functions over any type."}
	if err := db.Create(&node).Error; err != nil {
		t.Fatalf("create node: %v", err)
	}

	callCount := 0
	service := newChatService(t, db, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			fmt.Fprint(w, `{
				"id": "resp-1",
				"choices": [{
					"message": {
						"role": "assistant",
						"tool_calls": [{
							"id": "call-1",
							"type": "function",
							"function": {"name": "search_notes", "arguments": "{\"query\":\"generics\"}"}
						}]
					},
					"finish_reason": "tool_calls"
				}]
			}`)
			return
		}
		fmt.Fprint(w, `{
			"id": "resp-2",
			"choices": [{"message": {"role": "assistant", "content": "Generics explained."}, "finish_reason": "stop"}]
		}`)
	})

	chat, err := service.CurrentChat()
	if err != nil {
		t.Fatalf("CurrentChat: %v", err)
	}

	final, err := service.SendMessage(context.Background(), chat.ID, "tell me about generics")
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if final.Content != "Generics explained." {
		t.Errorf("got final content %q, want %q", final.Content, "Generics explained.")
	}

	history, err := service.History(chat.ID)
	if err != nil {
		t.Fatalf("History: %v", err)
	}

	var toolMessage *models.ChatMessage
	for i := range history {
		if history[i].Role == "tool" {
			toolMessage = &history[i]
		}
	}
	if toolMessage == nil {
		t.Fatal("expected a tool-role message recording the search_notes result")
	}
	if toolMessage.ToolCallID != "call-1" || toolMessage.Name != "search_notes" {
		t.Errorf("got tool message %+v, want ToolCallID=call-1 Name=search_notes", toolMessage)
	}
	if !strings.Contains(toolMessage.Content, "Go generics") {
		t.Errorf("got tool result content %q, want it to mention the note title", toolMessage.Content)
	}

	if callCount != 2 {
		t.Errorf("got %d model calls, want 2 (one tool round, one final answer)", callCount)
	}
}

func TestChatServiceInvalidToolArgsFeedsBackError(t *testing.T) {
	db := testDB(t)

	callCount := 0
	service := newChatService(t, db, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			fmt.Fprint(w, `{
				"id": "resp-1",
				"choices": [{
					"message": {
						"role": "assistant",
						"tool_calls": [{
							"id": "call-1",
							"type": "function",
							"function": {"name": "search_notes", "arguments": "{}"}
						}]
					},
					"finish_reason": "tool_calls"
				}]
			}`)
			return
		}
		fmt.Fprint(w, `{
			"id": "resp-2",
			"choices": [{"message": {"role": "assistant", "content": "ok, retried"}, "finish_reason": "stop"}]
		}`)
	})

	chat, err := service.CurrentChat()
	if err != nil {
		t.Fatalf("CurrentChat: %v", err)
	}

	if _, err := service.SendMessage(context.Background(), chat.ID, "search for something"); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	history, err := service.History(chat.ID)
	if err != nil {
		t.Fatalf("History: %v", err)
	}

	var toolMessage *models.ChatMessage
	for i := range history {
		if history[i].Role == "tool" {
			toolMessage = &history[i]
		}
	}
	if toolMessage == nil || !strings.HasPrefix(toolMessage.Content, "error:") {
		t.Fatalf("got tool message %+v, want a result starting with %q", toolMessage, "error:")
	}
}

func TestChatServiceGivesUpAfterMaxToolSteps(t *testing.T) {
	db := testDB(t)

	callCount := 0
	service := newChatService(t, db, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
			"id": "resp",
			"choices": [{
				"message": {
					"role": "assistant",
					"tool_calls": [{
						"id": "call-%d",
						"type": "function",
						"function": {"name": "search_notes", "arguments": "{\"query\":\"x\"}"}
					}]
				},
				"finish_reason": "tool_calls"
			}]
		}`, callCount)
	})

	chat, err := service.CurrentChat()
	if err != nil {
		t.Fatalf("CurrentChat: %v", err)
	}

	final, err := service.SendMessage(context.Background(), chat.ID, "loop forever")
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if !strings.Contains(final.Content, "allowed number of tool calls") {
		t.Errorf("got %q, want a give-up message", final.Content)
	}
}

func TestBuildMessagesSynthesizesDanglingToolCall(t *testing.T) {
	toolCallsJSON, _ := json.Marshal([]ai.ToolCall{{ID: "call-dangling", Type: "function", Function: ai.ToolCallFunction{Name: "search_notes", Arguments: "{}"}}})

	history := []models.ChatMessage{
		{Role: "user", Content: "hi"},
		{Role: "assistant", ToolCalls: string(toolCallsJSON)},
	}

	messages := buildMessages(history, "")

	var sawSynthesized bool
	for _, m := range messages {
		if m.Role == ai.RoleTool && m.ToolCallID == "call-dangling" {
			sawSynthesized = true
			if m.Content != "no result" {
				t.Errorf("got synthesized content %q, want %q", m.Content, "no result")
			}
		}
	}
	if !sawSynthesized {
		t.Fatal("expected a synthesized tool-result message for the dangling tool call")
	}
}

func TestChatServiceNodeChatSendsPageContext(t *testing.T) {
	db := testDB(t)

	node := models.Node{Title: "Binary search", Content: "Halve the range each step.", Type: models.NodeTypeTopic}
	if err := db.Create(&node).Error; err != nil {
		t.Fatalf("create node: %v", err)
	}

	var body string
	service := newChatService(t, db, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body = string(raw)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "resp-1", "choices": [{"message": {"role": "assistant", "content": "ok"}, "finish_reason": "stop"}]}`)
	})

	chat, err := service.StartNodeChat(node.ID)
	if err != nil {
		t.Fatalf("StartNodeChat: %v", err)
	}

	if _, err := service.SendMessage(context.Background(), chat.ID, "quiz me on this"); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	want := fmt.Sprintf("node_id %d, a topic titled \\\"Binary search\\\"", node.ID)
	if !strings.Contains(body, want) {
		t.Errorf("request body missing page context %q:\n%s", want, body)
	}

	current, err := service.CurrentChat()
	if err != nil {
		t.Fatalf("CurrentChat: %v", err)
	}
	if current.ID == chat.ID {
		t.Error("CurrentChat returned the node-bound chat, want it to skip chats with a node_id")
	}
}

func TestSendMessageInBackgroundRepliesThenPublishesRefresh(t *testing.T) {
	db := testDB(t)

	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "resp-1", "choices": [{"message": {"role": "assistant", "content": "done thinking"}, "finish_reason": "stop"}]}`)
	}))
	t.Cleanup(server.Close)

	client := ai.NewClient(ai.Config{BaseURL: server.URL, Model: "test"}, nil)
	hub := live.NewHub()
	service := NewChatService(db, client, nil, hub)

	chat, err := service.CurrentChat()
	if err != nil {
		t.Fatalf("CurrentChat: %v", err)
	}
	events, unsubscribe := hub.Subscribe(live.ChatStream(chat.ID))
	defer unsubscribe()

	started, err := service.SendMessageInBackground(chat.ID, "hello")
	if err != nil || !started {
		t.Fatalf("SendMessageInBackground: started=%v err=%v", started, err)
	}
	if !service.Busy(chat.ID) {
		t.Error("chat should be busy while the reply is in flight")
	}

	again, err := service.SendMessageInBackground(chat.ID, "second message")
	if err != nil || again {
		t.Errorf("second send while busy: started=%v err=%v, want false and no error", again, err)
	}

	close(release)

	select {
	case event := <-events:
		if event != live.RefreshAction() {
			t.Errorf("got event %q, want a refresh action", event)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no refresh published after the background reply finished")
	}

	if service.Busy(chat.ID) {
		t.Error("chat still busy after the reply finished")
	}

	history, err := service.History(chat.ID)
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(history) != 2 || history[0].Content != "hello" || history[1].Content != "done thinking" {
		t.Errorf("got history %+v, want the user message then the reply (and no second message)", history)
	}
}
