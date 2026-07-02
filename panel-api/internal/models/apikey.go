package models

import "time"

// APIKey is a personal access token. Only its hash is stored; the raw key is
// shown once at creation.
type APIKey struct {
	ID         string
	UserID     string
	Name       string
	LastUsedAt *time.Time
	CreatedAt  time.Time
}
