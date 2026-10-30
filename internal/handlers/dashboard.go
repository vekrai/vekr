package handlers

import (
	"net/http"

	"gorm.io/gorm"

	"vekr/internal/agent"
	"vekr/internal/models"
	"vekr/web/templates"
)

const dashboardListLimit = 5

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

	masteredScope := h.db.Where("type = ? AND status = ?", models.NodeTypeTopic, models.StatusMastered).Session(&gorm.Session{})
	if err := masteredScope.Model(&models.Node{}).Count(&view.MasteredCount).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := masteredScope.Order("updated_at desc").Limit(dashboardListLimit).Find(&view.Mastered).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var err error
	if view.PathProgress, err = h.loadPathProgress(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if view.Queue, err = h.loadQueue(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if view.WeakSpots, err = h.loadWeakSpots(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if view.Chat, err = h.chat.CurrentChat(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if view.Messages, err = h.chat.History(view.Chat.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if view.MessageHTML, err = renderMessages(view.Messages); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	templates.Dashboard(view).Render(r.Context(), w)
}

func (h *DashboardHandler) loadQueue() ([]models.QueueItem, error) {
	var queue []models.QueueItem
	err := h.db.Raw(`
		SELECT n.id AS node_id, n.title, COUNT(DISTINCT pi.id) AS item_count
		FROM nodes n
		LEFT JOIN practice_items pi ON pi.node_id = n.id
		LEFT JOIN attempts a ON a.practice_item_id = pi.id
		WHERE n.type = ? AND (n.status IS NULL OR n.status <> ?)
		GROUP BY n.id, n.title
		ORDER BY GREATEST(n.updated_at, MAX(a.created_at)) DESC, n.id DESC
		LIMIT ?
	`, models.NodeTypeTopic, models.StatusMastered, dashboardListLimit).Scan(&queue).Error
	return queue, err
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

type pathProgressRow struct {
	ID       uint
	Title    string
	Total    int64
	Mastered int64
}

func (h *DashboardHandler) loadPathProgress() ([]models.PathProgress, error) {
	var rows []pathProgressRow
	err := h.db.Raw(`
		SELECT p.id, p.title,
		       COUNT(n.id) AS total,
		       COUNT(n.id) FILTER (WHERE n.status = ?) AS mastered
		FROM paths p
		LEFT JOIN nodes n ON n.path_id = p.id AND n.type = ?
		GROUP BY p.id, p.title
		ORDER BY p.created_at DESC
	`, models.StatusMastered, models.NodeTypeTopic).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	progress := make([]models.PathProgress, len(rows))
	for i, row := range rows {
		progress[i] = models.PathProgress{
			Path:     models.Path{ID: row.ID, Title: row.Title},
			Total:    row.Total,
			Mastered: row.Mastered,
		}
	}
	return progress, nil
}
