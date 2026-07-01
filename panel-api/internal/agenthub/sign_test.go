package agenthub

import "testing"

func TestSignAndVerifyRoundTrip(t *testing.T) {
	secret := []byte("node-secret-token")
	sig := Sign(secret, "heartbeat", 1_700_000_000, "abc123", []byte(`{"ok":true}`))
	if !Verify(secret, "heartbeat", 1_700_000_000, "abc123", []byte(`{"ok":true}`), sig) {
		t.Error("expected verify to succeed with matching fields")
	}
}

func TestVerifyRejectsTamperedPayload(t *testing.T) {
	secret := []byte("node-secret-token")
	sig := Sign(secret, "heartbeat", 1_700_000_000, "abc123", []byte(`{"ok":true}`))
	if Verify(secret, "heartbeat", 1_700_000_000, "abc123", []byte(`{"ok":false}`), sig) {
		t.Error("expected verify to reject a tampered payload")
	}
}

func TestVerifyRejectsWrongSecret(t *testing.T) {
	sig := Sign([]byte("secret-a"), "heartbeat", 1_700_000_000, "abc123", []byte("{}"))
	if Verify([]byte("secret-b"), "heartbeat", 1_700_000_000, "abc123", []byte("{}"), sig) {
		t.Error("expected verify to reject a different secret")
	}
}

func TestVerifyRejectsReplayUnderDifferentType(t *testing.T) {
	secret := []byte("node-secret-token")
	sig := Sign(secret, "event", 1_700_000_000, "abc123", []byte("{}"))
	if Verify(secret, "heartbeat", 1_700_000_000, "abc123", []byte("{}"), sig) {
		t.Error("expected a signature computed for one type to be rejected under another")
	}
}
