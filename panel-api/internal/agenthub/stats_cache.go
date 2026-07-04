package agenthub

import (
	"sync"
	"time"
)

// statsCache remembers the most recent ContainerHeartbeat JSON per server, so a
// browser that just loaded the page (or briefly lost its WebSocket) can fetch
// current stats over HTTP instead of waiting for the next 5s push and showing a
// dash in the meantime. Entries older than statsStaleAfter are treated as gone
// — the daemon has stopped reporting that container (server stopped/removed).
type statsCache struct {
	mu      sync.RWMutex
	entries map[string]statsEntry
}

type statsEntry struct {
	json []byte
	at   time.Time
}

const statsStaleAfter = 20 * time.Second

func newStatsCache() *statsCache {
	return &statsCache{entries: make(map[string]statsEntry)}
}

func (c *statsCache) put(serverID string, json []byte) {
	c.mu.Lock()
	c.entries[serverID] = statsEntry{json: json, at: time.Now()}
	c.mu.Unlock()
}

// get returns the cached heartbeat JSON if present and still fresh, evicting
// the entry when it has gone stale so a deleted/stopped server doesn't linger.
func (c *statsCache) get(serverID string) ([]byte, bool) {
	c.mu.RLock()
	e, ok := c.entries[serverID]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Since(e.at) > statsStaleAfter {
		c.mu.Lock()
		// Re-check under the write lock in case a heartbeat refreshed it.
		if cur, still := c.entries[serverID]; still && time.Since(cur.at) > statsStaleAfter {
			delete(c.entries, serverID)
		}
		c.mu.Unlock()
		return nil, false
	}
	return e.json, true
}

// sweep drops every entry that has gone stale. Run periodically so entries for
// servers whose page is never opened again are still reclaimed (get() only
// evicts on read).
func (c *statsCache) sweep() {
	c.mu.Lock()
	for id, e := range c.entries {
		if time.Since(e.at) > statsStaleAfter {
			delete(c.entries, id)
		}
	}
	c.mu.Unlock()
}

// sweepLoop reclaims stale entries on a fixed cadence for the process lifetime.
func (c *statsCache) sweepLoop() {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for range t.C {
		c.sweep()
	}
}
