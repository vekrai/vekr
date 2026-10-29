package models

import (
	"testing"

	"gorm.io/gorm"
)

func testDB(t *testing.T) *gorm.DB {
	db, err := Connect()
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	tx := db.Begin()
	t.Cleanup(func() {
		tx.Rollback()
	})

	return tx
}
