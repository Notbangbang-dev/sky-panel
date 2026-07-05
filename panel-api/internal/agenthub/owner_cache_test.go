package agenthub

import "testing"

type fakeLocator struct {
	owner map[string]string
	calls int
}

func (f *fakeLocator) NodeIDForServer(serverID string) (string, bool) {
	f.calls++
	n, ok := f.owner[serverID]
	return n, ok
}

func TestOwnerCacheNilLocatorFailsOpen(t *testing.T) {
	c := newOwnerCache(nil)
	if !c.ownedBy("s1", "n1") {
		t.Fatal("with no locator wired the check must fail open")
	}
}

func TestOwnerCacheEnforcesOwnership(t *testing.T) {
	loc := &fakeLocator{owner: map[string]string{"s1": "n1"}}
	c := newOwnerCache(loc)

	if !c.ownedBy("s1", "n1") {
		t.Fatal("n1 hosts s1 — should be allowed")
	}
	if c.ownedBy("s1", "n2") {
		t.Fatal("n2 does not host s1 — should be rejected")
	}
	if c.ownedBy("ghost", "n1") {
		t.Fatal("unknown server must fail closed")
	}
	if c.ownedBy("", "n1") {
		t.Fatal("empty server id must fail closed")
	}
}

func TestOwnerCacheCachesLookups(t *testing.T) {
	loc := &fakeLocator{owner: map[string]string{"s1": "n1"}}
	c := newOwnerCache(loc)

	c.ownedBy("s1", "n1")
	c.ownedBy("s1", "n1")
	if loc.calls != 1 {
		t.Fatalf("expected a single cached lookup, got %d", loc.calls)
	}

	// A mismatch re-resolves (so a transfer is picked up promptly).
	c.ownedBy("s1", "n2")
	if loc.calls != 2 {
		t.Fatalf("a mismatch should trigger a fresh lookup, got %d calls", loc.calls)
	}
}
