package models

import "time"

type Path struct {
	ID        uint `gorm:"primaryKey"`
	Title     string
	CreatedAt time.Time
}

// PathProgress is a read-only aggregate, not a table — mastered topics
// versus total topics for one Path, used by the Active Learning Paths card.
type PathProgress struct {
	Path     Path
	Total    int64
	Mastered int64
}
