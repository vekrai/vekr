package models

import "testing"

func TestNodeInsertAndRead(t *testing.T) {
	db := testDB(t)

	node := Node{Title: "Test note", Content: "hello"}
	if err := db.Create(&node).Error; err != nil {
		t.Fatalf("create: %v", err)
	}

	var found Node
	if err := db.First(&found, node.ID).Error; err != nil {
		t.Fatalf("read back: %v", err)
	}

	if found.Title != "Test note" {
		t.Errorf("got title %q, want %q", found.Title, "Test note")
	}
}
