package models

const (
	RelationRelated        = "related"
	RelationPrerequisiteOf = "prerequisite_of"
	RelationSourceOf       = "source_of"
)

type NodeLink struct {
	ID         uint `gorm:"primaryKey"`
	FromNodeID uint
	ToNodeID   uint
	Relation   string
}
