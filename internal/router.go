package internal

import (
	"net/http"

	"vekr/internal/assets"
	"vekr/internal/handlers"
	"vekr/internal/i18n"
)

func NewRouter(app *App) http.Handler {
	dashboard := handlers.NewDashboardHandler(app.DB, app.Chat)
	notes := handlers.NewNotesHandler(app.DB, app.Curriculum)
	chat := handlers.NewChatHandler(app.Chat, app.Distill)
	topics := handlers.NewTopicsHandler(app.DB, app.Explain)
	practice := handlers.NewPracticeHandler(app.DB, app.Grading, app.Practice)
	streams := handlers.NewStreamsHandler(app.Hub, app.Chat)
	locale := handlers.NewLocaleHandler(app.DB)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", dashboard.Index)
	mux.HandleFunc("GET /notes", notes.Index)
	mux.HandleFunc("POST /notes", notes.Create)
	mux.HandleFunc("GET /notes/{id}", notes.Show)
	mux.HandleFunc("GET /notes/{id}/edit", notes.Edit)
	mux.HandleFunc("POST /notes/{id}", notes.Update)
	mux.HandleFunc("POST /notes/{id}/delete", notes.Delete)
	mux.HandleFunc("POST /notes/{id}/promote", notes.Promote)
	mux.HandleFunc("POST /chat/messages", chat.Create)
	mux.HandleFunc("POST /chat/{id}/distill", chat.Distill)
	mux.HandleFunc("GET /streams/chat/{id}", streams.Chat)
	mux.HandleFunc("GET /topics/{id}", topics.Show)
	mux.HandleFunc("POST /topics/{id}/explain", topics.Explain)
	mux.HandleFunc("GET /topics/{id}/practice", practice.Index)
	mux.HandleFunc("GET /drill", practice.Drill)
	mux.HandleFunc("POST /drill/generate", practice.GenerateDrill)
	mux.HandleFunc("POST /practice_items/{id}/attempts", practice.Submit)
	mux.HandleFunc("POST /locale", locale.Toggle)
	mux.Handle("GET /static/", assets.Handler())

	return Chain(mux, Recovery, Logging, app.Sessions.LoadAndSave, i18n.Middleware(handlers.SettingLocale(app.DB)), handlers.ChatState(app.Chat))
}
