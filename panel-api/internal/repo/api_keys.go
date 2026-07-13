package repo

import (
	"database/sql"
	"errors"
	"time"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
)

type APIKeys struct {
	db *sql.DB
}

func NewAPIKeys(db *sql.DB) *APIKeys {
	return &APIKeys{db: db}
}

func (r *APIKeys) Create(id, userID, name, keyHash string) error {
	_, err := r.db.Exec(
		`INSERT INTO api_keys (id, user_id, name, key_hash, created_at) VALUES (?, ?, ?, ?, ?)`,
		id, userID, name, keyHash, time.Now().UTC(),
	)
	return err
}

// CountByUser returns how many API keys a user currently has, used to enforce
// a per-user cap so a leaked session can't mint an unbounded number of keys.
func (r *APIKeys) CountByUser(userID string) (int, error) {
	var n int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM api_keys WHERE user_id = ?`, userID).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

func (r *APIKeys) ListByUser(userID string) ([]*models.APIKey, error) {
	rows, err := r.db.Query(
		`SELECT id, name, last_used_at, created_at FROM api_keys WHERE user_id = ? ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*models.APIKey
	for rows.Next() {
		var k models.APIKey
		var lastUsed sql.NullTime
		if err := rows.Scan(&k.ID, &k.Name, &lastUsed, &k.CreatedAt); err != nil {
			return nil, err
		}
		if lastUsed.Valid {
			t := lastUsed.Time
			k.LastUsedAt = &t
		}
		out = append(out, &k)
	}
	return out, rows.Err()
}

// UserIDForKeyHash resolves an API key hash to its owner, or ErrNotFound.
func (r *APIKeys) UserIDForKeyHash(keyHash string) (string, error) {
	var userID string
	err := r.db.QueryRow(`SELECT user_id FROM api_keys WHERE key_hash = ?`, keyHash).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return userID, err
}

func (r *APIKeys) TouchLastUsed(keyHash string) {
	_, _ = r.db.Exec(`UPDATE api_keys SET last_used_at = ? WHERE key_hash = ?`, time.Now().UTC(), keyHash)
}

func (r *APIKeys) DeleteByIDForUser(id, userID string) error {
	res, err := r.db.Exec(`DELETE FROM api_keys WHERE id = ? AND user_id = ?`, id, userID)
	return checkRowsAffected(res, err)
}
