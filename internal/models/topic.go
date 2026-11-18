package models

import "time"

const (
	StatusNotStarted = "not_started"
	StatusLearning   = "learning"
	StatusPracticing = "practicing"
	StatusMastered   = "mastered"
)

type Topic struct {
	ID        uint `gorm:"primaryKey"`
	Title     string
	Content   string
	Status    string `gorm:"default:not_started"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type TopicProgress struct {
	Topic  Topic
	Total  int64
	Passed int64
}
