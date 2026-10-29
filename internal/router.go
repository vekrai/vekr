package internal

import (
	"net/http"

	"vekr/internal/handlers"
)

func NewRouter(app *App) http.Handler {
	notes := handlers.NewNotesHandler(app.DB)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/notes", http.StatusSeeOther)
	})
	mux.HandleFunc("GET /notes", notes.Index)
	mux.HandleFunc("POST /notes", notes.Create)
	mux.HandleFunc("GET /notes/{id}", notes.Show)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	return Chain(mux, Recovery, Logging, app.Sessions.LoadAndSave)
}
