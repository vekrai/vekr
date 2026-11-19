package models

import "time"

type Chat struct {
	ID        uint `gorm:"primaryKey"`
	NodeID    *uint
	CreatedAt time.Time
}

type ChatSummary struct {
	ChatID    uint
	Title     string
	LastAt    time.Time
	NodeID    *uint
	NodeTitle *string
}

type ChatMessage struct {
	ID          uint `gorm:"primaryKey"`
	ChatID      uint
	Role        string
	Content     string
	ToolCallID  string
	ToolCalls   string
	Name        string
	TotalTokens int
	CreatedAt   time.Time
}
