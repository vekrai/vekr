package models

import "time"

// ConceptWeakness is a materialized rollup, recomputed whenever a
// MistakeLog is inserted for that concept — not computed on read, since
// the Weak Spots card reads it on every dashboard load.
type ConceptWeakness struct {
	ConceptID    uint `gorm:"primaryKey"`
	MissCount    int
	AttemptCount int
	LastMissedAt *time.Time
}

// TableName overrides GORM's default pluralization — the migration names
// this table concept_weakness (singular), matching overview.md's schema.
func (ConceptWeakness) TableName() string { return "concept_weakness" }
