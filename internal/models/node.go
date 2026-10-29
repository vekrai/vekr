package models

import "time"

const (
	NodeTypeNote  = "note"
	NodeTypeTopic = "topic"
)

const (
	StatusNotStarted = "not_started"
	StatusLearning   = "learning"
	StatusPracticing = "practicing"
	StatusMastered   = "mastered"
)

type Node struct {
	ID           uint   `gorm:"primaryKey"`
	Title        string
	Content      string
	Type         string `gorm:"default:note"`
	PathID       *uint
	Status       *string
	SourceChatID *uint
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
