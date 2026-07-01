package repo

import (
	"database/sql"
	"errors"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
)

type Nodes struct {
	db *sql.DB
}

func NewNodes(db *sql.DB) *Nodes {
	return &Nodes{db: db}
}

func (r *Nodes) Create(n *models.Node) error {
	_, err := r.db.Exec(
		`INSERT INTO nodes (id, name, token_hash, address, docker_socket, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		n.ID, n.Name, n.TokenHash, n.Address, n.DockerSocket, n.CreatedAt,
	)
	if isUniqueViolation(err) {
		return ErrDuplicate
	}
	return err
}

func (r *Nodes) GetByID(id string) (*models.Node, error) {
	return r.scanOne(`SELECT id, name, token_hash, address, docker_socket, created_at FROM nodes WHERE id = ?`, id)
}

func (r *Nodes) GetByTokenHash(tokenHash string) (*models.Node, error) {
	return r.scanOne(`SELECT id, name, token_hash, address, docker_socket, created_at FROM nodes WHERE token_hash = ?`, tokenHash)
}

// NodeIDForTokenHash implements agenthub.NodeLookup.
func (r *Nodes) NodeIDForTokenHash(tokenHash string) (string, error) {
	n, err := r.GetByTokenHash(tokenHash)
	if err != nil {
		return "", err
	}
	return n.ID, nil
}

func (r *Nodes) List() ([]*models.Node, error) {
	rows, err := r.db.Query(`SELECT id, name, token_hash, address, docker_socket, created_at FROM nodes ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*models.Node
	for rows.Next() {
		var n models.Node
		if err := rows.Scan(&n.ID, &n.Name, &n.TokenHash, &n.Address, &n.DockerSocket, &n.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &n)
	}
	return out, rows.Err()
}

func (r *Nodes) Delete(id string) error {
	res, err := r.db.Exec(`DELETE FROM nodes WHERE id = ?`, id)
	return checkRowsAffected(res, err)
}

func (r *Nodes) scanOne(query string, args ...any) (*models.Node, error) {
	row := r.db.QueryRow(query, args...)

	var n models.Node
	err := row.Scan(&n.ID, &n.Name, &n.TokenHash, &n.Address, &n.DockerSocket, &n.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &n, nil
}
