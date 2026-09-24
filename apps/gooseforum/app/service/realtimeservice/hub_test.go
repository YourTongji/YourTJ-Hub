package realtimeservice

import (
	"testing"
	"time"
)

func TestHubRoutesOnlyToOwnerAndAllOwnerConnections(t *testing.T) {
	hub := NewHub(2, 3, 2)
	a, err := hub.Subscribe(11)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := hub.Subscribe(11)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	other, err := hub.Subscribe(12)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	if _, err := hub.Subscribe(11); err == nil {
		t.Fatal("accepted a third owner connection")
	}

	hub.Publish(11, Event{Type: "chat.changed", ConvID: 7, Change: "created"})
	for _, sub := range []*Subscription{a, b} {
		select {
		case event := <-sub.Events():
			if event.ConvID != 7 || event.Type != "chat.changed" {
				t.Fatalf("wrong event: %+v", event)
			}
		case <-time.After(time.Second):
			t.Fatal("owner connection missed event")
		}
	}
	select {
	case event := <-other.Events():
		t.Fatalf("leaked event to another user: %+v", event)
	default:
	}
}

func TestHubOverflowDisconnectsAndCloseAllUnblocks(t *testing.T) {
	hub := NewHub(2, 2, 1)
	sub, err := hub.Subscribe(11)
	if err != nil {
		t.Fatal(err)
	}
	hub.Publish(11, Event{Type: "notifications.changed"})
	hub.Publish(11, Event{Type: "notifications.changed"})
	select {
	case <-sub.Done():
	case <-time.After(time.Second):
		t.Fatal("overflow did not disconnect slow consumer")
	}
	newSub, err := hub.Subscribe(11)
	if err != nil {
		t.Fatal(err)
	}
	hub.CloseAll()
	select {
	case <-newSub.Done():
	case <-time.After(time.Second):
		t.Fatal("shutdown did not close stream")
	}
}
