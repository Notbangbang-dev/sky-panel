package agenthub

import (
	"sync"
	"time"
)

// ServerLocator resolves which node currently hosts a given server. It's an
// interface so the handler can be unit-tested without a database; in production
// it's backed by repo.Servers.
type ServerLocator interface {
	NodeIDForServer(serverID string) (nodeID string, ok bool)
}

// A node authenticates itself (valid token + signed envelopes), but nothing in
// the wire protocol stops a compromised node from reporting events/heartbeats
// for a server hosted on a *different* node. ownerCache closes that gap: it
// verifies each reported serverID is actually hosted on the connecting node,
// caching serverID -> nodeID for a short TTL so the per-console-line check
// doesn't hit the database on every event. A cached mismatch re-resolves, so a
// legitimate server transfer is picked up on the next event rather than waiting
// out the TTL.
const ownerCacheTTL = 60 * time.Second

type ownerEntry struct {
	nodeID string
	at     time.Time
}

type ownerCache struct {
	mu      sync.Mutex
	entries map[string]ownerEntry
	locate  ServerLocator
}

func newOwnerCache(locate ServerLocator) *ownerCache {
	return &ownerCache{entries: make(map[string]ownerEntry), locate: locate}
}

// ownedBy reports whether serverID is hosted on nodeID. With no locator wired
// (unit tests), it fails open so existing behaviour is unchanged.
func (c *ownerCache) ownedBy(serverID, nodeID string) bool {
	if c == nil || c.locate == nil {
		return true
	}
	if serverID == "" {
		return false
	}

	c.mu.Lock()
	e, ok := c.entries[serverID]
	fresh := ok && time.Since(e.at) < ownerCacheTTL
	c.mu.Unlock()
	if fresh && e.nodeID == nodeID {
		return true
	}

	// Cache miss, stale, or a mismatch (possible transfer or spoof) — re-resolve
	// against the source of truth and refresh the cache.
	owner, found := c.locate.NodeIDForServer(serverID)
	if found {
		c.mu.Lock()
		c.entries[serverID] = ownerEntry{nodeID: owner, at: time.Now()}
		c.mu.Unlock()
	}
	return found && owner == nodeID
}
