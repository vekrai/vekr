package templates

import (
	"context"
	"log"

	"vekr/internal/models"
)

type ChatState struct {
	Chat        models.Chat
	Open        bool
	Messages    []models.ChatMessage
	MessageHTML []string
	Busy        bool
	PagePath    string
}

type chatLoaderKey struct{}

func WithChatLoader(ctx context.Context, load func() (ChatState, error)) context.Context {
	return context.WithValue(ctx, chatLoaderKey{}, load)
}

func chatState(ctx context.Context) ChatState {
	load, ok := ctx.Value(chatLoaderKey{}).(func() (ChatState, error))
	if !ok {
		return ChatState{PagePath: "/"}
	}
	state, err := load()
	if err != nil {
		log.Printf("chat: load state: %v", err)
	}
	if state.PagePath == "" {
		state.PagePath = "/"
	}
	return state
}
