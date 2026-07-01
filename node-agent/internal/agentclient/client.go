package agentclient

import (
	"context"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	minBackoff = 1 * time.Second
	maxBackoff = 30 * time.Second
)

// Run dials wsURL and keeps the connection alive, reconnecting with capped
// exponential backoff whenever it drops, until ctx is cancelled.
func Run(ctx context.Context, wsURL, nodeToken string, dispatch *Dispatcher, heartbeatInterval time.Duration) {
	backoff := minBackoff

	for {
		if ctx.Err() != nil {
			return
		}

		conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
		if err != nil {
			log.Printf("agentclient: dial %s failed: %v (retrying in %s)", wsURL, err, backoff)
			if !sleep(ctx, backoff) {
				return
			}
			backoff = nextBackoff(backoff)
			continue
		}

		backoff = minBackoff
		session := NewSession(conn, nodeToken, dispatch, heartbeatInterval)

		if err := session.Run(ctx); err != nil {
			log.Printf("agentclient: session ended: %v", err)
		}
		conn.Close()

		if !sleep(ctx, minBackoff) {
			return
		}
	}
}

func nextBackoff(cur time.Duration) time.Duration {
	next := cur * 2
	if next > maxBackoff {
		return maxBackoff
	}
	return next
}

// sleep waits for d or ctx cancellation, returning false if ctx was
// cancelled first (signalling the caller should stop looping).
func sleep(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
