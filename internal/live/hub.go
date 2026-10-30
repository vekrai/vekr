package live

import (
	"fmt"
	"sync"
)

const subscriberBuffer = 16

type Hub struct {
	mu          sync.Mutex
	subscribers map[string]map[chan string]struct{}
}

func NewHub() *Hub {
	return &Hub{subscribers: make(map[string]map[chan string]struct{})}
}

func (h *Hub) Subscribe(stream string) (<-chan string, func()) {
	ch := make(chan string, subscriberBuffer)

	h.mu.Lock()
	if h.subscribers[stream] == nil {
		h.subscribers[stream] = make(map[chan string]struct{})
	}
	h.subscribers[stream][ch] = struct{}{}
	h.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			h.mu.Lock()
			delete(h.subscribers[stream], ch)
			if len(h.subscribers[stream]) == 0 {
				delete(h.subscribers, stream)
			}
			h.mu.Unlock()
			close(ch)
		})
	}
	return ch, unsubscribe
}

func (h *Hub) Publish(stream, message string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subscribers[stream] {
		select {
		case ch <- message:
		default:
		}
	}
}

func ChatStream(chatID uint) string {
	return fmt.Sprintf("chat:%d", chatID)
}

func RefreshAction() string {
	return `<turbo-stream action="refresh"></turbo-stream>`
}
