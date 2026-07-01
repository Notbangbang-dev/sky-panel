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
