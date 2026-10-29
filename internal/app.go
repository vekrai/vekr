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
	DB         *gorm.DB
	Sessions   *scs.SessionManager
	Chat       *agent.ChatService
	Distill    *agent.DistillService
	Curriculum *agent.CurriculumService
}

func NewApp(db *gorm.DB) *App {
	sessions := scs.New()
	sessions.Lifetime = 24 * time.Hour

	aiClient := ai.NewClient(ai.ConfigFromEnv(), nil)

	curriculum := agent.NewCurriculumService(db, aiClient)

	webSearch := agent.NewWebSearchTool(os.Getenv("TAVILY_API_KEY"), nil)
	webFetch := agent.NewWebFetchTool()
	knowledge := agent.NewKnowledgeTools(db)
	curriculumTools := agent.NewCurriculumTools(curriculum, sessions)
	tools := agent.NewToolRegistry(webSearch, webFetch, knowledge, curriculumTools)

	chat := agent.NewChatService(db, aiClient, tools)
	distill := agent.NewDistillService(db, aiClient)

	return &App{DB: db, Sessions: sessions, Chat: chat, Distill: distill, Curriculum: curriculum}
}
