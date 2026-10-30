package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"gorm.io/gorm"

	"vekr/internal/agent"
	"vekr/internal/markdown"
	"vekr/internal/models"
	"vekr/web/templates"
)

type NotesHandler struct {
	db         *gorm.DB
	curriculum *agent.CurriculumService
	chat       *agent.ChatService
}

func NewNotesHandler(db *gorm.DB, curriculum *agent.CurriculumService, chat *agent.ChatService) *NotesHandler {
	return &NotesHandler{db: db, curriculum: curriculum, chat: chat}
}

func (h *NotesHandler) Index(w http.ResponseWriter, r *http.Request) {
	var nodes []models.Node
	if err := h.db.Order("created_at desc").Find(&nodes).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	templates.NotesIndex(nodes).Render(r.Context(), w)
}

func (h *NotesHandler) Show(w http.ResponseWriter, r *http.Request) {
	var node models.Node
	if err := h.db.First(&node, r.PathValue("id")).Error; err != nil {
		http.NotFound(w, r)
		return
	}

	contentHTML, err := markdown.ToHTML(node.Content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var chat models.Chat
	var messages []models.ChatMessage
	if rawChatID := r.URL.Query().Get("chat_id"); rawChatID != "" {
		chatID, err := strconv.ParseUint(rawChatID, 10, 64)
		if err != nil {
			http.Error(w, "invalid chat_id", http.StatusBadRequest)
			return
		}
		found, err := h.chat.FindChat(uint(chatID))
		if err == nil && found.NodeID != nil && *found.NodeID == node.ID {
			chat = found
			messages, err = h.chat.History(chat.ID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	messageHTML, err := renderMessages(messages)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	templates.NotesShow(node, contentHTML, chat, messages, messageHTML).Render(r.Context(), w)
}

func (h *NotesHandler) Create(w http.ResponseWriter, r *http.Request) {
	node := models.Node{
		Title:   r.FormValue("title"),
		Content: r.FormValue("content"),
	}
	if err := h.db.Create(&node).Error; err != nil {
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

	if _, err := h.curriculum.PromoteToTopic(r.Context(), uint(nodeID)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/notes/%d", nodeID), http.StatusSeeOther)
}
