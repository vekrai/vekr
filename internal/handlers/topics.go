package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"gorm.io/gorm"

	"vekr/internal/agent"
	"vekr/internal/markdown"
	"vekr/web/templates"
)

const topicAttemptsLimit = 8

type TopicsHandler struct {
	db      *gorm.DB
	explain *agent.ExplainService
}

func NewTopicsHandler(db *gorm.DB, explain *agent.ExplainService) *TopicsHandler {
	return &TopicsHandler{db: db, explain: explain}
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

	view := templates.TopicView{Progress: progress[0]}
	if view.ContentHTML, err = markdown.ToHTML(progress[0].Topic.Content); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.db.Where("topic_id = ?", topicID).Order("id").Find(&view.Notes).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = h.db.Raw(`
		SELECT c.id, c.name, COALESCE(cw.miss_count, 0) AS miss_count, COALESCE(cw.attempt_count, 0) AS attempt_count
		FROM concepts c
		LEFT JOIN concept_weakness cw ON cw.concept_id = c.id
		WHERE c.topic_id = ?
		ORDER BY COALESCE(cw.miss_count, 0) DESC, c.name
	`, topicID).Scan(&view.Concepts).Error
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = h.db.Raw(`
		SELECT a.*, pi.prompt,
		       EXISTS (SELECT 1 FROM mistake_log m WHERE m.attempt_id = a.id) AS flagged
		FROM attempts a
		JOIN practice_items pi ON pi.id = a.practice_item_id
		WHERE pi.topic_id = ?
		ORDER BY a.created_at DESC, a.id DESC
		LIMIT ?
	`, topicID, topicAttemptsLimit).Scan(&view.Attempts).Error
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	templates.TopicsShow(view).Render(r.Context(), w)
}

func (h *TopicsHandler) Explain(w http.ResponseWriter, r *http.Request) {
	topicID, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid topic id", http.StatusBadRequest)
		return
	}

	node, err := h.explain.Review(r.Context(), uint(topicID), r.FormValue("explanation"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/notes/%d", node.ID), http.StatusSeeOther)
}
