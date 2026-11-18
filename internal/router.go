package internal

import (
	"net/http"

	"vekr/internal/handlers"
	"vekr/internal/i18n"
)

func NewRouter(app *App) http.Handler {
	dashboard := handlers.NewDashboardHandler(app.DB, app.Chat)
	notes := handlers.NewNotesHandler(app.DB, app.Curriculum, app.Chat)
	chat := handlers.NewChatHandler(app.Chat, app.Distill)
	topics := handlers.NewTopicsHandler(app.DB)
	practice := handlers.NewPracticeHandler(app.DB, app.Grading)
	streams := handlers.NewStreamsHandler(app.Hub, app.Chat)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", dashboard.Index)
	mux.HandleFunc("GET /notes", notes.Index)
	mux.HandleFunc("POST /notes", notes.Create)
	mux.HandleFunc("GET /notes/{id}", notes.Show)
	mux.HandleFunc("POST /notes/{id}/promote", notes.Promote)
	mux.HandleFunc("POST /chat/messages", chat.Create)
	mux.HandleFunc("POST /chat/new", chat.New)
	mux.HandleFunc("POST /chat/{id}/distill", chat.Distill)
	mux.HandleFunc("GET /streams/chat/{id}", streams.Chat)
	mux.HandleFunc("GET /topics/{id}", topics.Show)
	mux.HandleFunc("GET /topics/{id}/practice", practice.Index)
	mux.HandleFunc("GET /drill", practice.Drill)
	mux.HandleFunc("POST /practice_items/{id}/attempts", practice.Submit)
	mux.Handle("GET /static/", Revalidate(http.StripPrefix("/static/", http.FileServer(http.Dir("web/static")))))

	return Chain(mux, Recovery, Logging, app.Sessions.LoadAndSave, i18n.Middleware(app.Sessions))
}
