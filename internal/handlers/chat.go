package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"vekr/internal/agent"
)

type ChatHandler struct {
	chat    *agent.ChatService
	distill *agent.DistillService
}

func NewChatHandler(chat *agent.ChatService, distill *agent.DistillService) *ChatHandler {
	return &ChatHandler{chat: chat, distill: distill}
}

func (h *ChatHandler) Create(w http.ResponseWriter, r *http.Request) {
	chatID, err := strconv.ParseUint(r.FormValue("chat_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid chat_id", http.StatusBadRequest)
		return
	}

	content := r.FormValue("content")
	if content == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if _, err := h.chat.SendMessage(r.Context(), uint(chatID), content); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
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
