package repo

import (
	"database/sql"
	"errors"
	"time"
)

type AFKState struct {
	db *sql.DB
}

func NewAFKState(db *sql.DB) *AFKState {
	return &AFKState{db: db}
}

// LastHeartbeat returns the time of userID's last recorded AFK heartbeat, or
// found=false if they have never sent one.
func (r *AFKState) LastHeartbeat(userID string) (t time.Time, found bool, err error) {
	err = r.db.QueryRow(`SELECT last_heartbeat_at FROM afk_state WHERE user_id = ?`, userID).Scan(&t)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, err
	}
	return t, true, nil
}

func (r *AFKState) SetLastHeartbeat(userID string, t time.Time) error {
	_, err := r.db.Exec(
		`INSERT INTO afk_state (user_id, last_heartbeat_at) VALUES (?, ?)
		 ON CONFLICT(user_id) DO UPDATE SET last_heartbeat_at = excluded.last_heartbeat_at`,
		userID, t,
	)
	return err
}
