package agent

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vekr/internal/ai"
)

type structuredPayload struct {
	Foo string `json:"foo"`
}

func newStructuredClient(t *testing.T, handler http.HandlerFunc) *ai.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return ai.NewClient(ai.Config{BaseURL: server.URL, Model: "test"}, nil)
}

func TestCallStructuredSucceedsFirstTry(t *testing.T) {
	callCount := 0
	client := newStructuredClient(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"id": "resp-1",
			"choices": [{"message": {"role": "assistant", "content": "{\"foo\":\"bar\"}"}, "finish_reason": "stop"}]
		}`)
	})

	result, err := CallStructured[structuredPayload](context.Background(), client, []ai.Message{{Role: ai.RoleUser, Content: "go"}})
	if err != nil {
		t.Fatalf("CallStructured: %v", err)
	}
	if result.Foo != "bar" {
		t.Errorf("got %+v, want Foo=bar", result)
	}
	if callCount != 1 {
		t.Errorf("got %d calls, want 1", callCount)
	}
}

func TestCallStructuredRetriesOnMalformedJSON(t *testing.T) {
	callCount := 0
	var lastRequestBody string
	client := newStructuredClient(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		body, _ := io.ReadAll(r.Body)
		lastRequestBody = string(body)

		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			fmt.Fprint(w, `{
				"id": "resp-1",
				"choices": [{"message": {"role": "assistant", "content": "not json"}, "finish_reason": "stop"}]
			}`)
			return
		}
		fmt.Fprint(w, `{
			"id": "resp-2",
			"choices": [{"message": {"role": "assistant", "content": "{\"foo\":\"bar\"}"}, "finish_reason": "stop"}]
		}`)
	})

	result, err := CallStructured[structuredPayload](context.Background(), client, []ai.Message{{Role: ai.RoleUser, Content: "go"}})
	if err != nil {
		t.Fatalf("CallStructured: %v", err)
	}
	if result.Foo != "bar" {
		t.Errorf("got %+v, want Foo=bar", result)
	}
	if callCount != 2 {
		t.Errorf("got %d calls, want 2 (one failure, one correction)", callCount)
	}
	if !strings.Contains(lastRequestBody, "wasn't valid JSON") {
		t.Errorf("correction turn not found in second request body: %s", lastRequestBody)
	}
}

func TestCallStructuredGivesUpAfterMaxAttempts(t *testing.T) {
	callCount := 0
	client := newStructuredClient(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"id": "resp",
			"choices": [{"message": {"role": "assistant", "content": "still not json"}, "finish_reason": "stop"}]
		}`)
	})

	_, err := CallStructured[structuredPayload](context.Background(), client, []ai.Message{{Role: ai.RoleUser, Content: "go"}})
	if err == nil {
		t.Fatal("expected an error after exhausting attempts")
	}
	if callCount != maxStructuredAttempts {
		t.Errorf("got %d calls, want %d", callCount, maxStructuredAttempts)
	}
}

func TestCallStructuredFailsImmediatelyOnTruncation(t *testing.T) {
	callCount := 0
	client := newStructuredClient(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"id": "resp",
			"choices": [{"message": {"role": "assistant", "content": "{\"foo\":"}, "finish_reason": "length"}]
		}`)
	})

	_, err := CallStructured[structuredPayload](context.Background(), client, []ai.Message{{Role: ai.RoleUser, Content: "go"}})
	if err == nil {
		t.Fatal("expected an error for a truncated response")
	}
	if !strings.Contains(err.Error(), "truncated") {
		t.Errorf("got error %q, want it to mention truncation", err)
	}
	if callCount != 1 {
		t.Errorf("got %d calls, want 1 (no retry on truncation)", callCount)
	}
}
