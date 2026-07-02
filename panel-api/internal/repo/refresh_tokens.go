package repo

import (
	"database/sql"
	"errors"
	"time"
)

type RefreshTokens struct {
	db *sql.DB
}

func NewRefreshTokens(db *sql.DB) *RefreshTokens {
	return &RefreshTokens{db: db}
}

func (r *RefreshTokens) Create(id, userID, tokenHash string, expiresAt time.Time) error {
	_, err := r.db.Exec(
		`INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at) VALUES (?, ?, ?, ?, ?)`,
		id, userID, tokenHash, expiresAt, time.Now().UTC(),
	)
	return err
}

// UserIDForValidToken returns the owning user ID for a non-expired token
// hash, or ErrNotFound if it doesn't exist or has expired.
func (r *RefreshTokens) UserIDForValidToken(tokenHash string) (string, error) {
	var userID string
	var expiresAt time.Time
	err := r.db.QueryRow(
		`SELECT user_id, expires_at FROM refresh_tokens WHERE token_hash = ?`, tokenHash,
	).Scan(&userID, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if time.Now().After(expiresAt) {
		return "", ErrNotFound
	}
	return userID, nil
}

func (r *RefreshTokens) DeleteByHash(tokenHash string) error {
	_, err := r.db.Exec(`DELETE FROM refresh_tokens WHERE token_hash = ?`, tokenHash)
	return err
}

func (r *RefreshTokens) DeleteAllForUser(userID string) error {
	_, err := r.db.Exec(`DELETE FROM refresh_tokens WHERE user_id = ?`, userID)
	return err
}

// Session is one of a user's active refresh tokens, shown as a login session.
type Session struct {
	ID        string
	CreatedAt time.Time
	ExpiresAt time.Time
	// Current is set by the handler when this row matches the caller's own
	// refresh token, so the UI can label "this device".
	Current bool
}

// ListByUser returns a user's unexpired refresh tokens, newest first.
func (r *RefreshTokens) ListByUser(userID string) ([]*Session, error) {
	rows, err := r.db.Query(
		`SELECT id, created_at, expires_at FROM refresh_tokens
		 WHERE user_id = ? AND expires_at > ? ORDER BY created_at DESC`,
		userID, time.Now().UTC(),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*Session
	for rows.Next() {
		var s Session
		if err := rows.Scan(&s.ID, &s.CreatedAt, &s.ExpiresAt); err != nil {
			return nil, err
		}
		out = append(out, &s)
	}
	return out, rows.Err()
}

// HashForID returns the token hash of a user's session by id (used to tell
// whether it's the caller's current session), or ErrNotFound.
func (r *RefreshTokens) HashForID(id, userID string) (string, error) {
	var hash string
	err := r.db.QueryRow(`SELECT token_hash FROM refresh_tokens WHERE id = ? AND user_id = ?`, id, userID).Scan(&hash)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return hash, err
}

// DeleteByIDForUser revokes one of the user's own sessions.
func (r *RefreshTokens) DeleteByIDForUser(id, userID string) error {
	res, err := r.db.Exec(`DELETE FROM refresh_tokens WHERE id = ? AND user_id = ?`, id, userID)
	return checkRowsAffected(res, err)
}

// DeleteOthersForUser revokes every session for the user except the one with
// keepHash (their current one).
func (r *RefreshTokens) DeleteOthersForUser(userID, keepHash string) error {
	_, err := r.db.Exec(`DELETE FROM refresh_tokens WHERE user_id = ? AND token_hash != ?`, userID, keepHash)
	return err
}
