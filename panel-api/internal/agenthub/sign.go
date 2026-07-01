package agenthub

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
)

// MaxClockSkewSecs bounds how far a timestamp may drift from "now" (either
// direction) and still be accepted. Mirrors sky-daemon's protocol crate —
// both sides must agree on this window.
const MaxClockSkewSecs = 30

// canonicalString builds the string that gets HMAC'd:
// "type.timestamp.nonce.sha256(payload)". Matches sky-daemon's Rust
// implementation byte-for-byte so signatures verify across the wire.
func canonicalString(kind string, timestamp int64, nonce string, payloadBytes []byte) string {
	sum := sha256.Sum256(payloadBytes)
	return fmt.Sprintf("%s.%d.%s.%s", kind, timestamp, nonce, hex.EncodeToString(sum[:]))
}

// Sign returns the hex-encoded HMAC-SHA256 signature for the given fields,
// keyed by secret (the node's raw token).
func Sign(secret []byte, kind string, timestamp int64, nonce string, payloadBytes []byte) string {
	canonical := canonicalString(kind, timestamp, nonce, payloadBytes)
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(canonical))
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify checks sigHex against the given fields in constant time.
func Verify(secret []byte, kind string, timestamp int64, nonce string, payloadBytes []byte, sigHex string) bool {
	expected := Sign(secret, kind, timestamp, nonce, payloadBytes)
	return subtle.ConstantTimeCompare([]byte(expected), []byte(sigHex)) == 1
}
