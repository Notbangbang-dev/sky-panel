package repo

import (
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
)

// ErrAllocationInUse is returned when trying to delete an allocation that a
// server currently holds.
var ErrAllocationInUse = errors.New("allocation is in use by a server")

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

// CreateRange adds every port in [start, end] as a free allocation on nodeID,
// silently skipping ports that already exist (UNIQUE(node_id, port)). It
// returns how many were newly created. Used both by the admin UI and to
// auto-seed a default port range when a node is registered.
func (r *Allocations) CreateRange(nodeID string, start, end int) (int, error) {
	if start > end {
		start, end = end, start
	}
	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(
		`INSERT INTO allocations (id, node_id, port, server_id) VALUES (?, ?, ?, NULL)
		 ON CONFLICT(node_id, port) DO NOTHING`,
	)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	created := 0
	for p := start; p <= end; p++ {
		res, err := stmt.Exec(uuid.NewString(), nodeID, p)
		if err != nil {
			return created, err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			created++
		}
	}
	return created, tx.Commit()
}

// Delete removes a free allocation. It refuses (ErrAllocationInUse) if a
// server still holds it, and returns ErrNotFound if it doesn't exist. The
// check-and-delete is atomic (a single conditional DELETE in a transaction)
// so a concurrent ClaimFree can't slip a server onto the port between the
// check and the delete.
func (r *Allocations) Delete(id string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`DELETE FROM allocations WHERE id = ? AND server_id IS NULL`, id)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		// Nothing deleted: either the row doesn't exist, or it's in use.
		var serverID sql.NullString
		err := tx.QueryRow(`SELECT server_id FROM allocations WHERE id = ?`, id).Scan(&serverID)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		return ErrAllocationInUse
	}
	return tx.Commit()
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
