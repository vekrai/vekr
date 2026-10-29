package internal

import (
	"net/http"

	"vekr/internal/handlers"
)

func NewRouter(app *App) http.Handler {
	dashboard := handlers.NewDashboardHandler(app.DB, app.Chat)
	notes := handlers.NewNotesHandler(app.DB)
	chat := handlers.NewChatHandler(app.Chat)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", dashboard.Index)
	mux.HandleFunc("GET /notes", notes.Index)
	mux.HandleFunc("POST /notes", notes.Create)
	mux.HandleFunc("GET /notes/{id}", notes.Show)
	mux.HandleFunc("POST /chat/messages", chat.Create)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	return Chain(mux, Recovery, Logging, app.Sessions.LoadAndSave)
}
