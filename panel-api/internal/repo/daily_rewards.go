package repo

import (
	"database/sql"
	"errors"
	"time"
)

type DailyRewards struct {
	db *sql.DB
}

func NewDailyRewards(db *sql.DB) *DailyRewards {
	return &DailyRewards{db: db}
}

func (r *DailyRewards) LastClaimed(userID string) (t time.Time, found bool, err error) {
	err = r.db.QueryRow(`SELECT last_claimed_at FROM daily_reward_claims WHERE user_id = ?`, userID).Scan(&t)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, err
	}
	return t, true, nil
}

func (r *DailyRewards) SetLastClaimed(userID string, t time.Time) error {
	_, err := r.db.Exec(
		`INSERT INTO daily_reward_claims (user_id, last_claimed_at) VALUES (?, ?)
		 ON CONFLICT(user_id) DO UPDATE SET last_claimed_at = excluded.last_claimed_at`,
		userID, t,
	)
	return err
}

// ClaimIfDue atomically records a claim at `now`, but only if the user has
// never claimed or last claimed at or before `cutoff` (= now - interval). It
// returns whether the claim was recorded. Because the check and the write are
// a single conditional UPSERT, two concurrent requests can't both succeed —
// this is what prevents the daily reward from being double-claimed in a race.
func (r *DailyRewards) ClaimIfDue(userID string, now, cutoff time.Time) (bool, error) {
	res, err := r.db.Exec(
		`INSERT INTO daily_reward_claims (user_id, last_claimed_at) VALUES (?, ?)
		 ON CONFLICT(user_id) DO UPDATE SET last_claimed_at = excluded.last_claimed_at
		   WHERE daily_reward_claims.last_claimed_at <= ?`,
		userID, now, cutoff,
	)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
