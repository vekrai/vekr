package handlers

import (
	"net/http"
	"time"

	"gorm.io/gorm"

	"vekr/internal/agent"
	"vekr/internal/i18n"
	"vekr/internal/models"
	"vekr/web/templates"
)

const dashboardListLimit = 5

const recentChatsLimit = 5

const streakLookbackDays = 400

const graphNodeLimit = 60

type DashboardHandler struct {
	db   *gorm.DB
	chat *agent.ChatService
}

func NewDashboardHandler(db *gorm.DB, chat *agent.ChatService) *DashboardHandler {
	return &DashboardHandler{db: db, chat: chat}
}

func (h *DashboardHandler) Index(w http.ResponseWriter, r *http.Request) {
	var view templates.DashboardView

	if err := h.db.Order("created_at desc").Limit(dashboardListLimit).Find(&view.RecentNotes).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.db.Model(&models.Node{}).Count(&view.NodeCount).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	masteredScope := h.db.Where("status = ?", models.StatusMastered).Session(&gorm.Session{})
	if err := masteredScope.Model(&models.Topic{}).Count(&view.MasteredCount).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := masteredScope.Order("updated_at desc").Limit(dashboardListLimit).Find(&view.Mastered).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var err error
	if view.Queue, err = h.loadQueue(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if view.WeakSpots, err = h.loadWeakSpots(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if view.RecentChats, err = h.chat.RecentChats(recentChatsLimit); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if view.Streak, err = h.loadStreak(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if view.Graph, err = h.loadGraph(i18n.T(r.Context(), "graph.unsorted")); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	templates.Dashboard(view).Render(r.Context(), w)
}

func (h *DashboardHandler) loadQueue() ([]models.QueueItem, error) {
	var queue []models.QueueItem
	err := h.db.Raw(`
		SELECT t.id AS topic_id, t.title,
		       COUNT(pi.id) AS item_count,
		       COUNT(pi.id) FILTER (
		           WHERE latest.passed
		             AND NOT EXISTS (SELECT 1 FROM mistake_log m WHERE m.attempt_id = latest.id)
		       ) AS passed
		FROM topics t
		LEFT JOIN practice_items pi ON pi.topic_id = t.id
		LEFT JOIN LATERAL (
		    SELECT a.id, a.passed
		      FROM attempts a
		     WHERE a.practice_item_id = pi.id
		     ORDER BY a.created_at DESC, a.id DESC
		     LIMIT 1
		) latest ON true
		WHERE t.status <> ?
		GROUP BY t.id, t.title
		ORDER BY GREATEST(t.updated_at, (
		    SELECT MAX(a.created_at)
		      FROM attempts a
		      JOIN practice_items p ON p.id = a.practice_item_id
		     WHERE p.topic_id = t.id
		)) DESC, t.id DESC
	`, models.StatusMastered).Scan(&queue).Error
	return queue, err
}

func (h *DashboardHandler) loadStreak() (int, error) {
	now := time.Now()
	var activity []time.Time
	err := h.db.Model(&models.Attempt{}).
		Where("created_at > ?", now.AddDate(0, 0, -streakLookbackDays)).
		Pluck("created_at", &activity).Error
	if err != nil {
		return 0, err
	}
	return models.Streak(activity, now), nil
}

func (h *DashboardHandler) loadGraph(unsortedLabel string) (templates.GraphView, error) {
	var nodes []models.Node
	if err := h.db.Select("id", "title", "topic_id").Order("id").Limit(graphNodeLimit).Find(&nodes).Error; err != nil {
		return templates.GraphView{}, err
	}
	var links []models.NodeLink
	if err := h.db.Find(&links).Error; err != nil {
		return templates.GraphView{}, err
	}
	var topics []models.Topic
	if err := h.db.Select("id", "title").Order("id").Find(&topics).Error; err != nil {
		return templates.GraphView{}, err
	}
	return templates.BuildGraph(nodes, links, topics, unsortedLabel), nil
}

func (h *DashboardHandler) loadWeakSpots() ([]models.WeakSpot, error) {
	var spots []models.WeakSpot
	err := h.db.Raw(`
		SELECT c.id AS concept_id, c.name, cw.miss_count, cw.attempt_count, cw.last_missed_at
		FROM concept_weakness cw
		JOIN concepts c ON c.id = cw.concept_id
		WHERE cw.miss_count > 0
		ORDER BY cw.last_missed_at DESC NULLS LAST, c.id DESC
		LIMIT ?
	`, dashboardListLimit).Scan(&spots).Error
	return spots, err
}

type topicProgressRow struct {
	models.Topic
	Total  int64
	Passed int64
}

func loadTopicProgress(db *gorm.DB, where string, args ...any) ([]models.TopicProgress, error) {
	if where == "" {
		where = "TRUE"
	}
	var rows []topicProgressRow
	err := db.Raw(`
		SELECT t.*,
		       COUNT(pi.id) AS total,
		       COUNT(pi.id) FILTER (
		           WHERE latest.passed
		             AND NOT EXISTS (SELECT 1 FROM mistake_log m WHERE m.attempt_id = latest.id)
		       ) AS passed
		FROM topics t
		LEFT JOIN practice_items pi ON pi.topic_id = t.id
		LEFT JOIN LATERAL (
		    SELECT a.id, a.passed
		      FROM attempts a
		     WHERE a.practice_item_id = pi.id
		     ORDER BY a.created_at DESC, a.id DESC
		     LIMIT 1
		) latest ON true
		WHERE `+where+`
		GROUP BY t.id
		ORDER BY t.created_at DESC
	`, args...).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	progress := make([]models.TopicProgress, len(rows))
	for i, row := range rows {
		progress[i] = models.TopicProgress{Topic: row.Topic, Total: row.Total, Passed: row.Passed}
	}
	return progress, nil
}
