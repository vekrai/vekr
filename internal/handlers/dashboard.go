package handlers

import (
	"net/http"

	"gorm.io/gorm"

	"vekr/internal/agent"
	"vekr/internal/markdown"
	"vekr/internal/models"
	"vekr/web/templates"
)

type DashboardHandler struct {
	db   *gorm.DB
	chat *agent.ChatService
}

func NewDashboardHandler(db *gorm.DB, chat *agent.ChatService) *DashboardHandler {
	return &DashboardHandler{db: db, chat: chat}
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

	pathProgress, err := h.loadPathProgress()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	chat, err := h.chat.CurrentChat()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	messages, err := h.chat.History(chat.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	messageHTML := make([]string, len(messages))
	for i, m := range messages {
		html, err := markdown.ToHTML(m.Content)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		messageHTML[i] = html
	}

	templates.Dashboard(recentNotes, nodeCount, pathProgress, chat, messages, messageHTML).Render(r.Context(), w)
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
