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

// AFKSession is the persisted state of a user's AFK earning session: when they
// last checked in, and which browser tab ("session") owns the session. Only
// one session earns at a time — see coinsvc.Heartbeat.
type AFKSession struct {
	LastHeartbeat    time.Time
	SessionID        string
	SessionStartedAt time.Time
}

// Get returns userID's AFK session state, or found=false if they've never
// sent a heartbeat.
func (r *AFKState) Get(userID string) (sess AFKSession, found bool, err error) {
	var startedAt sql.NullTime
	err = r.db.QueryRow(
		`SELECT last_heartbeat_at, session_id, session_started_at FROM afk_state WHERE user_id = ?`,
		userID,
	).Scan(&sess.LastHeartbeat, &sess.SessionID, &startedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return AFKSession{}, false, nil
	}
	if err != nil {
		return AFKSession{}, false, err
	}
	if startedAt.Valid {
		sess.SessionStartedAt = startedAt.Time
	}
	return sess, true, nil
}

// Set upserts a user's AFK session state.
func (r *AFKState) Set(userID, sessionID string, lastHeartbeat, sessionStartedAt time.Time) error {
	_, err := r.db.Exec(
		`INSERT INTO afk_state (user_id, last_heartbeat_at, session_id, session_started_at) VALUES (?, ?, ?, ?)
		 ON CONFLICT(user_id) DO UPDATE SET
		   last_heartbeat_at  = excluded.last_heartbeat_at,
		   session_id         = excluded.session_id,
		   session_started_at = excluded.session_started_at`,
		userID, lastHeartbeat, sessionID, sessionStartedAt,
	)
	return err
}
