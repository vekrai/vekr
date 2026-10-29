package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebFetchConvertsHTMLToMarkdown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(`<html><head><style>body{color:red}</style></head><body><h1>Hi &amp; welcome</h1><p>hello <strong>world</strong></p><a href="https://example.com">a link</a><script>evil()</script></body></html>`))
	}))
	defer server.Close()

	tool := NewWebFetchTool()
	text, err := tool.Fetch(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	if !strings.Contains(text, "# Hi & welcome") {
		t.Errorf("got %q, want a markdown heading for the <h1>", text)
	}
	if !strings.Contains(text, "**world**") {
		t.Errorf("got %q, want markdown bold syntax for the <strong>", text)
	}
	if !strings.Contains(text, "[a link](https://example.com)") {
		t.Errorf("got %q, want markdown link syntax", text)
	}
	if strings.Contains(text, "evil()") || strings.Contains(text, "color:red") {
		t.Errorf("got %q, script/style content should have been stripped", text)
	}
}

func TestWebFetchRejectsDisallowedContentType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte("binary"))
	}))
	defer server.Close()

	tool := NewWebFetchTool()
	_, err := tool.Fetch(context.Background(), server.URL)
	if err == nil {
		t.Fatal("expected an error for disallowed content type, got nil")
	}
}

func TestWebFetchRejectsNonHTTPScheme(t *testing.T) {
	tool := NewWebFetchTool()
	_, err := tool.Fetch(context.Background(), "file:///etc/passwd")
	if err == nil {
		t.Fatal("expected an error for a non-http(s) scheme, got nil")
	}
}

func TestWebFetchTruncatesLargeResponses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(strings.Repeat("a", webfetchMaxBytes+1000)))
	}))
	defer server.Close()

	tool := NewWebFetchTool()
	text, err := tool.Fetch(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(text) != webfetchMaxBytes {
		t.Errorf("got %d bytes, want exactly %d (truncated)", len(text), webfetchMaxBytes)
	}
}
