package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"vekr/internal/agent"
	"vekr/internal/models"
	"vekr/web/templates"
)

const drillItemLimit = 10

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
	if err := h.db.Where("node_id = ?", nodeID).Order("created_at, id").Find(&items).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	latest, err := h.latestAttempts(items)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	templates.PracticeIndex(topic, items, latest, r.URL.RequestURI()).Render(r.Context(), w)
}

func (h *PracticeHandler) Drill(w http.ResponseWriter, r *http.Request) {
	var conceptIDs []uint
	for _, raw := range r.URL.Query()["concept_id"] {
		id, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			http.Error(w, "invalid concept_id", http.StatusBadRequest)
			return
		}
		conceptIDs = append(conceptIDs, uint(id))
	}
	if len(conceptIDs) == 0 {
		http.Error(w, "at least one concept_id is required", http.StatusBadRequest)
		return
	}

	var concepts []models.Concept
	if err := h.db.Where("id IN ?", conceptIDs).Order("name").Find(&concepts).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var items []models.PracticeItem
	err := h.db.
		Where("id IN (SELECT practice_item_id FROM practice_item_concepts WHERE concept_id IN ?)", conceptIDs).
		Order("node_id, created_at, id").
		Find(&items).Error
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	items = interleaveByNode(items)
	if len(items) > drillItemLimit {
		items = items[:drillItemLimit]
	}

	latest, err := h.latestAttempts(items)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	templates.PracticeDrill(concepts, items, latest, r.URL.RequestURI()).Render(r.Context(), w)
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

	returnTo := r.FormValue("return_to")
	if !isLocalPath(returnTo) {
		returnTo = fmt.Sprintf("/topics/%d/practice", item.NodeID)
	}
	http.Redirect(w, r, returnTo, http.StatusSeeOther)
}

func (h *PracticeHandler) latestAttempts(items []models.PracticeItem) (map[uint]models.Attempt, error) {
	latest := make(map[uint]models.Attempt, len(items))
	if len(items) == 0 {
		return latest, nil
	}

	itemIDs := make([]uint, len(items))
	for i, item := range items {
		itemIDs[i] = item.ID
	}

	var attempts []models.Attempt
	err := h.db.Raw(`
		SELECT DISTINCT ON (practice_item_id) *
		FROM attempts
		WHERE practice_item_id IN ?
		ORDER BY practice_item_id, created_at DESC, id DESC
	`, itemIDs).Scan(&attempts).Error
	if err != nil {
		return nil, err
	}

	for _, a := range attempts {
		latest[a.PracticeItemID] = a
	}
	return latest, nil
}

func interleaveByNode(items []models.PracticeItem) []models.PracticeItem {
	var order []uint
	groups := make(map[uint][]models.PracticeItem)
	for _, item := range items {
		if _, seen := groups[item.NodeID]; !seen {
			order = append(order, item.NodeID)
		}
		groups[item.NodeID] = append(groups[item.NodeID], item)
	}

	out := make([]models.PracticeItem, 0, len(items))
	for len(out) < len(items) {
		for _, nodeID := range order {
			if group := groups[nodeID]; len(group) > 0 {
				out = append(out, group[0])
				groups[nodeID] = group[1:]
			}
		}
	}
	return out
}

func isLocalPath(path string) bool {
	return strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "//") && !strings.HasPrefix(path, "/\\")
}
