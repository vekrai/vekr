package agent

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"vekr/internal/models"
)

type KnowledgeTools struct {
	db *gorm.DB
}

func NewKnowledgeTools(db *gorm.DB) *KnowledgeTools {
	return &KnowledgeTools{db: db}
}

// SearchNotes matches two ways: English full-text search (word-tokenized,
// stemmed, ranked) and a trigram-based substring/similarity match. The
// trigram path exists because to_tsvector('english', ...) only splits on
// whitespace/punctuation — it treats a whole run of CJK characters as one
// untokenized lexeme, so English FTS alone never matches a CJK query like
// "人工智能" against a larger block of Chinese text. Trigram matching works
// on raw character sequences instead of words, so it works regardless of
// script.
func (k *KnowledgeTools) SearchNotes(ctx context.Context, query string) (string, error) {
	rows, err := models.SearchNotes(k.db.WithContext(ctx), query, 10)
	if err != nil {
		return "", fmt.Errorf("search_notes: %w", err)
	}

	if len(rows) == 0 {
		return "no matching notes found", nil
	}

	var sb strings.Builder
	for _, row := range rows {
		fmt.Fprintf(&sb, "node_id=%d", row.NodeID)
		if row.TopicID != nil {
			fmt.Fprintf(&sb, " topic_id=%d", *row.TopicID)
		}
		fmt.Fprintf(&sb, " title=%q\n%s\n\n", row.Title, row.Snippet)
	}
	return strings.TrimSpace(sb.String()), nil
}

func (k *KnowledgeTools) ReadNote(ctx context.Context, nodeID uint) (string, error) {
	var node models.Node
	if err := k.db.WithContext(ctx).First(&node, nodeID).Error; err != nil {
		return "", fmt.Errorf("read_note: note %d not found", nodeID)
	}
	return fmt.Sprintf("Title: %s\n\n%s", node.Title, node.Content), nil
}

// EditNote updates an existing note's title and/or content — a single-row
// edit, so it commits directly with no preview step (editing after save is
// cheap, unlike undoing a graph-shaped curriculum commit). Only the fields
// passed as non-nil are changed.
func (k *KnowledgeTools) EditNote(ctx context.Context, nodeID uint, title, content *string) (string, error) {
	updates := map[string]any{}
	if title != nil {
		updates["title"] = *title
	}
	if content != nil {
		updates["content"] = *content
	}
	if len(updates) == 0 {
		return "", fmt.Errorf("edit_note: nothing to update — provide a title or content")
	}

	if err := k.db.WithContext(ctx).Model(&models.Node{}).Where("id = ?", nodeID).Updates(updates).Error; err != nil {
		return "", fmt.Errorf("edit_note: %w", err)
	}

	var node models.Node
	if err := k.db.WithContext(ctx).First(&node, nodeID).Error; err != nil {
		return "", fmt.Errorf("edit_note: note %d not found", nodeID)
	}
	return fmt.Sprintf("Updated note %q (id=%d).", node.Title, node.ID), nil
}
