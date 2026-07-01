package agenthub

import (
	"sync"
	"time"
)

// nonceCache is a tiny in-memory replay-protection cache: a nonce is only
// accepted once within ttl. Process-local and unpersisted — a restart
// briefly reopens the replay window, an accepted trade-off (see
// docs/ARCHITECTURE.md) rather than pulling in a shared store.
type nonceCache struct {
	mu   sync.Mutex
	seen map[string]time.Time
	ttl  time.Duration
}

func newNonceCache(ttl time.Duration) *nonceCache {
	return &nonceCache{seen: make(map[string]time.Time), ttl: ttl}
}

// checkAndRecord returns true (and records the nonce) if it has not been
// seen within the last ttl; returns false if it's a replay. Also
// opportunistically sweeps expired entries.
func (c *nonceCache) checkAndRecord(nonce string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for n, seenAt := range c.seen {
		if now.Sub(seenAt) >= c.ttl {
			delete(c.seen, n)
		}
	}

	if _, ok := c.seen[nonce]; ok {
		return false
	}
	c.seen[nonce] = now
	return true
}
