package models

import "time"

type Node struct {
	ID           uint `gorm:"primaryKey"`
	Title        string
	Content      string
	SourceChatID *uint
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
