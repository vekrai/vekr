package agent

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
)

const (
	webfetchMaxBytes     = 1 << 20
	webfetchTimeout      = 30 * time.Second
	webfetchMaxRedirects = 5
)

var allowedContentTypes = []string{
	"text/html",
	"text/plain",
	"text/markdown",
	"application/json",
	"application/xml",
	"text/xml",
	"application/xhtml+xml",
}

type WebFetchTool struct {
	httpClient *http.Client
}

func NewWebFetchTool() *WebFetchTool {
	client := &http.Client{
		Timeout: webfetchTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= webfetchMaxRedirects {
				return fmt.Errorf("too many redirects")
			}
			if !isAllowedScheme(req.URL.Scheme) {
				return fmt.Errorf("redirect to disallowed scheme %q", req.URL.Scheme)
			}
			return nil
		},
	}
	return &WebFetchTool{httpClient: client}
}

func isAllowedScheme(scheme string) bool {
	return scheme == "http" || scheme == "https"
}

func (t *WebFetchTool) Fetch(ctx context.Context, rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("webfetch: invalid URL: %w", err)
	}
	if !isAllowedScheme(parsed.Scheme) {
		return "", fmt.Errorf("webfetch: scheme %q not allowed, only http/https", parsed.Scheme)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("webfetch: build request: %w", err)
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("webfetch: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("webfetch: unexpected status %d", resp.StatusCode)
	}

	contentType := firstContentType(resp.Header.Get("Content-Type"))
	if !contentTypeAllowed(contentType) {
		return "", fmt.Errorf("webfetch: content type %q not allowed", contentType)
	}

	limited := io.LimitReader(resp.Body, webfetchMaxBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return "", fmt.Errorf("webfetch: read response: %w", err)
	}
	if len(body) > webfetchMaxBytes {
		body = body[:webfetchMaxBytes]
	}

	text := string(body)
	if contentType == "text/html" || contentType == "application/xhtml+xml" {
		markdown, err := htmltomarkdown.ConvertString(text)
		if err != nil {
			return "", fmt.Errorf("webfetch: convert html to markdown: %w", err)
		}
		text = markdown
	}
	return text, nil
}

func firstContentType(header string) string {
	if idx := strings.Index(header, ";"); idx != -1 {
		header = header[:idx]
	}
	return strings.TrimSpace(strings.ToLower(header))
}

func contentTypeAllowed(contentType string) bool {
	for _, allowed := range allowedContentTypes {
		if contentType == allowed {
			return true
		}
	}
	return false
}
