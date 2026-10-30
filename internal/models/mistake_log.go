package models

import "time"

const (
	ErrorTypeMisconception    = "misconception"
	ErrorTypeCarelessSlip     = "careless_slip"
	ErrorTypeKnowledgeGap     = "knowledge_gap"
	ErrorTypeCalculationError = "calculation_error"
)

type MistakeLog struct {
	ID        uint `gorm:"primaryKey"`
	AttemptID uint
	ConceptID uint
	ErrorType string
	CreatedAt time.Time
}

// TableName overrides GORM's default pluralization — the migration names
// this table mistake_log (singular), matching overview.md's schema.
func (MistakeLog) TableName() string { return "mistake_log" }
