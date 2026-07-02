package agenthub

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestSendCommandTimesOutWithoutAck(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		// Read and discard the command, but never ack it.
		var env Envelope
		conn.ReadJSON(&env)
		time.Sleep(2 * time.Second)
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer ws.Close()

	conn := newConn("node-1", ws, []byte("node-secret"), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	if _, err := conn.SendCommand(ctx, CommandPayload{CommandID: "cmd-1", Action: ActionStart}); err == nil {
		t.Error("expected SendCommand to time out when no ack arrives")
	}
}

func TestRegistryGetUnregistered(t *testing.T) {
	r := NewRegistry()
	if _, ok := r.Get("nonexistent"); ok {
		t.Error("expected Get to return false for an unregistered node")
	}
}

func TestRegistrySendCommandOffline(t *testing.T) {
	r := NewRegistry()
	if _, err := r.SendCommand("nonexistent", CommandPayload{CommandID: "1"}); err != ErrNodeOffline {
		t.Errorf("expected ErrNodeOffline, got %v", err)
	}
}
