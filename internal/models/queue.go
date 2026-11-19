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
	Passed    int64
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

type ConceptStat struct {
	ID           uint
	Name         string
	MissCount    int
	AttemptCount int
}

type RelatedNote struct {
	NodeID   uint
	Title    string
	Relation string
	Outgoing bool
}
