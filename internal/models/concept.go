package models

type Concept struct {
	ID      uint `gorm:"primaryKey"`
	TopicID *uint
	Name    string
}
