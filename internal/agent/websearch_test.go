package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebSearchKeylessSetsAccessModeHeader(t *testing.T) {
	var gotBody map[string]any
	var gotHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Tavily-Access-Mode")
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"results":[{"title":"Go generics","url":"https://go.dev/x","content":"a summary"}]}`)
	}))
	defer server.Close()

	tool := NewWebSearchTool("", nil)
	tool.searchURL = server.URL

	result, err := tool.Search(context.Background(), "go generics")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if gotHeader != "keyless" {
		t.Errorf("got X-Tavily-Access-Mode %q, want %q", gotHeader, "keyless")
	}
	if _, present := gotBody["api_key"]; present {
		t.Errorf("api_key should not be set when no key is configured")
	}
	if !strings.Contains(result, "Go generics") || !strings.Contains(result, "a summary") {
		t.Errorf("got %q, want it to contain the result title and content", result)
	}
}

func TestWebSearchWithAPIKey(t *testing.T) {
	var gotBody map[string]any
	var gotHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Tavily-Access-Mode")
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"results":[]}`)
	}))
	defer server.Close()

	tool := NewWebSearchTool("secret-key", nil)
	tool.searchURL = server.URL

	result, err := tool.Search(context.Background(), "anything")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if gotBody["api_key"] != "secret-key" {
		t.Errorf("got api_key %v, want %q", gotBody["api_key"], "secret-key")
	}
	if gotHeader != "" {
		t.Errorf("got X-Tavily-Access-Mode %q, want it unset when an API key is configured", gotHeader)
	}
	if result != "no results found" {
		t.Errorf("got %q, want %q", result, "no results found")
	}
}

func TestWebSearchUnauthorizedGivesClearError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	tool := NewWebSearchTool("", nil)
	tool.searchURL = server.URL

	_, err := tool.Search(context.Background(), "anything")
	if err == nil || !strings.Contains(err.Error(), "configure a Tavily API key") {
		t.Fatalf("got error %v, want a clear message about configuring a Tavily API key", err)
	}
}
