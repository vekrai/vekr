package agent

import (
	"context"
	"strings"
	"testing"

	"gorm.io/gorm"

	"vekr/internal/models"
)

func testDB(t *testing.T) *gorm.DB {
	db, err := models.Connect()
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	tx := db.Begin()
	t.Cleanup(func() {
		tx.Rollback()
	})

	return tx
}

func TestKnowledgeToolsSearchAndReadNote(t *testing.T) {
	db := testDB(t)

	node := models.Node{Title: "Go generics", Content: "Generics let you write functions over any type."}
	if err := db.Create(&node).Error; err != nil {
		t.Fatalf("create node: %v", err)
	}

	tools := NewKnowledgeTools(db)

	searchResult, err := tools.SearchNotes(context.Background(), "generics")
	if err != nil {
		t.Fatalf("SearchNotes: %v", err)
	}
	if !strings.Contains(searchResult, "Go generics") {
		t.Errorf("got %q, want it to mention the matching note's title", searchResult)
	}

	readResult, err := tools.ReadNote(context.Background(), node.ID)
	if err != nil {
		t.Fatalf("ReadNote: %v", err)
	}
	if !strings.Contains(readResult, "Generics let you write functions over any type.") {
		t.Errorf("got %q, want full note content", readResult)
	}
}

func TestKnowledgeToolsSearchNotesMatchesCJKSubstring(t *testing.T) {
	db := testDB(t)

	node := models.Node{Title: "笔记", Content: "机器学习是人工智能的分支，它让计算机从数据中学习。"}
	if err := db.Create(&node).Error; err != nil {
		t.Fatalf("create node: %v", err)
	}

	tools := NewKnowledgeTools(db)

	result, err := tools.SearchNotes(context.Background(), "人工智能")
	if err != nil {
		t.Fatalf("SearchNotes: %v", err)
	}
	if !strings.Contains(result, "笔记") {
		t.Errorf("got %q, want it to match the note containing 人工智能 as a substring", result)
	}
}

func TestKnowledgeToolsSearchNotesNoMatch(t *testing.T) {
	db := testDB(t)
	tools := NewKnowledgeTools(db)

	result, err := tools.SearchNotes(context.Background(), "zzz-nonexistent-term-zzz")
	if err != nil {
		t.Fatalf("SearchNotes: %v", err)
	}
	if result != "no matching notes found" {
		t.Errorf("got %q, want %q", result, "no matching notes found")
	}
}

func TestKnowledgeToolsReadNoteNotFound(t *testing.T) {
	db := testDB(t)
	tools := NewKnowledgeTools(db)

	_, err := tools.ReadNote(context.Background(), 999999)
	if err == nil {
		t.Fatal("expected an error for a missing node, got nil")
	}
}

func strPtr(s string) *string { return &s }

func TestKnowledgeToolsEditNoteUpdatesOnlyGivenFields(t *testing.T) {
	db := testDB(t)

	node := models.Node{Title: "Old title", Content: "Old content"}
	if err := db.Create(&node).Error; err != nil {
		t.Fatalf("create node: %v", err)
	}

	tools := NewKnowledgeTools(db)

	if _, err := tools.EditNote(context.Background(), node.ID, strPtr("New title"), nil); err != nil {
		t.Fatalf("EditNote: %v", err)
	}

	var reloaded models.Node
	if err := db.First(&reloaded, node.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Title != "New title" {
		t.Errorf("got title %q, want %q", reloaded.Title, "New title")
	}
	if reloaded.Content != "Old content" {
		t.Errorf("got content %q, want it unchanged (%q)", reloaded.Content, "Old content")
	}

	if _, err := tools.EditNote(context.Background(), node.ID, nil, strPtr("New content")); err != nil {
		t.Fatalf("EditNote: %v", err)
	}
	if err := db.First(&reloaded, node.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Content != "New content" {
		t.Errorf("got content %q, want %q", reloaded.Content, "New content")
	}
	if reloaded.Title != "New title" {
		t.Errorf("got title %q, want it unchanged (%q)", reloaded.Title, "New title")
	}
}

func TestKnowledgeToolsEditNoteRequiresAField(t *testing.T) {
	db := testDB(t)

	node := models.Node{Title: "Title", Content: "Content"}
	if err := db.Create(&node).Error; err != nil {
		t.Fatalf("create node: %v", err)
	}

	tools := NewKnowledgeTools(db)

	if _, err := tools.EditNote(context.Background(), node.ID, nil, nil); err == nil {
		t.Fatal("expected an error when neither title nor content is given")
	}
}
