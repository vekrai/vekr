package models

import "time"

const (
	PracticeItemTypeMCQ       = "mcq"
	PracticeItemTypeFlashcard = "flashcard"
	PracticeItemTypeOpen      = "open"
	PracticeItemTypeCode      = "code"
)

type PracticeItem struct {
	ID        uint `gorm:"primaryKey"`
	TopicID   uint
	Prompt    string
	AnswerKey *string
	Type      string
	CreatedAt time.Time
}

type PracticeItemConcept struct {
	PracticeItemID uint `gorm:"primaryKey"`
	ConceptID      uint `gorm:"primaryKey"`
}
