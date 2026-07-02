package agenthub

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var ErrNodeOffline = fmt.Errorf("node is not connected")

// Conn wraps one live sky-daemon connection: sending commands and matching
// their async acks back up by command ID.
type Conn struct {
	NodeID string

	conn    *websocket.Conn
	secret  []byte
	writeMu sync.Mutex

	pendingMu sync.Mutex
	pending   map[string]chan AckPayload
}

func newConn(nodeID string, ws *websocket.Conn, secret []byte) *Conn {
	return &Conn{NodeID: nodeID, conn: ws, secret: secret, pending: make(map[string]chan AckPayload)}
}

// SendCommand writes cmd to the node and blocks until the matching ack
// arrives or ctx is done.
func (c *Conn) SendCommand(ctx context.Context, cmd CommandPayload) (AckPayload, error) {
	ch := make(chan AckPayload, 1)

	c.pendingMu.Lock()
	c.pending[cmd.CommandID] = ch
	c.pendingMu.Unlock()

	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, cmd.CommandID)
		c.pendingMu.Unlock()
	}()

	env, err := EncodeSigned(c.secret, TypeCommand, cmd)
	if err != nil {
		return AckPayload{}, err
	}

	c.writeMu.Lock()
	err = c.conn.WriteJSON(env)
	c.writeMu.Unlock()
	if err != nil {
		return AckPayload{}, fmt.Errorf("send command: %w", err)
	}

	select {
	case ack := <-ch:
		return ack, nil
	case <-ctx.Done():
		return AckPayload{}, ctx.Err()
	}
}

// resolveAck delivers an incoming ack to whichever SendCommand call is
// waiting for it, if any (acks for unknown/expired command IDs are dropped).
func (c *Conn) resolveAck(ack AckPayload) {
	c.pendingMu.Lock()
	ch, ok := c.pending[ack.CommandID]
	c.pendingMu.Unlock()
	if ok {
		ch <- ack
	}
}

func (c *Conn) Close() error {
	return c.conn.Close()
}

// Registry tracks the one live connection per online node.
type Registry struct {
	mu    sync.RWMutex
	conns map[string]*Conn
}

func NewRegistry() *Registry {
	return &Registry{conns: make(map[string]*Conn)}
}

func (r *Registry) register(nodeID string, c *Conn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.conns[nodeID] = c
}

func (r *Registry) unregister(nodeID string, c *Conn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Only remove if it's still the same connection (avoids a race where a
	// reconnect has already replaced it).
	if r.conns[nodeID] == c {
		delete(r.conns, nodeID)
	}
}

func (r *Registry) Get(nodeID string) (*Conn, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.conns[nodeID]
	return c, ok
}

// Connected reports whether nodeID currently has a live connection.
func (r *Registry) Connected(nodeID string) bool {
	_, ok := r.Get(nodeID)
	return ok
}

// SendCommand is a convenience wrapper for the common case of looking the
// node up and sending in one call, with a sensible default timeout.
func (r *Registry) SendCommand(nodeID string, cmd CommandPayload) (AckPayload, error) {
	return r.SendCommandTimeout(nodeID, cmd, 15*time.Second)
}

// SendCommandTimeout is SendCommand with a caller-chosen ack deadline. Used for
// operations that can legitimately take a long time on the node — notably
// container creation, which may pull a multi-hundred-MB image on first use.
func (r *Registry) SendCommandTimeout(nodeID string, cmd CommandPayload, timeout time.Duration) (AckPayload, error) {
	conn, ok := r.Get(nodeID)
	if !ok {
		return AckPayload{}, ErrNodeOffline
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return conn.SendCommand(ctx, cmd)
}
