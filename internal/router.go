package internal

import (
	"net/http"

	"gorm.io/gorm"

	"vekr/internal/handlers"
)

func NewRouter(db *gorm.DB) http.Handler {
	notes := handlers.NewNotesHandler(db)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/notes", http.StatusSeeOther)
	})
	mux.HandleFunc("GET /notes", notes.Index)
	mux.HandleFunc("POST /notes", notes.Create)
	mux.HandleFunc("GET /notes/{id}", notes.Show)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	return mux
}
