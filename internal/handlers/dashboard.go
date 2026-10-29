package handlers

import (
	"net/http"

	"gorm.io/gorm"

	"vekr/internal/models"
	"vekr/web/templates"
)

type DashboardHandler struct {
	db *gorm.DB
}

func NewDashboardHandler(db *gorm.DB) *DashboardHandler {
	return &DashboardHandler{db: db}
}

func (h *DashboardHandler) Index(w http.ResponseWriter, r *http.Request) {
	var recentNotes []models.Node
	if err := h.db.Order("created_at desc").Limit(5).Find(&recentNotes).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var nodeCount int64
	if err := h.db.Model(&models.Node{}).Count(&nodeCount).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	templates.Dashboard(recentNotes, nodeCount).Render(r.Context(), w)
}
