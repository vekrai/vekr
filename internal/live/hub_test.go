package live

import "testing"

func TestHubDeliversToSubscribersOfThatStreamOnly(t *testing.T) {
	hub := NewHub()

	a, unsubA := hub.Subscribe("chat:1")
	defer unsubA()
	b, unsubB := hub.Subscribe("chat:2")
	defer unsubB()

	hub.Publish("chat:1", "hello")

	select {
	case got := <-a:
		if got != "hello" {
			t.Errorf("got %q, want %q", got, "hello")
		}
	default:
		t.Fatal("subscriber of chat:1 got nothing")
	}

	select {
	case got := <-b:
		t.Errorf("subscriber of chat:2 got %q, want nothing", got)
	default:
	}
}

func TestHubPublishNeverBlocksOnSlowSubscriber(t *testing.T) {
	hub := NewHub()
	_, unsub := hub.Subscribe("chat:1")
	defer unsub()

	for i := 0; i < subscriberBuffer*3; i++ {
		hub.Publish("chat:1", "msg")
	}
}

func TestHubUnsubscribeClosesChannelAndIsIdempotent(t *testing.T) {
	hub := NewHub()
	ch, unsub := hub.Subscribe("chat:1")

	unsub()
	unsub()

	if _, open := <-ch; open {
		t.Error("channel still open after unsubscribe")
	}
	hub.Publish("chat:1", "after")
}
