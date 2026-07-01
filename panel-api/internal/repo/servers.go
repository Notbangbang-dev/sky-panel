package repo

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
)

type Servers struct {
	db *sql.DB
}

func NewServers(db *sql.DB) *Servers {
	return &Servers{db: db}
}

func (r *Servers) Create(s *models.Server) error {
	varsJSON, err := json.Marshal(s.Variables)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(
		`INSERT INTO servers (id, owner_id, node_id, egg_id, name, container_id, status, memory_bytes, variables_json, primary_port, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.OwnerID, s.NodeID, s.EggID, s.Name, s.ContainerID, string(s.Status), s.MemoryBytes, varsJSON, s.PrimaryPort, s.CreatedAt, s.UpdatedAt,
	)
	return err
}

func (r *Servers) GetByID(id string) (*models.Server, error) {
	row := r.db.QueryRow(
		`SELECT id, owner_id, node_id, egg_id, name, container_id, status, memory_bytes, variables_json, primary_port, created_at, updated_at
		 FROM servers WHERE id = ?`, id)
	return scanServer(row)
}

func (r *Servers) ListByOwner(ownerID string) ([]*models.Server, error) {
	rows, err := r.db.Query(
		`SELECT id, owner_id, node_id, egg_id, name, container_id, status, memory_bytes, variables_json, primary_port, created_at, updated_at
		 FROM servers WHERE owner_id = ? ORDER BY created_at`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanServerRows(rows)
}

func (r *Servers) ListAll() ([]*models.Server, error) {
	rows, err := r.db.Query(
		`SELECT id, owner_id, node_id, egg_id, name, container_id, status, memory_bytes, variables_json, primary_port, created_at, updated_at
		 FROM servers ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanServerRows(rows)
}

func (r *Servers) SetPrimaryPort(id string, port int) error {
	res, err := r.db.Exec(`UPDATE servers SET primary_port = ?, updated_at = ? WHERE id = ?`, port, time.Now().UTC(), id)
	return checkRowsAffected(res, err)
}

func (r *Servers) SetContainerID(id, containerID string) error {
	res, err := r.db.Exec(`UPDATE servers SET container_id = ?, updated_at = ? WHERE id = ?`, containerID, time.Now().UTC(), id)
	return checkRowsAffected(res, err)
}

func (r *Servers) SetStatus(id string, status models.ServerStatus) error {
	res, err := r.db.Exec(`UPDATE servers SET status = ?, updated_at = ? WHERE id = ?`, string(status), time.Now().UTC(), id)
	return checkRowsAffected(res, err)
}

func (r *Servers) Delete(id string) error {
	res, err := r.db.Exec(`DELETE FROM servers WHERE id = ?`, id)
	return checkRowsAffected(res, err)
}

func scanServer(row rowScanner) (*models.Server, error) {
	s, err := scanServerRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return s, err
}

func scanServerRow(row rowScanner) (*models.Server, error) {
	var s models.Server
	var status, varsJSON string

	if err := row.Scan(&s.ID, &s.OwnerID, &s.NodeID, &s.EggID, &s.Name, &s.ContainerID, &status, &s.MemoryBytes, &varsJSON, &s.PrimaryPort, &s.CreatedAt, &s.UpdatedAt); err != nil {
		return nil, err
	}

	s.Status = models.ServerStatus(status)
	if err := json.Unmarshal([]byte(varsJSON), &s.Variables); err != nil {
		return nil, err
	}
	return &s, nil
}

func scanServerRows(rows *sql.Rows) ([]*models.Server, error) {
	var out []*models.Server
	for rows.Next() {
		s, err := scanServerRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
