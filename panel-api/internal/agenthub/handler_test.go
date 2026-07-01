package agenthub

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/auth"
)

type nodeAuthEntry struct {
	nodeID    string
	token     string
	expiresAt time.Time
}

type fakeNodeLookup struct {
	byHash map[string]nodeAuthEntry
}

func (f *fakeNodeLookup) AuthenticateNode(tokenHash string) (string, string, time.Time, error) {
	e, ok := f.byHash[tokenHash]
	if !ok {
		return "", "", time.Time{}, errors.New("unknown token")
	}
	return e.nodeID, e.token, e.expiresAt, nil
}

type fakeSink struct {
	mu   sync.Mutex
	msgs map[string][][]byte
}

func newFakeSink() *fakeSink {
	return &fakeSink{msgs: make(map[string][][]byte)}
}

func (f *fakeSink) Broadcast(topic string, msg []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.msgs[topic] = append(f.msgs[topic], msg)
}

func (f *fakeSink) get(topic string) [][]byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.msgs[topic]
}

func dialTestServer(t *testing.T, srv *httptest.Server) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	return conn
}

func goodLookup() (*fakeNodeLookup, string) {
	const token = "good-token"
	return &fakeNodeLookup{byHash: map[string]nodeAuthEntry{
		hashOf(token): {nodeID: "node-1", token: token, expiresAt: time.Now().Add(time.Hour)},
	}}, token
}

func TestHandlerRejectsBadHello(t *testing.T) {
	lookup := &fakeNodeLookup{byHash: map[string]nodeAuthEntry{}}
	h := NewHandler(NewRegistry(), lookup, newFakeSink())

	srv := httptest.NewServer(http.HandlerFunc(h.ServeWS))
	defer srv.Close()

	conn := dialTestServer(t, srv)
	defer conn.Close()

	env, err := EncodeSigned([]byte("not-a-real-token"), TypeHello, HelloPayload{NodeToken: "not-a-real-token"})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if err := conn.WriteJSON(env); err != nil {
		t.Fatalf("write: %v", err)
	}

	// The server should close the connection rather than register it.
	conn.SetReadDeadline(time.Now().Add(time.Second))
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Error("expected connection to be closed after a bad hello")
	}
}

func TestHandlerRejectsExpiredToken(t *testing.T) {
	const token = "expired-token"
	lookup := &fakeNodeLookup{byHash: map[string]nodeAuthEntry{
		hashOf(token): {nodeID: "node-1", token: token, expiresAt: time.Now().Add(-time.Hour)},
	}}
	h := NewHandler(NewRegistry(), lookup, newFakeSink())

	srv := httptest.NewServer(http.HandlerFunc(h.ServeWS))
	defer srv.Close()

	conn := dialTestServer(t, srv)
	defer conn.Close()
	sendHello(t, conn, token)

	conn.SetReadDeadline(time.Now().Add(time.Second))
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Error("expected connection to be closed after an expired-token hello")
	}
}

func TestHandlerRegistersNodeAndDispatchesCommand(t *testing.T) {
	lookup, token := goodLookup()
	registry := NewRegistry()
	h := NewHandler(registry, lookup, newFakeSink())

	srv := httptest.NewServer(http.HandlerFunc(h.ServeWS))
	defer srv.Close()

	conn := dialTestServer(t, srv)
	defer conn.Close()

	sendHello(t, conn, token)

	// Wait for registration by polling (avoids sleeping a fixed amount).
	waitForCondition(t, func() bool {
		_, ok := registry.Get("node-1")
		return ok
	})

	// Simulate a command sent from the panel side and the node acking it.
	done := make(chan AckPayload, 1)
	go func() {
		ack, err := registry.SendCommand("node-1", CommandPayload{CommandID: "cmd-1", Action: ActionStart, ServerID: "server-1"})
		if err != nil {
			t.Errorf("SendCommand: %v", err)
			return
		}
		done <- ack
	}()

	var env Envelope
	if err := conn.ReadJSON(&env); err != nil {
		t.Fatalf("read command: %v", err)
	}
	if env.Type != TypeCommand {
		t.Fatalf("expected command envelope, got %q", env.Type)
	}
	if !env.Verify([]byte(token)) {
		t.Fatal("expected the command envelope sent by the panel to verify against the node's token")
	}
	var cmd CommandPayload
	if err := json.Unmarshal(env.Payload, &cmd); err != nil {
		t.Fatalf("unmarshal command: %v", err)
	}

	ackEnv, err := EncodeSigned([]byte(token), TypeAck, AckPayload{CommandID: cmd.CommandID, OK: true})
	if err != nil {
		t.Fatalf("encode ack: %v", err)
	}
	if err := conn.WriteJSON(ackEnv); err != nil {
		t.Fatalf("write ack: %v", err)
	}

	select {
	case ack := <-done:
		if !ack.OK {
			t.Errorf("expected ok ack, got %+v", ack)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SendCommand to resolve")
	}
}

func TestHandlerRejectsAckSignedWithWrongSecret(t *testing.T) {
	lookup, token := goodLookup()
	registry := NewRegistry()
	h := NewHandler(registry, lookup, newFakeSink())

	srv := httptest.NewServer(http.HandlerFunc(h.ServeWS))
	defer srv.Close()

	conn := dialTestServer(t, srv)
	defer conn.Close()
	sendHello(t, conn, token)

	waitForCondition(t, func() bool {
		_, ok := registry.Get("node-1")
		return ok
	})

	forged, err := EncodeSigned([]byte("wrong-secret"), TypeAck, AckPayload{CommandID: "whatever", OK: true})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if err := conn.WriteJSON(forged); err != nil {
		t.Fatalf("write: %v", err)
	}

	// The connection should be dropped rather than accepting the forged ack.
	conn.SetReadDeadline(time.Now().Add(time.Second))
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Error("expected connection to be closed after a forged envelope")
	}
}

func TestHandlerForwardsHeartbeatToSink(t *testing.T) {
	lookup, token := goodLookup()
	sink := newFakeSink()
	h := NewHandler(NewRegistry(), lookup, sink)

	srv := httptest.NewServer(http.HandlerFunc(h.ServeWS))
	defer srv.Close()

	conn := dialTestServer(t, srv)
	defer conn.Close()

	sendHello(t, conn, token)

	hbEnv, err := EncodeSigned([]byte(token), TypeHeartbeat, HeartbeatPayload{Containers: []ContainerHeartbeat{
		{ServerID: "server-1", Running: true, CPU: 5},
	}})
	if err != nil {
		t.Fatalf("encode heartbeat: %v", err)
	}
	if err := conn.WriteJSON(hbEnv); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}

	waitForCondition(t, func() bool {
		return len(sink.get("server:server-1:stats")) > 0
	})
}

func sendHello(t *testing.T, conn *websocket.Conn, token string) {
	t.Helper()
	env, err := EncodeSigned([]byte(token), TypeHello, HelloPayload{NodeToken: token})
	if err != nil {
		t.Fatalf("encode hello: %v", err)
	}
	if err := conn.WriteJSON(env); err != nil {
		t.Fatalf("write hello: %v", err)
	}
}

func waitForCondition(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for condition")
}

// hashOf lets tests seed fakeNodeLookup with the same hash the handler
// computes from an incoming hello's raw node token.
func hashOf(token string) string {
	return auth.HashToken(token)
}
