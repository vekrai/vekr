package models

import "time"

type Chat struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
}

type ChatMessage struct {
	ID        uint `gorm:"primaryKey"`
	ChatID    uint
	Role      string
	Content   string
	CreatedAt time.Time
}
