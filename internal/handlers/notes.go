package handlers

import (
	"net/http"

	"gorm.io/gorm"

	"vekr/internal/models"
	"vekr/web/templates"
)

type NotesHandler struct {
	db *gorm.DB
}

func NewNotesHandler(db *gorm.DB) *NotesHandler {
	return &NotesHandler{db: db}
}

func (h *NotesHandler) Index(w http.ResponseWriter, r *http.Request) {
	var nodes []models.Node
	if err := h.db.Order("created_at desc").Find(&nodes).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.NotesIndex(nodes).Render(r.Context(), w)
}

func (h *NotesHandler) Show(w http.ResponseWriter, r *http.Request) {
	var node models.Node
	if err := h.db.First(&node, r.PathValue("id")).Error; err != nil {
		http.NotFound(w, r)
		return
	}
	templates.NotesShow(node).Render(r.Context(), w)
}

func (h *NotesHandler) Create(w http.ResponseWriter, r *http.Request) {
	node := models.Node{
		Title:   r.FormValue("title"),
		Content: r.FormValue("content"),
	}
	if err := h.db.Create(&node).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/notes", http.StatusSeeOther)
}
