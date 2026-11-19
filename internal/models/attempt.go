package models

import "time"

const (
	VerdictPassed         = "passed"
	VerdictPassedWithNote = "passed_with_note"
	VerdictNotYet         = "not_yet"
)

type Attempt struct {
	ID             uint `gorm:"primaryKey"`
	PracticeItemID uint
	Submission     string
	Passed         *bool
	AgentFeedback  string
	CreatedAt      time.Time
}

type AttemptResult struct {
	Attempt
	Flagged bool
	Prompt  string
}

func (a AttemptResult) Verdict() string {
	if a.Passed == nil || !*a.Passed {
		return VerdictNotYet
	}
	if a.Flagged {
		return VerdictPassedWithNote
	}
	return VerdictPassed
}
