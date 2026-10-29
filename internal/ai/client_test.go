package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChatCompletion(t *testing.T) {
	var gotBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"id": "resp-1",
			"model": "qwen-test",
			"choices": [{
				"message": {"role": "assistant", "content": "hi there"},
				"finish_reason": "stop"
			}],
			"usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}
		}`)
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:        server.URL,
		Model:          "qwen-test",
		MaxTokensField: "n_predict",
	}, nil)

	resp, err := client.ChatCompletion(context.Background(), ChatRequest{
		Messages:  []Message{{Role: RoleUser, Content: "hello"}},
		MaxTokens: 256,
	})
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}

	if resp.Message.Content != "hi there" {
		t.Errorf("got content %q, want %q", resp.Message.Content, "hi there")
	}
	if resp.FinishReason != "stop" {
		t.Errorf("got finish reason %q, want %q", resp.FinishReason, "stop")
	}
	if resp.Usage.TotalTokens != 15 {
		t.Errorf("got total tokens %d, want 15", resp.Usage.TotalTokens)
	}

	if gotBody["n_predict"] != float64(256) {
		t.Errorf("got n_predict %v, want 256 (quirk field name not applied)", gotBody["n_predict"])
	}
	if _, present := gotBody["max_tokens"]; present {
		t.Errorf("max_tokens should not be set when MaxTokensField quirk overrides it")
	}
	if gotBody["stream"] != false {
		t.Errorf("got stream %v, want false", gotBody["stream"])
	}
}

func TestChatCompletionErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "boom")
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, Model: "qwen-test"}, nil)

	_, err := client.ChatCompletion(context.Background(), ChatRequest{
		Messages: []Message{{Role: RoleUser, Content: "hello"}},
	})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestChatCompletionWithToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"id": "resp-2",
			"model": "qwen-test",
			"choices": [{
				"message": {
					"role": "assistant",
					"tool_calls": [{
						"id": "call-1",
						"type": "function",
						"function": {"name": "search_notes", "arguments": "{\"query\":\"go generics\"}"}
					}]
				},
				"finish_reason": "tool_calls"
			}],
			"usage": {"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2}
		}`)
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, Model: "qwen-test"}, nil)

	resp, err := client.ChatCompletion(context.Background(), ChatRequest{
		Messages: []Message{{Role: RoleUser, Content: "look this up"}},
	})
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}

	if resp.FinishReason != "tool_calls" {
		t.Errorf("got finish reason %q, want %q", resp.FinishReason, "tool_calls")
	}
	if len(resp.Message.ToolCalls) != 1 {
		t.Fatalf("got %d tool calls, want 1", len(resp.Message.ToolCalls))
	}

	call := resp.Message.ToolCalls[0]
	if call.Function.Name != "search_notes" {
		t.Errorf("got tool call name %q, want %q", call.Function.Name, "search_notes")
	}
	if call.Function.Arguments != `{"query":"go generics"}` {
		t.Errorf("got tool call arguments %q", call.Function.Arguments)
	}
}
