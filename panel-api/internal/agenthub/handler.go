package agenthub

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/auth"
)

var errUnexpectedFirstMessage = errors.New("agenthub: expected hello as the first message")

// NodeLookup resolves a node token hash to a node ID. It's an interface
// (rather than *repo.Nodes directly) so the handler can be unit tested
// without a database.
type NodeLookup interface {
	NodeIDForTokenHash(tokenHash string) (string, error)
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

// ServeWS accepts a node-agent's inbound connection. The HTTP layer itself
// requires no auth (nodes dial in from arbitrary VPS IPs); identity is
// established by validating the first message's node token instead.
func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer ws.Close()

	nodeID, err := h.awaitHello(ws)
	if err != nil {
		log.Printf("agenthub: rejecting connection: %v", err)
		return
	}

	conn := newConn(nodeID, ws)
	h.Registry.register(nodeID, conn)
	defer h.Registry.unregister(nodeID, conn)

	log.Printf("agenthub: node %s connected", nodeID)
	h.readLoop(nodeID, conn, ws)
	log.Printf("agenthub: node %s disconnected", nodeID)
}

func (h *Handler) awaitHello(ws *websocket.Conn) (string, error) {
	var env Envelope
	if err := ws.ReadJSON(&env); err != nil {
		return "", err
	}
	if env.Type != TypeHello {
		return "", errUnexpectedFirstMessage
	}

	var hello HelloPayload
	if err := json.Unmarshal(env.Payload, &hello); err != nil {
		return "", err
	}

	nodeID, err := h.Nodes.NodeIDForTokenHash(auth.HashToken(hello.NodeToken))
	if err != nil {
		return "", err
	}
	return nodeID, nil
}

func (h *Handler) readLoop(nodeID string, conn *Conn, ws *websocket.Conn) {
	for {
		var env Envelope
		if err := ws.ReadJSON(&env); err != nil {
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
