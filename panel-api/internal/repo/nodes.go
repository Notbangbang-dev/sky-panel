package repo

import (
	"database/sql"
	"errors"
	"time"

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
		`INSERT INTO nodes (id, name, token_hash, token, expires_at, address, docker_socket, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		n.ID, n.Name, n.TokenHash, n.Token, n.ExpiresAt, n.Address, n.DockerSocket, n.CreatedAt,
	)
	if isUniqueViolation(err) {
		return ErrDuplicate
	}
	return err
}

func (r *Nodes) GetByID(id string) (*models.Node, error) {
	return r.scanOne(`SELECT id, name, token_hash, token, expires_at, address, docker_socket, created_at FROM nodes WHERE id = ?`, id)
}

func (r *Nodes) GetByTokenHash(tokenHash string) (*models.Node, error) {
	return r.scanOne(`SELECT id, name, token_hash, token, expires_at, address, docker_socket, created_at FROM nodes WHERE token_hash = ?`, tokenHash)
}

// NodeIDForTokenHash implements agenthub.NodeLookup's simple identity check.
func (r *Nodes) NodeIDForTokenHash(tokenHash string) (string, error) {
	n, err := r.GetByTokenHash(tokenHash)
	if err != nil {
		return "", err
	}
	return n.ID, nil
}

// AuthenticateNode implements agenthub.NodeLookup: it resolves a node's
// identity and returns everything the handler needs to validate the hello
// (raw token for later HMAC verification, expiry) without depending on
// panel-api's models type from the agenthub package.
func (r *Nodes) AuthenticateNode(tokenHash string) (nodeID, token string, expiresAt time.Time, err error) {
	n, err := r.GetByTokenHash(tokenHash)
	if err != nil {
		return "", "", time.Time{}, err
	}
	return n.ID, n.Token, n.ExpiresAt, nil
}

func (r *Nodes) List() ([]*models.Node, error) {
	rows, err := r.db.Query(`SELECT id, name, token_hash, token, expires_at, address, docker_socket, created_at FROM nodes ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*models.Node
	for rows.Next() {
		n, err := scanNodeRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *Nodes) Delete(id string) error {
	res, err := r.db.Exec(`DELETE FROM nodes WHERE id = ?`, id)
	return checkRowsAffected(res, err)
}

// RotateToken replaces a node's token (hash + raw value) and expiry,
// immediately invalidating the old one.
func (r *Nodes) RotateToken(id, newTokenHash, newToken string, expiresAt time.Time) error {
	res, err := r.db.Exec(
		`UPDATE nodes SET token_hash = ?, token = ?, expires_at = ? WHERE id = ?`,
		newTokenHash, newToken, expiresAt, id,
	)
	return checkRowsAffected(res, err)
}

func (r *Nodes) scanOne(query string, args ...any) (*models.Node, error) {
	n, err := scanNodeRow(r.db.QueryRow(query, args...))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return n, err
}

func scanNodeRow(row rowScanner) (*models.Node, error) {
	var n models.Node
	if err := row.Scan(&n.ID, &n.Name, &n.TokenHash, &n.Token, &n.ExpiresAt, &n.Address, &n.DockerSocket, &n.CreatedAt); err != nil {
		return nil, err
	}
	return &n, nil
}
