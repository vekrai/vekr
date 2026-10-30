package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"gorm.io/gorm"

	"vekr/internal/agent"
	"vekr/internal/models"
	"vekr/web/templates"
)

type PracticeHandler struct {
	db      *gorm.DB
	grading *agent.GradingService
}

func NewPracticeHandler(db *gorm.DB, grading *agent.GradingService) *PracticeHandler {
	return &PracticeHandler{db: db, grading: grading}
}

func (h *PracticeHandler) Index(w http.ResponseWriter, r *http.Request) {
	nodeID, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid topic id", http.StatusBadRequest)
		return
	}

	var topic models.Node
	if err := h.db.First(&topic, nodeID).Error; err != nil {
		http.NotFound(w, r)
		return
	}

	var items []models.PracticeItem
	if err := h.db.Where("node_id = ?", nodeID).Order("created_at").Find(&items).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	latestAttempts := make(map[uint]models.Attempt, len(items))
	for _, item := range items {
		var attempt models.Attempt
		if err := h.db.Where("practice_item_id = ?", item.ID).Order("created_at desc").First(&attempt).Error; err == nil {
			latestAttempts[item.ID] = attempt
		}
	}

	templates.PracticeIndex(topic, items, latestAttempts).Render(r.Context(), w)
}

func (h *PracticeHandler) Submit(w http.ResponseWriter, r *http.Request) {
	itemID, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid practice item id", http.StatusBadRequest)
		return
	}

	var item models.PracticeItem
	if err := h.db.First(&item, itemID).Error; err != nil {
		http.NotFound(w, r)
		return
	}

	if _, err := h.grading.Grade(r.Context(), uint(itemID), r.FormValue("submission")); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/topics/%d/practice", item.NodeID), http.StatusSeeOther)
}
