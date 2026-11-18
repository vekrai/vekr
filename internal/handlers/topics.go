package handlers

import (
	"net/http"
	"strconv"

	"gorm.io/gorm"

	"vekr/internal/markdown"
	"vekr/internal/models"
	"vekr/web/templates"
)

type TopicsHandler struct {
	db *gorm.DB
}

func NewTopicsHandler(db *gorm.DB) *TopicsHandler {
	return &TopicsHandler{db: db}
}

func (h *TopicsHandler) Show(w http.ResponseWriter, r *http.Request) {
	topicID, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid topic id", http.StatusBadRequest)
		return
	}

	progress, err := loadTopicProgress(h.db, "t.id = ?", topicID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(progress) == 0 {
		http.NotFound(w, r)
		return
	}

	contentHTML, err := markdown.ToHTML(progress[0].Topic.Content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var notes []models.Node
	if err := h.db.Where("topic_id = ?", topicID).Order("id").Find(&notes).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	templates.TopicsShow(progress[0], contentHTML, notes).Render(r.Context(), w)
}
