package wshub

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
	sendBuffer = 32
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Sky Panel is same-origin (the web build is served behind the same
	// reverse proxy as the API in production); CheckOrigin stays permissive
	// here for local dev across ports and is tightened at the proxy layer.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Client wraps one browser WebSocket connection and implements Subscriber.
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

func (c *Client) Send(msg []byte) {
	select {
	case c.send <- msg:
	default:
		// Slow consumer: drop the message rather than blocking the
		// broadcaster or growing memory unbounded.
	}
}

// Upgrade upgrades r into a WebSocket connection subscribed to topics, then
// blocks running the read/write pumps until the connection closes.
func Upgrade(hub *Hub, w http.ResponseWriter, r *http.Request, topics []string) error {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}

	c := &Client{hub: hub, conn: conn, send: make(chan []byte, sendBuffer)}
	for _, topic := range topics {
		hub.Subscribe(topic, c)
	}
	defer hub.UnsubscribeAll(c)

	done := make(chan struct{})
	go c.writePump(done)
	c.readPump(done)

	return nil
}

func (c *Client) readPump(done chan struct{}) {
	defer close(done)
	defer c.conn.Close()

	c.conn.SetReadLimit(4096)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		// Clients only receive in this MVP (no client->server chat over
		// this socket yet); we still read to drive the pong handler and
		// detect disconnects.
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (c *Client) writePump(done chan struct{}) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	defer c.conn.Close()

	for {
		select {
		case <-done:
			return
		case msg := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
