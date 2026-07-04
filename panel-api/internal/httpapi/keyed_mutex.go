package httpapi

import "sync"

// keyedMutex provides per-key mutual exclusion. It serializes the non-atomic
// "check quota, then create the server row" sequence per user: without it, a
// burst of concurrent create/clone requests from one user would each pass the
// quota check against the same pre-insert usage snapshot and collectively
// overshoot the limit (a TOCTOU). Scoped to a single panel-api process.
type keyedMutex struct {
	mu    sync.Mutex
	locks map[string]*refCountedLock
}

type refCountedLock struct {
	mu   sync.Mutex
	refs int
}

func newKeyedMutex() *keyedMutex {
	return &keyedMutex{locks: make(map[string]*refCountedLock)}
}

// lock acquires the lock for key and returns a function that releases it.
// Entries are reference-counted and dropped when the last holder unlocks, so
// the map does not grow without bound as users come and go.
func (k *keyedMutex) lock(key string) func() {
	k.mu.Lock()
	l, ok := k.locks[key]
	if !ok {
		l = &refCountedLock{}
		k.locks[key] = l
	}
	l.refs++
	k.mu.Unlock()

	l.mu.Lock()

	return func() {
		l.mu.Unlock()
		k.mu.Lock()
		l.refs--
		if l.refs == 0 {
			delete(k.locks, key)
		}
		k.mu.Unlock()
	}
}

// serverCreateLocks serializes per-user server creation (create + clone) so the
// quota check and the row insert can't interleave across concurrent requests.
var serverCreateLocks = newKeyedMutex()
