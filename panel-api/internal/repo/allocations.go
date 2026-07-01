package repo

import (
	"database/sql"
	"errors"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
)

type Allocations struct {
	db *sql.DB
}

func NewAllocations(db *sql.DB) *Allocations {
	return &Allocations{db: db}
}

func (r *Allocations) Create(id, nodeID string, port int) error {
	_, err := r.db.Exec(`INSERT INTO allocations (id, node_id, port, server_id) VALUES (?, ?, ?, NULL)`, id, nodeID, port)
	if isUniqueViolation(err) {
		return ErrDuplicate
	}
	return err
}

// ClaimFree atomically claims one free (server_id IS NULL) allocation on the
// given node for serverID and returns the port it claimed. It returns
// ErrNotFound if no free allocation exists on that node.
func (r *Allocations) ClaimFree(nodeID, serverID string) (port int, err error) {
	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var allocID string
	err = tx.QueryRow(
		`SELECT id, port FROM allocations WHERE node_id = ? AND server_id IS NULL ORDER BY port LIMIT 1`, nodeID,
	).Scan(&allocID, &port)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, err
	}

	if _, err := tx.Exec(`UPDATE allocations SET server_id = ? WHERE id = ?`, serverID, allocID); err != nil {
		return 0, err
	}

	return port, tx.Commit()
}

func (r *Allocations) ReleaseByServerID(serverID string) error {
	_, err := r.db.Exec(`UPDATE allocations SET server_id = NULL WHERE server_id = ?`, serverID)
	return err
}

func (r *Allocations) ListByNode(nodeID string) ([]*models.Allocation, error) {
	rows, err := r.db.Query(`SELECT id, node_id, port, server_id FROM allocations WHERE node_id = ? ORDER BY port`, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*models.Allocation
	for rows.Next() {
		var a models.Allocation
		var serverID sql.NullString
		if err := rows.Scan(&a.ID, &a.NodeID, &a.Port, &serverID); err != nil {
			return nil, err
		}
		if serverID.Valid {
			a.ServerID = &serverID.String
		}
		out = append(out, &a)
	}
	return out, rows.Err()
}
