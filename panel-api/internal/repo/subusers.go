package repo

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
)

type Subusers struct {
	db *sql.DB
}

func NewSubusers(db *sql.DB) *Subusers {
	return &Subusers{db: db}
}

func (r *Subusers) Create(id, serverID, userID string, permissions []string) error {
	_, err := r.db.Exec(
		`INSERT INTO server_subusers (id, server_id, user_id, permissions, created_at) VALUES (?, ?, ?, ?, ?)`,
		id, serverID, userID, strings.Join(permissions, ","), time.Now().UTC(),
	)
	if isUniqueViolation(err) {
		return ErrDuplicate
	}
	return err
}

func (r *Subusers) Get(serverID, userID string) (*models.Subuser, error) {
	row := r.db.QueryRow(
		`SELECT id, server_id, user_id, permissions, created_at FROM server_subusers WHERE server_id = ? AND user_id = ?`,
		serverID, userID,
	)
	s, err := scanSubuserRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return s, err
}

func (r *Subusers) ListByServer(serverID string) ([]*models.Subuser, error) {
	rows, err := r.db.Query(
		`SELECT id, server_id, user_id, permissions, created_at FROM server_subusers WHERE server_id = ? ORDER BY created_at`,
		serverID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*models.Subuser
	for rows.Next() {
		s, err := scanSubuserRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Subusers) Delete(serverID, userID string) error {
	res, err := r.db.Exec(`DELETE FROM server_subusers WHERE server_id = ? AND user_id = ?`, serverID, userID)
	return checkRowsAffected(res, err)
}

func scanSubuserRow(row rowScanner) (*models.Subuser, error) {
	var s models.Subuser
	var permissions string
	if err := row.Scan(&s.ID, &s.ServerID, &s.UserID, &permissions, &s.CreatedAt); err != nil {
		return nil, err
	}
	if permissions != "" {
		s.Permissions = strings.Split(permissions, ",")
	}
	return &s, nil
}
