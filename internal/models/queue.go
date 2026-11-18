package models

import "time"

const (
	QueueKindLearn    = "learn"
	QueueKindPractice = "practice"
)

type QueueItem struct {
	TopicID   uint
	Title     string
	ItemCount int64
}

func (q QueueItem) Kind() string {
	if q.ItemCount > 0 {
		return QueueKindPractice
	}
	return QueueKindLearn
}

type WeakSpot struct {
	ConceptID    uint
	Name         string
	MissCount    int
	AttemptCount int
	LastMissedAt *time.Time
}
