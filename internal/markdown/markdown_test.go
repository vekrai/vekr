package markdown

import (
	"strings"
	"testing"
)

func TestToHTMLBasicFormatting(t *testing.T) {
	html, err := ToHTML("**bold** and a [link](https://example.com)")
	if err != nil {
		t.Fatalf("ToHTML: %v", err)
	}
	if !strings.Contains(html, "<strong>bold</strong>") || !strings.Contains(html, `<a href="https://example.com">link</a>`) {
		t.Errorf("got %q, want bold and link rendered", html)
	}
}

func TestToHTMLTable(t *testing.T) {
	source := "| a | b |\n|---|---|\n| 1 | 2 |\n"
	html, err := ToHTML(source)
	if err != nil {
		t.Fatalf("ToHTML: %v", err)
	}
	if !strings.Contains(html, "<table>") {
		t.Errorf("got %q, want a rendered table (GFM extension)", html)
	}
}

func TestToHTMLStripsRawScriptTags(t *testing.T) {
	html, err := ToHTML("hello <script>alert('xss')</script> world")
	if err != nil {
		t.Fatalf("ToHTML: %v", err)
	}
	if strings.Contains(html, "<script>") {
		t.Fatalf("got %q, raw HTML must not pass through unescaped", html)
	}
}
