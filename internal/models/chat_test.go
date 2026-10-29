package models

import "testing"

func TestChatMessageInsertAndRead(t *testing.T) {
	db := testDB(t)

	chat := Chat{}
	if err := db.Create(&chat).Error; err != nil {
		t.Fatalf("create chat: %v", err)
	}

	msg := ChatMessage{ChatID: chat.ID, Role: "user", Content: "hello"}
	if err := db.Create(&msg).Error; err != nil {
		t.Fatalf("create message: %v", err)
	}

	var found []ChatMessage
	if err := db.Where("chat_id = ?", chat.ID).Order("created_at").Find(&found).Error; err != nil {
		t.Fatalf("read back: %v", err)
	}

	if len(found) != 1 || found[0].Content != "hello" {
		t.Fatalf("got %+v, want one message with content %q", found, "hello")
	}
}
