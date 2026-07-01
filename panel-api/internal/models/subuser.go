package models

import "time"

const (
	PermConsole  = "console"
	PermFiles    = "files"
	PermPower    = "power"
	PermSettings = "settings"
)

// AllPermissions is used to validate a requested permission list.
var AllPermissions = map[string]bool{
	PermConsole:  true,
	PermFiles:    true,
	PermPower:    true,
	PermSettings: true,
}

type Subuser struct {
	ID          string
	ServerID    string
	UserID      string
	Permissions []string
	CreatedAt   time.Time
}

func (s *Subuser) HasPermission(perm string) bool {
	for _, p := range s.Permissions {
		if p == perm {
			return true
		}
	}
	return false
}
