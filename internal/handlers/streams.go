package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"vekr/internal/agent"
	"vekr/internal/live"
)

const streamHeartbeat = 25 * time.Second

type StreamsHandler struct {
	hub  *live.Hub
	chat *agent.ChatService
}

func NewStreamsHandler(hub *live.Hub, chat *agent.ChatService) *StreamsHandler {
	return &StreamsHandler{hub: hub, chat: chat}
}

func (h *StreamsHandler) Chat(w http.ResponseWriter, r *http.Request) {
	chatID, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid chat id", http.StatusBadRequest)
		return
	}
	after, _ := strconv.ParseUint(r.URL.Query().Get("after"), 10, 64)

	messages, unsubscribe := h.hub.Subscribe(live.ChatStream(uint(chatID)))
	defer unsubscribe()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher := http.NewResponseController(w)
	if err := flusher.Flush(); err != nil {
		return
	}

	if !h.chat.Busy(uint(chatID)) {
		latest, err := h.chat.LatestMessageID(uint(chatID))
		if err == nil && uint64(latest) > after {
			writeEvent(w, live.RefreshAction())
			flusher.Flush()
		}
	}

	heartbeat := time.NewTicker(streamHeartbeat)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			if _, err := io.WriteString(w, ": ping\n\n"); err != nil {
				return
			}
		case message, open := <-messages:
			if !open {
				return
			}
			if err := writeEvent(w, message); err != nil {
				return
			}
		}
		if err := flusher.Flush(); err != nil {
			return
		}
	}
}

func writeEvent(w io.Writer, data string) error {
	var sb strings.Builder
	for _, line := range strings.Split(data, "\n") {
		fmt.Fprintf(&sb, "data: %s\n", line)
	}
	sb.WriteString("\n")
	_, err := io.WriteString(w, sb.String())
	return err
}
