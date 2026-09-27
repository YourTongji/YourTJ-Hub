// Package realtimeservice delivers small, owner-scoped invalidation hints to
// foreground clients. REST remains the source of truth after every reconnect.
package realtimeservice

import (
	"errors"
	"sync"
)

const (
	EventChatChanged          = "chat.changed"
	EventNotificationsChanged = "notifications.changed"
	EventUnreadChanged        = "unread.changed"
)

var ErrConnectionLimit = errors.New("realtime connection limit reached")
var ErrHubClosed = errors.New("realtime hub is shutting down")

// Event deliberately carries no private message body or notification preview.
type Event struct {
	Type   string
	ConvID uint64
	Change string
}

type Hub struct {
	mu        sync.Mutex
	byUser    map[uint64]map[*Subscription]struct{}
	count     int
	perUser   int
	total     int
	queueSize int
	closed    bool
}

type Subscription struct {
	hub    *Hub
	userID uint64
	events chan Event
	done   chan struct{}
}

func NewHub(perUser, total, queueSize int) *Hub {
	if perUser < 1 || total < 1 || queueSize < 1 {
		panic("realtime hub limits must be positive")
	}
	return &Hub{byUser: make(map[uint64]map[*Subscription]struct{}), perUser: perUser, total: total, queueSize: queueSize}
}

// DefaultHub is process-local. A multi-instance deployment needs a shared
// invalidation bus; sticky sessions alone do not make this reliable.
var DefaultHub = NewHub(5, 10000, 64)

func (h *Hub) Subscribe(userID uint64) (*Subscription, error) {
	if userID == 0 {
		return nil, ErrConnectionLimit
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return nil, ErrHubClosed
	}
	if h.count >= h.total || len(h.byUser[userID]) >= h.perUser {
		return nil, ErrConnectionLimit
	}
	sub := &Subscription{hub: h, userID: userID, events: make(chan Event, h.queueSize), done: make(chan struct{})}
	if h.byUser[userID] == nil {
		h.byUser[userID] = make(map[*Subscription]struct{})
	}
	h.byUser[userID][sub] = struct{}{}
	h.count++
	return sub, nil
}

func (h *Hub) Publish(userID uint64, event Event) {
	if userID == 0 {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for sub := range h.byUser[userID] {
		select {
		case sub.events <- event:
		default:
			// A silent drop would leave a healthy-looking stream with stale state.
			// Disconnect so the client performs its full REST reconciliation.
			h.removeLocked(sub)
		}
	}
}

func (h *Hub) removeLocked(sub *Subscription) {
	if _, ok := h.byUser[sub.userID][sub]; !ok {
		return
	}
	delete(h.byUser[sub.userID], sub)
	if len(h.byUser[sub.userID]) == 0 {
		delete(h.byUser, sub.userID)
	}
	h.count--
	close(sub.done)
}

// CloseAll rejects new subscriptions and lets existing HTTP handlers finish
// before http.Server.Shutdown. It is terminal for this Hub.
func (h *Hub) CloseAll() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.closed = true
	for _, connections := range h.byUser {
		for sub := range connections {
			h.removeLocked(sub)
		}
	}
}

func (s *Subscription) Events() <-chan Event  { return s.events }
func (s *Subscription) Done() <-chan struct{} { return s.done }

func (s *Subscription) Close() {
	s.hub.mu.Lock()
	defer s.hub.mu.Unlock()
	s.hub.removeLocked(s)
}

func PublishChatChanged(userID, convID uint64, change string) {
	DefaultHub.Publish(userID, Event{Type: EventChatChanged, ConvID: convID, Change: change})
	DefaultHub.Publish(userID, Event{Type: EventUnreadChanged})
}

func PublishNotificationsChanged(userID uint64, change string) {
	DefaultHub.Publish(userID, Event{Type: EventNotificationsChanged, Change: change})
	DefaultHub.Publish(userID, Event{Type: EventUnreadChanged})
}
