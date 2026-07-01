package agentclient

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/Notbangbang-dev/sky-panel/node-agent/internal/protocol"
)

// AgentVersion is stamped into the hello message so panel-api can log which
// build of node-agent is connecting.
const AgentVersion = "0.1.0"

// Session owns one live WebSocket connection to panel-api: sending hello,
// periodic heartbeats, and dispatching inbound commands. It is deliberately
// decoupled from dialing/reconnection (see Run in client.go) so it can be
// exercised in tests against an httptest WebSocket server without any real
// network reconnect logic involved.
type Session struct {
	conn      *websocket.Conn
	nodeToken string
	dispatch  *Dispatcher

	heartbeatInterval time.Duration

	writeMu sync.Mutex
}

func NewSession(conn *websocket.Conn, nodeToken string, dispatch *Dispatcher, heartbeatInterval time.Duration) *Session {
	return &Session{
		conn:              conn,
		nodeToken:         nodeToken,
		dispatch:          dispatch,
		heartbeatInterval: heartbeatInterval,
	}
}

// Run blocks until the connection closes, the context is cancelled, or a
// protocol error occurs.
func (s *Session) Run(ctx context.Context) error {
	if err := s.writeEnvelope(protocol.TypeHello, protocol.HelloPayload{
		NodeToken:    s.nodeToken,
		AgentVersion: AgentVersion,
	}); err != nil {
		return fmt.Errorf("send hello: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go s.heartbeatLoop(ctx)

	return s.readLoop(ctx)
}

func (s *Session) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(s.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			payload := s.dispatch.Heartbeat(ctx)
			if err := s.writeEnvelope(protocol.TypeHeartbeat, payload); err != nil {
				return
			}
		}
	}
}

func (s *Session) readLoop(ctx context.Context) error {
	for {
		var env protocol.Envelope
		if err := s.conn.ReadJSON(&env); err != nil {
			return fmt.Errorf("read message: %w", err)
		}

		if env.Type != protocol.TypeCommand {
			continue
		}

		var cmd protocol.CommandPayload
		if err := json.Unmarshal(env.Payload, &cmd); err != nil {
			continue
		}

		ack := s.dispatch.Handle(ctx, cmd)
		if err := s.writeEnvelope(protocol.TypeAck, ack); err != nil {
			return fmt.Errorf("send ack: %w", err)
		}
	}
}

func (s *Session) writeEnvelope(msgType string, payload any) error {
	env, err := protocol.Encode(msgType, payload)
	if err != nil {
		return err
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.conn.WriteJSON(env)
}
