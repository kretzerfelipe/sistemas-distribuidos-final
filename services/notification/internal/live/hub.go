package live

import (
	"encoding/json"
	"sync"
)

const subscriberBuffer = 64

type Update struct {
	Type     string          `json:"type"`
	MatchID  string          `json:"matchId"`
	Sequence int             `json:"sequence"`
	Payload  json.RawMessage `json:"payload"`
	Final    bool            `json:"-"`
}

type Subscription struct {
	Updates <-chan Update
	updates chan Update
	matchID string
	hub     *Hub
	closed  bool
}

type Hub struct {
	mutex       sync.Mutex
	subscribers map[string]map[*Subscription]struct{}
}

func NewHub() *Hub {
	return &Hub{subscribers: make(map[string]map[*Subscription]struct{})}
}

func (h *Hub) Subscribe(matchID string) *Subscription {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	updates := make(chan Update, subscriberBuffer)
	subscription := &Subscription{Updates: updates, updates: updates, matchID: matchID, hub: h}
	if h.subscribers[matchID] == nil {
		h.subscribers[matchID] = make(map[*Subscription]struct{})
	}
	h.subscribers[matchID][subscription] = struct{}{}
	return subscription
}

func (h *Hub) Publish(update Update) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	for subscription := range h.subscribers[update.MatchID] {
		select {
		case subscription.updates <- update:
		default:
			h.dropLocked(subscription)
		}
	}

	if update.Final {
		for subscription := range h.subscribers[update.MatchID] {
			h.dropLocked(subscription)
		}
	}
}

func (h *Hub) SubscriberCount(matchID string) int {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	return len(h.subscribers[matchID])
}

func (s *Subscription) Close() {
	s.hub.mutex.Lock()
	defer s.hub.mutex.Unlock()
	s.hub.dropLocked(s)
}

func (h *Hub) dropLocked(subscription *Subscription) {
	if subscription.closed {
		return
	}
	subscription.closed = true
	close(subscription.updates)

	delete(h.subscribers[subscription.matchID], subscription)
	if len(h.subscribers[subscription.matchID]) == 0 {
		delete(h.subscribers, subscription.matchID)
	}
}
