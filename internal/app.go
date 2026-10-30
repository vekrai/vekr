package internal

import (
	"os"
	"time"

	"github.com/alexedwards/scs/v2"
	"gorm.io/gorm"

	"vekr/internal/agent"
	"vekr/internal/ai"
	"vekr/internal/live"
)

type App struct {
	DB         *gorm.DB
	Sessions   *scs.SessionManager
	Chat       *agent.ChatService
	Distill    *agent.DistillService
	Curriculum *agent.CurriculumService
	Grading    *agent.GradingService
	Hub        *live.Hub
}

func NewApp(db *gorm.DB) *App {
	sessions := scs.New()
	sessions.Lifetime = 24 * time.Hour

	hub := live.NewHub()
	aiClient := ai.NewClient(ai.ConfigFromEnv(), nil)

	curriculum := agent.NewCurriculumService(db, aiClient)
	practice := agent.NewPracticeService(db, aiClient)
	grading := agent.NewGradingService(db, aiClient)

	webSearch := agent.NewWebSearchTool(os.Getenv("TAVILY_API_KEY"), nil)
	webFetch := agent.NewWebFetchTool()
	knowledge := agent.NewKnowledgeTools(db)
	curriculumTools := agent.NewCurriculumTools(curriculum)
	practiceTools := agent.NewPracticeTools(practice)
	tools := agent.NewToolRegistry(webSearch, webFetch, knowledge, curriculumTools, practiceTools)

	chat := agent.NewChatService(db, aiClient, tools, hub)
	distill := agent.NewDistillService(db, aiClient)

	return &App{DB: db, Sessions: sessions, Chat: chat, Distill: distill, Curriculum: curriculum, Grading: grading, Hub: hub}
}
