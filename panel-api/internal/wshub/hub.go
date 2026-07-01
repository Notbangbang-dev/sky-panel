// Package wshub implements a small topic pub/sub hub used to fan real-time
// events (server console lines, stats samples, coin ticks, admin broadcasts)
// out to connected browser WebSocket clients.
package wshub

import "sync"

// Subscriber is anything that can receive a message. Kept as an interface
// (rather than depending on *Client directly) so the broadcast logic can be
// unit tested without a real WebSocket connection.
type Subscriber interface {
	Send(msg []byte)
}

type Hub struct {
	mu     sync.RWMutex
	topics map[string]map[Subscriber]struct{}
}

func NewHub() *Hub {
	return &Hub{topics: make(map[string]map[Subscriber]struct{})}
}

func (h *Hub) Subscribe(topic string, sub Subscriber) {
	h.mu.Lock()
	defer h.mu.Unlock()

	subs, ok := h.topics[topic]
	if !ok {
		subs = make(map[Subscriber]struct{})
		h.topics[topic] = subs
	}
	subs[sub] = struct{}{}
}

func (h *Hub) Unsubscribe(topic string, sub Subscriber) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if subs, ok := h.topics[topic]; ok {
		delete(subs, sub)
		if len(subs) == 0 {
			delete(h.topics, topic)
		}
	}
}

// UnsubscribeAll removes sub from every topic. Call this once when a
// connection closes.
func (h *Hub) UnsubscribeAll(sub Subscriber) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for topic, subs := range h.topics {
		delete(subs, sub)
		if len(subs) == 0 {
			delete(h.topics, topic)
		}
	}
}

func (h *Hub) Broadcast(topic string, msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for sub := range h.topics[topic] {
		sub.Send(msg)
	}
}

func (h *Hub) SubscriberCount(topic string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.topics[topic])
}
