package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"vekr/internal/agent"
	"vekr/internal/models"
	"vekr/web/templates"
)

const drillItemLimit = 10

const drillGeneratePerTopic = 3

type PracticeHandler struct {
	db       *gorm.DB
	grading  *agent.GradingService
	practice *agent.PracticeService
}

func NewPracticeHandler(db *gorm.DB, grading *agent.GradingService, practice *agent.PracticeService) *PracticeHandler {
	return &PracticeHandler{db: db, grading: grading, practice: practice}
}

func (h *PracticeHandler) Index(w http.ResponseWriter, r *http.Request) {
	topicID, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid topic id", http.StatusBadRequest)
		return
	}

	var topic models.Topic
	if err := h.db.First(&topic, topicID).Error; err != nil {
		http.NotFound(w, r)
		return
	}

	var items []models.PracticeItem
	if err := h.db.Where("topic_id = ?", topicID).Order("created_at, id").Find(&items).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	latest, err := h.latestAttempts(items)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	templates.PracticeIndex(templates.PracticeView{
		Topic:    &topic,
		Items:    items,
		Latest:   latest,
		ReturnTo: withoutGraded(r.URL),
		Graded:   gradedItem(r),
	}).Render(r.Context(), w)
}

func (h *PracticeHandler) Drill(w http.ResponseWriter, r *http.Request) {
	conceptIDs, err := parseConceptIDs(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var concepts []models.Concept
	if err := h.db.Where("id IN ?", conceptIDs).Order("name").Find(&concepts).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var items []models.PracticeItem
	err = h.db.
		Where("id IN (SELECT practice_item_id FROM practice_item_concepts WHERE concept_id IN ?)", conceptIDs).
		Order("topic_id, created_at, id").
		Find(&items).Error
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	latest, err := h.latestAttempts(items)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	items = drillOrder(items, latest)
	if len(items) > drillItemLimit {
		items = items[:drillItemLimit]
	}

	topicTitles := map[uint]string{}
	var topics []models.Topic
	if err := h.db.Select("id", "title").Find(&topics).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, t := range topics {
		topicTitles[t.ID] = t.Title
	}

	templates.PracticeDrill(templates.PracticeView{
		Concepts:    concepts,
		ConceptIDs:  conceptIDs,
		Items:       items,
		Latest:      latest,
		TopicTitles: topicTitles,
		ReturnTo:    withoutGraded(r.URL),
		Graded:      gradedItem(r),
	}).Render(r.Context(), w)
}

func (h *PracticeHandler) GenerateDrill(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	conceptIDs, err := parseConceptIDs(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if _, err := h.practice.GenerateForConcepts(r.Context(), conceptIDs, drillGeneratePerTopic); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	q := url.Values{}
	for _, id := range conceptIDs {
		q.Add("concept_id", strconv.FormatUint(uint64(id), 10))
	}
	http.Redirect(w, r, "/drill?"+q.Encode(), http.StatusSeeOther)
}

func parseConceptIDs(r *http.Request) ([]uint, error) {
	raw := r.URL.Query()["concept_id"]
	if r.Method == http.MethodPost {
		raw = r.PostForm["concept_id"]
	}
	var ids []uint
	for _, value := range raw {
		id, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid concept_id")
		}
		ids = append(ids, uint(id))
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("at least one concept_id is required")
	}
	return ids, nil
}

func drillOrder(items []models.PracticeItem, latest map[uint]models.AttemptResult) []models.PracticeItem {
	var open, done []models.PracticeItem
	for _, item := range items {
		if attempt, ok := latest[item.ID]; ok && attempt.Verdict() == models.VerdictPassed {
			done = append(done, item)
			continue
		}
		open = append(open, item)
	}
	return append(interleaveByTopic(open), interleaveByTopic(done)...)
}

func gradedItem(r *http.Request) uint {
	id, _ := strconv.ParseUint(r.URL.Query().Get("graded"), 10, 64)
	return uint(id)
}

func withoutGraded(u *url.URL) string {
	copied := *u
	q := copied.Query()
	q.Del("graded")
	copied.RawQuery = q.Encode()
	return copied.RequestURI()
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
		returnTo = fmt.Sprintf("/topics/%d/practice", item.TopicID)
	}
	http.Redirect(w, r, withGraded(returnTo, item.ID), http.StatusSeeOther)
}

func withGraded(path string, itemID uint) string {
	u, err := url.Parse(path)
	if err != nil {
		return path
	}
	q := u.Query()
	q.Set("graded", strconv.FormatUint(uint64(itemID), 10))
	u.RawQuery = q.Encode()
	u.Fragment = fmt.Sprintf("item-%d", itemID)
	return u.String()
}

func (h *PracticeHandler) latestAttempts(items []models.PracticeItem) (map[uint]models.AttemptResult, error) {
	latest := make(map[uint]models.AttemptResult, len(items))
	if len(items) == 0 {
		return latest, nil
	}

	itemIDs := make([]uint, len(items))
	for i, item := range items {
		itemIDs[i] = item.ID
	}

	var attempts []models.AttemptResult
	err := h.db.Raw(`
		SELECT DISTINCT ON (a.practice_item_id) a.*,
		       EXISTS (SELECT 1 FROM mistake_log m WHERE m.attempt_id = a.id) AS flagged
		FROM attempts a
		WHERE a.practice_item_id IN ?
		ORDER BY a.practice_item_id, a.created_at DESC, a.id DESC
	`, itemIDs).Scan(&attempts).Error
	if err != nil {
		return nil, err
	}

	for _, a := range attempts {
		latest[a.PracticeItemID] = a
	}
	return latest, nil
}

func interleaveByTopic(items []models.PracticeItem) []models.PracticeItem {
	var order []uint
	groups := make(map[uint][]models.PracticeItem)
	for _, item := range items {
		if _, seen := groups[item.TopicID]; !seen {
			order = append(order, item.TopicID)
		}
		groups[item.TopicID] = append(groups[item.TopicID], item)
	}

	out := make([]models.PracticeItem, 0, len(items))
	for len(out) < len(items) {
		for _, topicID := range order {
			if group := groups[topicID]; len(group) > 0 {
				out = append(out, group[0])
				groups[topicID] = group[1:]
			}
		}
	}
	return out
}

func isLocalPath(path string) bool {
	return strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "//") && !strings.HasPrefix(path, "/\\")
}
