package models

import "time"

type Attempt struct {
	ID             uint `gorm:"primaryKey"`
	PracticeItemID uint
	Submission     string
	Passed         *bool
	AgentFeedback  string
	CreatedAt      time.Time
}
