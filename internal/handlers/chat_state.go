package handlers

import (
	"net/http"
	"strconv"
	"sync"

	"vekr/internal/agent"
	"vekr/web/templates"
)

func ChatState(chat *agent.ChatService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			load := sync.OnceValues(func() (templates.ChatState, error) {
				return loadChatState(chat, r)
			})
			next.ServeHTTP(w, r.WithContext(templates.WithChatLoader(r.Context(), load)))
		})
	}
}

func loadChatState(chat *agent.ChatService, r *http.Request) (templates.ChatState, error) {
	state := templates.ChatState{PagePath: pagePath(r)}
	query := r.URL.Query()

	if chatID, err := strconv.ParseUint(query.Get("chat_id"), 10, 64); err == nil && chatID != 0 {
		found, err := chat.FindChat(uint(chatID))
		if err == nil {
			state.Chat = found
			state.Open = true
			state.Busy = chat.Busy(found.ID)
			if state.Messages, err = chat.History(found.ID); err != nil {
				return state, err
			}
			if state.MessageHTML, err = renderMessages(state.Messages); err != nil {
				return state, err
			}
			state.ContextWindow = chat.ContextWindow()
			for _, m := range state.Messages {
				if m.TotalTokens > 0 {
					state.TokensUsed = m.TotalTokens
				}
			}
			return state, nil
		}
	}

	state.Open = query.Get("chat") == "new"
	return state, nil
}

func pagePath(r *http.Request) string {
	u := *r.URL
	query := u.Query()
	query.Del("chat_id")
	query.Del("chat")
	u.RawQuery = query.Encode()
	u.Scheme, u.Host, u.User, u.Fragment = "", "", nil, ""
	return u.RequestURI()
}
