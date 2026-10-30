package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"vekr/internal/agent"
	"vekr/internal/markdown"
	"vekr/internal/models"
)

type ChatHandler struct {
	chat    *agent.ChatService
	distill *agent.DistillService
}

func NewChatHandler(chat *agent.ChatService, distill *agent.DistillService) *ChatHandler {
	return &ChatHandler{chat: chat, distill: distill}
}

func (h *ChatHandler) Create(w http.ResponseWriter, r *http.Request) {
	content := r.FormValue("content")

	var chat models.Chat
	if rawChatID := r.FormValue("chat_id"); rawChatID != "" {
		chatID, err := strconv.ParseUint(rawChatID, 10, 64)
		if err != nil {
			http.Error(w, "invalid chat_id", http.StatusBadRequest)
			return
		}
		chat, err = h.chat.FindChat(uint(chatID))
		if err != nil {
			http.NotFound(w, r)
			return
		}
	} else {
		nodeID, err := strconv.ParseUint(r.FormValue("node_id"), 10, 64)
		if err != nil {
			http.Error(w, "chat_id or node_id is required", http.StatusBadRequest)
			return
		}
		if content == "" {
			http.Redirect(w, r, fmt.Sprintf("/notes/%d", nodeID), http.StatusSeeOther)
			return
		}
		chat, err = h.chat.StartNodeChat(uint(nodeID))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if content != "" {
		if _, err := h.chat.SendMessageInBackground(chat.ID, content); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, chatReturnURL(chat), http.StatusSeeOther)
}

func (h *ChatHandler) Distill(w http.ResponseWriter, r *http.Request) {
	chatID, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid chat id", http.StatusBadRequest)
		return
	}

	node, err := h.distill.Distill(r.Context(), uint(chatID))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/notes/%d", node.ID), http.StatusSeeOther)
}

func chatReturnURL(chat models.Chat) string {
	if chat.NodeID == nil {
		return "/"
	}
	return fmt.Sprintf("/notes/%d?chat_id=%d", *chat.NodeID, chat.ID)
}

func renderMessages(messages []models.ChatMessage) ([]string, error) {
	messageHTML := make([]string, len(messages))
	for i, m := range messages {
		html, err := markdown.ToHTML(m.Content)
		if err != nil {
			return nil, err
		}
		messageHTML[i] = html
	}
	return messageHTML, nil
}
