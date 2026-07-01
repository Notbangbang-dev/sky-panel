package agenthub

import (
	"testing"
	"time"
)

func TestNonceCacheFirstUseAccepted(t *testing.T) {
	c := newNonceCache(30 * time.Second)
	if !c.checkAndRecord("abc") {
		t.Error("expected first use to be accepted")
	}
}

func TestNonceCacheRejectsReplay(t *testing.T) {
	c := newNonceCache(30 * time.Second)
	if !c.checkAndRecord("abc") {
		t.Fatal("expected first use to be accepted")
	}
	if c.checkAndRecord("abc") {
		t.Error("expected replay to be rejected")
	}
}

func TestNonceCacheDistinctNoncesIndependent(t *testing.T) {
	c := newNonceCache(30 * time.Second)
	if !c.checkAndRecord("a") {
		t.Error("expected 'a' to be accepted")
	}
	if !c.checkAndRecord("b") {
		t.Error("expected 'b' to be accepted")
	}
}

func TestNonceCacheAcceptsAgainAfterTTL(t *testing.T) {
	c := newNonceCache(20 * time.Millisecond)
	if !c.checkAndRecord("abc") {
		t.Fatal("expected first use to be accepted")
	}
	time.Sleep(40 * time.Millisecond)
	if !c.checkAndRecord("abc") {
		t.Error("expected nonce to be accepted again after TTL expiry")
	}
}
