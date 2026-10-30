package models

import "time"

type Chat struct {
	ID        uint `gorm:"primaryKey"`
	NodeID    *uint
	CreatedAt time.Time
}

type ChatMessage struct {
	ID         uint `gorm:"primaryKey"`
	ChatID     uint
	Role       string
	Content    string
	ToolCallID string
	ToolCalls  string
	Name       string
	CreatedAt  time.Time
}
