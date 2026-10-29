package internal

import (
	"time"

	"github.com/alexedwards/scs/v2"
	"gorm.io/gorm"
)

type App struct {
	DB       *gorm.DB
	Sessions *scs.SessionManager
}

func NewApp(db *gorm.DB) *App {
	sessions := scs.New()
	sessions.Lifetime = 24 * time.Hour

	return &App{DB: db, Sessions: sessions}
}
