package ai

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func writeSSE(w http.ResponseWriter, data string) {
	fmt.Fprintf(w, "data: %s\n\n", data)
}

func TestStreamChatCompletionContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		writeSSE(w, `{"choices":[{"delta":{"role":"assistant"},"finish_reason":null}]}`)
		writeSSE(w, `{"choices":[{"delta":{"content":"hel"},"finish_reason":null}]}`)
		writeSSE(w, `{"choices":[{"delta":{"content":"lo"},"finish_reason":null}]}`)
		writeSSE(w, `{"choices":[{"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":3,"completion_tokens":2,"total_tokens":5}}`)
		writeSSE(w, "[DONE]")
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, Model: "qwen-test"}, nil)

	var deltas []string
	resp, err := client.StreamChatCompletion(context.Background(), ChatRequest{
		Messages: []Message{{Role: RoleUser, Content: "hi"}},
	}, func(ev StreamEvent) {
		deltas = append(deltas, ev.ContentDelta)
	})
	if err != nil {
		t.Fatalf("StreamChatCompletion: %v", err)
	}

	if resp.Message.Content != "hello" {
		t.Errorf("got content %q, want %q", resp.Message.Content, "hello")
	}
	if resp.FinishReason != "stop" {
		t.Errorf("got finish reason %q, want %q", resp.FinishReason, "stop")
	}
	if resp.Usage.TotalTokens != 5 {
		t.Errorf("got total tokens %d, want 5", resp.Usage.TotalTokens)
	}
	if len(deltas) != 2 || deltas[0] != "hel" || deltas[1] != "lo" {
		t.Errorf("got content deltas %v, want [hel lo]", deltas)
	}
}

func TestStreamChatCompletionToolCallFragments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		writeSSE(w, `{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call-1","type":"function","function":{"name":"search_notes","arguments":""}}]},"finish_reason":null}]}`)
		writeSSE(w, `{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"query\":"}}]},"finish_reason":null}]}`)
		writeSSE(w, `{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"go generics\"}"}}]},"finish_reason":null}]}`)
		writeSSE(w, `{"choices":[{"delta":{},"finish_reason":"tool_calls"}]}`)
		writeSSE(w, "[DONE]")
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, Model: "qwen-test"}, nil)

	resp, err := client.StreamChatCompletion(context.Background(), ChatRequest{
		Messages: []Message{{Role: RoleUser, Content: "look this up"}},
	}, nil)
	if err != nil {
		t.Fatalf("StreamChatCompletion: %v", err)
	}

	if resp.FinishReason != "tool_calls" {
		t.Errorf("got finish reason %q, want %q", resp.FinishReason, "tool_calls")
	}
	if len(resp.Message.ToolCalls) != 1 {
		t.Fatalf("got %d tool calls, want 1", len(resp.Message.ToolCalls))
	}

	call := resp.Message.ToolCalls[0]
	if call.ID != "call-1" {
		t.Errorf("got tool call id %q, want %q", call.ID, "call-1")
	}
	if call.Function.Name != "search_notes" {
		t.Errorf("got tool call name %q, want %q", call.Function.Name, "search_notes")
	}
	if call.Function.Arguments != `{"query":"go generics"}` {
		t.Errorf("got assembled arguments %q", call.Function.Arguments)
	}
}

func TestStreamChatCompletionErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(w, "down")
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, Model: "qwen-test"}, nil)

	_, err := client.StreamChatCompletion(context.Background(), ChatRequest{
		Messages: []Message{{Role: RoleUser, Content: "hi"}},
	}, nil)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}
