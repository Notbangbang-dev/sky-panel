package agentclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/Notbangbang-dev/sky-panel/node-agent/internal/protocol"
	"github.com/Notbangbang-dev/sky-panel/node-agent/internal/runtime"
)

var testUpgrader = websocket.Upgrader{}

func TestSessionSendsHelloAndHeartbeat(t *testing.T) {
	helloReceived := make(chan protocol.HelloPayload, 1)
	heartbeatReceived := make(chan protocol.HeartbeatPayload, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("server upgrade: %v", err)
			return
		}
		defer conn.Close()

		for i := 0; i < 2; i++ {
			var env protocol.Envelope
			if err := conn.ReadJSON(&env); err != nil {
				return
			}
			switch env.Type {
			case protocol.TypeHello:
				var hello protocol.HelloPayload
				json.Unmarshal(env.Payload, &hello)
				helloReceived <- hello
			case protocol.TypeHeartbeat:
				var hb protocol.HeartbeatPayload
				json.Unmarshal(env.Payload, &hb)
				heartbeatReceived <- hb
			}
		}
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	dispatch := NewDispatcher(runtime.NewFake())
	session := NewSession(conn, "test-node-token", dispatch, 20*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go session.Run(ctx)

	select {
	case hello := <-helloReceived:
		if hello.NodeToken != "test-node-token" {
			t.Errorf("expected node token 'test-node-token', got %q", hello.NodeToken)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for hello")
	}

	select {
	case <-heartbeatReceived:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for heartbeat")
	}
}

func TestSessionDispatchesCommandAndAcks(t *testing.T) {
	commandSent := make(chan struct{})
	ackReceived := make(chan protocol.AckPayload, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("server upgrade: %v", err)
			return
		}
		defer conn.Close()

		// First message from the client is always hello; drain it.
		var hello protocol.Envelope
		if err := conn.ReadJSON(&hello); err != nil {
			return
		}

		env, err := protocol.Encode(protocol.TypeCommand, protocol.CommandPayload{
			CommandID: "cmd-1",
			Action:    protocol.ActionCreate,
			ServerID:  "server-1",
			Spec:      &protocol.ContainerSpec{Image: "test-image"},
		})
		if err != nil {
			t.Errorf("encode command: %v", err)
			return
		}
		if err := conn.WriteJSON(env); err != nil {
			return
		}
		close(commandSent)

		var ackEnv protocol.Envelope
		if err := conn.ReadJSON(&ackEnv); err != nil {
			return
		}
		var ack protocol.AckPayload
		json.Unmarshal(ackEnv.Payload, &ack)
		ackReceived <- ack
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	dispatch := NewDispatcher(runtime.NewFake())
	session := NewSession(conn, "test-node-token", dispatch, time.Hour) // heartbeat won't fire during this test

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go session.Run(ctx)

	select {
	case <-commandSent:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for command to be sent")
	}

	select {
	case ack := <-ackReceived:
		if !ack.OK || ack.CommandID != "cmd-1" {
			t.Errorf("unexpected ack: %+v", ack)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ack")
	}
}
