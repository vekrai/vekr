package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"vekr/internal/agent"
	"vekr/internal/markdown"
	"vekr/internal/models"
	"vekr/web/templates"
)

const (
	noteSearchLimit  = 50
	noteKeywordLimit = 8
)

type NotesHandler struct {
	db         *gorm.DB
	curriculum *agent.CurriculumService
}

func NewNotesHandler(db *gorm.DB, curriculum *agent.CurriculumService) *NotesHandler {
	return &NotesHandler{db: db, curriculum: curriculum}
}

func (h *NotesHandler) Index(w http.ResponseWriter, r *http.Request) {
	view := templates.NotesIndexView{Query: strings.TrimSpace(r.URL.Query().Get("q")), TopicTitles: map[uint]string{}}

	if view.Query != "" {
		hits, err := models.SearchNotes(h.db, view.Query, noteSearchLimit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		view.Hits = hits
	} else if err := h.db.Order("created_at desc").Find(&view.Nodes).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var topics []models.Topic
	if err := h.db.Select("id", "title").Find(&topics).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, t := range topics {
		view.TopicTitles[t.ID] = t.Title
	}

	templates.NotesIndex(view).Render(r.Context(), w)
}

func (h *NotesHandler) Show(w http.ResponseWriter, r *http.Request) {
	var view templates.NoteView
	if err := h.db.First(&view.Node, r.PathValue("id")).Error; err != nil {
		http.NotFound(w, r)
		return
	}
	nodeID := view.Node.ID

	var err error
	if view.ContentHTML, err = markdown.ToHTML(view.Node.Content); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if view.Node.TopicID != nil {
		var found models.Topic
		if err := h.db.First(&found, *view.Node.TopicID).Error; err == nil {
			view.Topic = &found
		}
	}

	err = h.db.Raw(`
		SELECT word
		FROM ts_stat(format('SELECT to_tsvector(''simple'', title || '' '' || content) FROM nodes WHERE id = %s', ?::bigint))
		WHERE length(word) > 3
		  AND word !~ '^[0-9]+$'
		  AND numnode(plainto_tsquery('english', word)) > 0
		ORDER BY nentry DESC, word
		LIMIT ?
	`, nodeID, noteKeywordLimit).Scan(&view.Keywords).Error
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = h.db.Raw(`
		SELECT n.id AS node_id, n.title, l.relation, true AS outgoing
		FROM node_links l JOIN nodes n ON n.id = l.to_node_id
		WHERE l.from_node_id = ?
		UNION ALL
		SELECT n.id AS node_id, n.title, l.relation, false AS outgoing
		FROM node_links l JOIN nodes n ON n.id = l.from_node_id
		WHERE l.to_node_id = ?
		ORDER BY outgoing, title
	`, nodeID, nodeID).Scan(&view.Related).Error
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = h.db.Raw(`
		SELECT c.id AS chat_id,
		       (SELECT m.content FROM chat_messages m WHERE m.chat_id = c.id AND m.role = 'user' ORDER BY m.id LIMIT 1) AS title,
		       MAX(cm.created_at) AS last_at,
		       c.node_id
		FROM chats c
		JOIN chat_messages cm ON cm.chat_id = c.id
		WHERE c.node_id = ?
		GROUP BY c.id
		ORDER BY last_at DESC
	`, nodeID).Scan(&view.Chats).Error
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	templates.NotesShow(view).Render(r.Context(), w)
}

func (h *NotesHandler) Create(w http.ResponseWriter, r *http.Request) {
	node := models.Node{
		Title:   strings.TrimSpace(r.FormValue("title")),
		Content: r.FormValue("content"),
	}
	if node.Title == "" {
		http.Error(w, "title is required", http.StatusUnprocessableEntity)
		return
	}
	if err := h.db.Create(&node).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/notes/%d", node.ID), http.StatusSeeOther)
}

func (h *NotesHandler) Edit(w http.ResponseWriter, r *http.Request) {
	var node models.Node
	if err := h.db.First(&node, r.PathValue("id")).Error; err != nil {
		http.NotFound(w, r)
		return
	}
	var topic *models.Topic
	if node.TopicID != nil {
		var found models.Topic
		if err := h.db.First(&found, *node.TopicID).Error; err == nil {
			topic = &found
		}
	}
	templates.NotesEdit(node, topic).Render(r.Context(), w)
}

func (h *NotesHandler) Update(w http.ResponseWriter, r *http.Request) {
	var node models.Node
	if err := h.db.First(&node, r.PathValue("id")).Error; err != nil {
		http.NotFound(w, r)
		return
	}
	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		http.Error(w, "title is required", http.StatusUnprocessableEntity)
		return
	}
	if err := h.db.Model(&node).Updates(map[string]any{"title": title, "content": r.FormValue("content")}).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/notes/%d", node.ID), http.StatusSeeOther)
}

func (h *NotesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	nodeID, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid note id", http.StatusBadRequest)
		return
	}

	err = h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("from_node_id = ? OR to_node_id = ?", nodeID, nodeID).Delete(&models.NodeLink{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.Chat{}).Where("node_id = ?", nodeID).Update("node_id", nil).Error; err != nil {
			return err
		}
		return tx.Delete(&models.Node{}, nodeID).Error
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/notes", http.StatusSeeOther)
}

func (h *NotesHandler) Promote(w http.ResponseWriter, r *http.Request) {
	nodeID, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid note id", http.StatusBadRequest)
		return
	}

	topic, err := h.curriculum.PromoteToTopic(r.Context(), uint(nodeID))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/topics/%d", topic.ID), http.StatusSeeOther)
}
