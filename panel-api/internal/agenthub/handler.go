package agenthub

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/time/rate"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/auth"
)

var (
	errUnexpectedFirstMessage = errors.New("agenthub: expected hello as the first message")
	errNodeTokenExpired       = errors.New("agenthub: node token has expired")
)

// How long an incoming envelope's nonce is remembered for replay detection.
// Comfortably wider than MaxClockSkewSecs so a message can never both pass
// the freshness check and have its nonce cache entry expire before a
// replay would be caught.
const nonceCacheTTL = 120 * time.Second

// Per-connection inbound message budget: generous for heartbeats (every
// few seconds) plus occasional events/acks, but tight enough that a
// misbehaving or compromised node can't flood the panel.
const (
	rateLimitPerSecond = 20
	rateLimitBurst     = 40
)

// NodeLookup resolves a node's identity and secret from its hello token.
// It's an interface (rather than *repo.Nodes directly) so the handler can
// be unit tested without a database.
type NodeLookup interface {
	AuthenticateNode(tokenHash string) (nodeID, token string, expiresAt time.Time, err error)
}

// EventSink receives heartbeat/event traffic forwarded from a node, keyed by
// server ID, so it can be re-broadcast to subscribed browser clients. In
// production this is backed by wshub.Hub.
type EventSink interface {
	Broadcast(topic string, msg []byte)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type Handler struct {
	Registry *Registry
	Nodes    NodeLookup
	Sink     EventSink
}

func NewHandler(registry *Registry, nodes NodeLookup, sink EventSink) *Handler {
	return &Handler{Registry: registry, Nodes: nodes, Sink: sink}
}

// ServeWS accepts a node's inbound connection. The HTTP layer itself
// requires no auth (nodes dial in from arbitrary VPS IPs); identity is
// established by validating the first message's node token instead, and
// every message after that must carry a valid signature keyed by that
// node's secret (see Envelope.Verify).
func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer ws.Close()

	nodeID, secret, err := h.awaitHello(ws)
	if err != nil {
		log.Printf("agenthub: rejecting connection: %v", err)
		return
	}

	conn := newConn(nodeID, ws, secret)
	h.Registry.register(nodeID, conn)
	defer h.Registry.unregister(nodeID, conn)

	log.Printf("agenthub: node %s connected", nodeID)
	h.readLoop(conn, ws, secret)
	log.Printf("agenthub: node %s disconnected", nodeID)
}

func (h *Handler) awaitHello(ws *websocket.Conn) (nodeID string, secret []byte, err error) {
	var env Envelope
	if err := ws.ReadJSON(&env); err != nil {
		return "", nil, err
	}
	if env.Type != TypeHello {
		return "", nil, errUnexpectedFirstMessage
	}

	var hello HelloPayload
	if err := json.Unmarshal(env.Payload, &hello); err != nil {
		return "", nil, err
	}

	id, token, expiresAt, err := h.Nodes.AuthenticateNode(auth.HashToken(hello.NodeToken))
	if err != nil {
		return "", nil, err
	}
	if time.Now().After(expiresAt) {
		return "", nil, errNodeTokenExpired
	}

	return id, []byte(token), nil
}

func (h *Handler) readLoop(conn *Conn, ws *websocket.Conn, secret []byte) {
	nonces := newNonceCache(nonceCacheTTL)
	limiter := rate.NewLimiter(rate.Limit(rateLimitPerSecond), rateLimitBurst)

	for {
		var env Envelope
		if err := ws.ReadJSON(&env); err != nil {
			return
		}

		if !limiter.Allow() {
			log.Printf("agenthub: node %s exceeded its message rate limit, closing connection", conn.NodeID)
			return
		}
		if !env.Verify(secret) {
			log.Printf("agenthub: node %s sent an envelope with an invalid signature or stale timestamp (type=%s), closing connection", conn.NodeID, env.Type)
			return
		}
		if !nonces.checkAndRecord(env.Nonce) {
			log.Printf("agenthub: node %s replayed a nonce, closing connection", conn.NodeID)
			return
		}

		switch env.Type {
		case TypeAck:
			var ack AckPayload
			if json.Unmarshal(env.Payload, &ack) == nil {
				conn.resolveAck(ack)
			}

		case TypeHeartbeat:
			var hb HeartbeatPayload
			if json.Unmarshal(env.Payload, &hb) == nil {
				h.forwardHeartbeat(hb)
			}

		case TypeEvent:
			var ev EventPayload
			if json.Unmarshal(env.Payload, &ev) == nil {
				h.forwardEvent(ev)
			}
		}
	}
}

func (h *Handler) forwardHeartbeat(hb HeartbeatPayload) {
	for _, c := range hb.Containers {
		msg, err := json.Marshal(c)
		if err != nil {
			continue
		}
		h.Sink.Broadcast("server:"+c.ServerID+":stats", msg)
	}
}

func (h *Handler) forwardEvent(ev EventPayload) {
	msg, err := json.Marshal(ev)
	if err != nil {
		return
	}

	topic := "server:" + ev.ServerID + ":console"
	if ev.Kind == EventStateChanged {
		topic = "server:" + ev.ServerID + ":state"
	}
	h.Sink.Broadcast(topic, msg)
}
