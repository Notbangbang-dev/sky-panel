package models

import "time"

type Node struct {
	ID           string
	Name         string
	TokenHash    string
	// Token is the raw secret. Kept in the clear (not just hashed) because
	// it doubles as the HMAC key for verifying every signed message after
	// hello — see internal/agenthub.
	Token        string
	ExpiresAt    time.Time
	Address      string
	DockerSocket string
	CreatedAt    time.Time
}
