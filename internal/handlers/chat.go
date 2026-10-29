package handlers

import (
	"net/http"
	"strconv"

	"vekr/internal/agent"
)

type ChatHandler struct {
	chat *agent.ChatService
}

func NewChatHandler(chat *agent.ChatService) *ChatHandler {
	return &ChatHandler{chat: chat}
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
