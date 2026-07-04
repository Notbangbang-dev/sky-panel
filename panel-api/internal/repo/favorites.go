package repo

import (
	"database/sql"
	"time"
)

// Favorites tracks which servers a user has starred.
type Favorites struct {
	db *sql.DB
}

func NewFavorites(db *sql.DB) *Favorites {
	return &Favorites{db: db}
}

// Add stars a server for a user (idempotent).
func (r *Favorites) Add(userID, serverID string) error {
	_, err := r.db.Exec(
		`INSERT INTO server_favorites (user_id, server_id, created_at) VALUES (?, ?, ?)
		 ON CONFLICT(user_id, server_id) DO NOTHING`,
		userID, serverID, time.Now().UTC(),
	)
	return err
}

// Remove unstars a server for a user (idempotent).
func (r *Favorites) Remove(userID, serverID string) error {
	_, err := r.db.Exec(`DELETE FROM server_favorites WHERE user_id = ? AND server_id = ?`, userID, serverID)
	return err
}

// ListByUser returns the server IDs a user has favorited.
func (r *Favorites) ListByUser(userID string) ([]string, error) {
	rows, err := r.db.Query(`SELECT server_id FROM server_favorites WHERE user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
