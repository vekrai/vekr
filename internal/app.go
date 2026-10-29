package internal

import (
	"os"
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
	Distill  *agent.DistillService
}

func NewApp(db *gorm.DB) *App {
	sessions := scs.New()
	sessions.Lifetime = 24 * time.Hour

	aiClient := ai.NewClient(ai.ConfigFromEnv(), nil)

	webSearch := agent.NewWebSearchTool(os.Getenv("TAVILY_API_KEY"), nil)
	webFetch := agent.NewWebFetchTool()
	knowledge := agent.NewKnowledgeTools(db)
	tools := agent.NewToolRegistry(webSearch, webFetch, knowledge)

	chat := agent.NewChatService(db, aiClient, tools)
	distill := agent.NewDistillService(db, aiClient)

	return &App{DB: db, Sessions: sessions, Chat: chat, Distill: distill}
}
