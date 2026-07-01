package models

import "time"

type AuditEntry struct {
	ID        string
	ActorID   string
	Action    string
	Target    string
	Metadata  string
	CreatedAt time.Time
}
