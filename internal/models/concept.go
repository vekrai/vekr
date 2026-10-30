package models

type Concept struct {
	ID     uint `gorm:"primaryKey"`
	PathID *uint
	Name   string
}

type NodeConcept struct {
	NodeID    uint `gorm:"primaryKey"`
	ConceptID uint `gorm:"primaryKey"`
}
