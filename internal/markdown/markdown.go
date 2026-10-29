package markdown

import (
	"bytes"
	"fmt"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

var renderer = goldmark.New(goldmark.WithExtensions(extension.GFM))

func ToHTML(source string) (string, error) {
	var buf bytes.Buffer
	if err := renderer.Convert([]byte(source), &buf); err != nil {
		return "", fmt.Errorf("markdown: render: %w", err)
	}
	return buf.String(), nil
}
