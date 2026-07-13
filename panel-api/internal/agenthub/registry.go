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

	// supportsPull is true when the node advertised the pull_image capability
	// in its hello, so the panel can safely send it that command.
	supportsPull bool
	// supportsDatabases is true when the node advertised the databases
	// capability (v0.5.0+ with MariaDB configured).
	supportsDatabases bool

	pendingMu sync.Mutex
	pending   map[string]chan AckPayload
}

func newConn(nodeID string, ws *websocket.Conn, secret []byte, capabilities []string) *Conn {
	c := &Conn{NodeID: nodeID, conn: ws, secret: secret, pending: make(map[string]chan AckPayload)}
	for _, cap := range capabilities {
		switch cap {
		case CapPullImage:
			c.supportsPull = true
		case CapDatabases:
			c.supportsDatabases = true
		}
	}
	return c
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
	// Bound the write so a stuck/slow node can't block this command goroutine
	// (and, via writeMu, every other command to the same node) indefinitely.
	_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
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
		// Non-blocking: the channel is buffered(1) and read at most once, so a
		// duplicate ack (or one racing failPending) must never wedge the read
		// loop on a full buffer.
		select {
		case ch <- ack:
		default:
		}
	}
}

func (c *Conn) Close() error {
	return c.conn.Close()
}

// failPending resolves every in-flight command with a failure ack. Called when
// the connection drops so a waiting SendCommand returns immediately instead of
// blocking until its (possibly long) context deadline — e.g. if a node dies
// mid image-pull.
func (c *Conn) failPending() {
	c.pendingMu.Lock()
	defer c.pendingMu.Unlock()
	for id, ch := range c.pending {
		select {
		case ch <- AckPayload{CommandID: id, OK: false, Error: "node disconnected"}:
		default:
		}
		delete(c.pending, id)
	}
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

// Close severs the live connection for a node, if any. Used when a node token
// is rotated or the node is deleted, so the old secret stops working
// immediately instead of remaining valid until the daemon happens to reconnect.
func (r *Registry) Close(nodeID string) {
	if c, ok := r.Get(nodeID); ok {
		_ = c.Close()
	}
}

// SupportsPullImage reports whether a connected node advertised the pull_image
// capability. False for offline nodes and for older daemons that predate it —
// callers use this to avoid sending a command an old daemon can't decode.
func (r *Registry) SupportsPullImage(nodeID string) bool {
	conn, ok := r.Get(nodeID)
	return ok && conn.supportsPull
}

// SupportsDatabases reports whether a connected node can provision databases
// (daemon v0.5.0+ with MariaDB configured). False for offline nodes and older
// daemons, so the panel can reject a create with a clear message.
func (r *Registry) SupportsDatabases(nodeID string) bool {
	conn, ok := r.Get(nodeID)
	return ok && conn.supportsDatabases
}

// ConnectedNodeIDs returns the IDs of every node with a live connection.
// Used to fan an image warm-up out to all online nodes.
func (r *Registry) ConnectedNodeIDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.conns))
	for id := range r.conns {
		ids = append(ids, id)
	}
	return ids
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
