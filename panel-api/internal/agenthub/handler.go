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
	stats    *statsCache
	players  *playerTracker
	owners   *ownerCache
	// OnNodeConnected, if set, is invoked (in its own goroutine) each time a
	// node finishes its hello handshake — used to warm the node's image cache.
	// Optional so the handler stays testable without pulling in serversvc.
	OnNodeConnected func(nodeID string)
}

func NewHandler(registry *Registry, nodes NodeLookup, sink EventSink) *Handler {
	stats := newStatsCache()
	go stats.sweepLoop()
	players := newPlayerTracker()
	go players.sweepLoop()
	return &Handler{Registry: registry, Nodes: nodes, Sink: sink, stats: stats, players: players, owners: newOwnerCache(nil)}
}

// UseServerLocator enables cross-node authorization: reported events and
// heartbeats are dropped unless the connecting node actually hosts the server
// they reference. Called by main once the servers repo is available; without it
// the check is disabled (unit tests).
func (h *Handler) UseServerLocator(locate ServerLocator) {
	h.owners = newOwnerCache(locate)
}

// Forget drops a server's tracked roster immediately (e.g. on deletion).
func (h *Handler) Forget(serverID string) {
	if h.players != nil {
		h.players.forget(serverID)
	}
}

// LatestStats returns the most recent (and still fresh) heartbeat JSON for a
// server, so an HTTP caller can seed the UI without waiting for a WS push.
func (h *Handler) LatestStats(serverID string) ([]byte, bool) {
	return h.stats.get(serverID)
}

// Players returns the live roster (and version) tracked from the console stream.
func (h *Handler) Players(serverID string) PlayerInfo {
	return h.players.get(serverID)
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

	nodeID, secret, caps, err := h.awaitHello(ws)
	if err != nil {
		log.Printf("agenthub: rejecting connection: %v", err)
		return
	}

	conn := newConn(nodeID, ws, secret, caps)
	h.Registry.register(nodeID, conn)
	defer h.Registry.unregister(nodeID, conn)

	log.Printf("agenthub: node %s connected", nodeID)
	if h.OnNodeConnected != nil {
		go h.OnNodeConnected(nodeID)
	}
	h.readLoop(conn, ws, secret)
	// The socket is dead; unblock anything still awaiting an ack on it.
	conn.failPending()
	log.Printf("agenthub: node %s disconnected", nodeID)
}

func (h *Handler) awaitHello(ws *websocket.Conn) (nodeID string, secret []byte, capabilities []string, err error) {
	var env Envelope
	if err := ws.ReadJSON(&env); err != nil {
		return "", nil, nil, err
	}
	if env.Type != TypeHello {
		return "", nil, nil, errUnexpectedFirstMessage
	}

	var hello HelloPayload
	if err := json.Unmarshal(env.Payload, &hello); err != nil {
		return "", nil, nil, err
	}

	id, token, expiresAt, err := h.Nodes.AuthenticateNode(auth.HashToken(hello.NodeToken))
	if err != nil {
		return "", nil, nil, err
	}
	if time.Now().After(expiresAt) {
		return "", nil, nil, errNodeTokenExpired
	}

	return id, []byte(token), hello.Capabilities, nil
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
			if err := json.Unmarshal(env.Payload, &hb); err != nil {
				log.Printf("agenthub: node %s sent an undecodable heartbeat: %v", conn.NodeID, err)
			} else {
				h.forwardHeartbeat(conn.NodeID, hb)
			}

		case TypeEvent:
			var ev EventPayload
			if json.Unmarshal(env.Payload, &ev) == nil {
				h.forwardEvent(conn.NodeID, ev)
			}
		}
	}
}

func (h *Handler) forwardHeartbeat(nodeID string, hb HeartbeatPayload) {
	for _, c := range hb.Containers {
		if c.ServerID == "" {
			// Would publish to "server::stats", which no client subscribes to.
			continue
		}
		// Drop stats for servers this node doesn't host, so a compromised node
		// can't poison another server's cached metrics.
		if !h.owners.ownedBy(c.ServerID, nodeID) {
			continue
		}
		msg, err := json.Marshal(c)
		if err != nil {
			continue
		}
		if h.stats != nil {
			h.stats.put(c.ServerID, msg)
		}
		h.Sink.Broadcast("server:"+c.ServerID+":stats", msg)
	}
}

func (h *Handler) forwardEvent(nodeID string, ev EventPayload) {
	// Drop events for servers this node doesn't host, so a compromised node
	// can't corrupt another server's roster or spoof its console/state stream.
	if !h.owners.ownedBy(ev.ServerID, nodeID) {
		log.Printf("agenthub: node %s reported an event for server %s it does not host; dropping", nodeID, ev.ServerID)
		return
	}
	if h.players != nil {
		h.players.observe(ev)
	}
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
