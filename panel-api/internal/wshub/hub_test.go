package wshub

import "testing"

type fakeSubscriber struct {
	received [][]byte
}

func (f *fakeSubscriber) Send(msg []byte) {
	f.received = append(f.received, msg)
}

func TestBroadcastDeliversToSubscribers(t *testing.T) {
	hub := NewHub()
	a := &fakeSubscriber{}
	b := &fakeSubscriber{}

	hub.Subscribe("topic-1", a)
	hub.Subscribe("topic-1", b)
	hub.Subscribe("topic-2", b)

	hub.Broadcast("topic-1", []byte("hello"))

	if len(a.received) != 1 || string(a.received[0]) != "hello" {
		t.Errorf("subscriber a should have received one message, got %v", a.received)
	}
	if len(b.received) != 1 || string(b.received[0]) != "hello" {
		t.Errorf("subscriber b should have received one message, got %v", b.received)
	}

	hub.Broadcast("topic-2", []byte("only-b"))
	if len(a.received) != 1 {
		t.Errorf("subscriber a should not receive topic-2 messages")
	}
	if len(b.received) != 2 {
		t.Errorf("subscriber b should have received the topic-2 message too")
	}
}

func TestUnsubscribeStopsDelivery(t *testing.T) {
	hub := NewHub()
	a := &fakeSubscriber{}

	hub.Subscribe("topic", a)
	hub.Unsubscribe("topic", a)
	hub.Broadcast("topic", []byte("should not arrive"))

	if len(a.received) != 0 {
		t.Errorf("expected no messages after unsubscribe, got %v", a.received)
	}
}

func TestUnsubscribeAllRemovesFromEveryTopic(t *testing.T) {
	hub := NewHub()
	a := &fakeSubscriber{}

	hub.Subscribe("topic-1", a)
	hub.Subscribe("topic-2", a)
	hub.UnsubscribeAll(a)

	if got := hub.SubscriberCount("topic-1"); got != 0 {
		t.Errorf("expected 0 subscribers on topic-1, got %d", got)
	}
	if got := hub.SubscriberCount("topic-2"); got != 0 {
		t.Errorf("expected 0 subscribers on topic-2, got %d", got)
	}
}
