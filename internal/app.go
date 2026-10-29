package internal

import (
	"time"

	"github.com/alexedwards/scs/v2"
	"gorm.io/gorm"

	"vekr/internal/agent"
	"vekr/internal/ai"
)

type App struct {
	DB       *gorm.DB
	Sessions *scs.SessionManager
	Chat     *agent.ChatService
}

func NewApp(db *gorm.DB) *App {
	sessions := scs.New()
	sessions.Lifetime = 24 * time.Hour

	aiClient := ai.NewClient(ai.ConfigFromEnv(), nil)
	chat := agent.NewChatService(db, aiClient)

	return &App{DB: db, Sessions: sessions, Chat: chat}
}
