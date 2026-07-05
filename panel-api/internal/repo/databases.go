package repo

import (
	"database/sql"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
)

// Databases records the MariaDB databases provisioned for users. The row is the
// panel's source of truth (name, credentials, which node/server); the actual
// database lives on the node.
type Databases struct {
	db *sql.DB
}

func NewDatabases(db *sql.DB) *Databases {
	return &Databases{db: db}
}

const databaseColumns = `id, owner_id, server_id, node_id, name, username, password, host, port, created_at`

func scanDatabase(row rowScanner) (*models.Database, error) {
	var d models.Database
	err := row.Scan(&d.ID, &d.OwnerID, &d.ServerID, &d.NodeID, &d.Name, &d.Username, &d.Password, &d.Host, &d.Port, &d.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &d, err
}

func (r *Databases) Create(d *models.Database) error {
	_, err := r.db.Exec(
		`INSERT INTO databases (`+databaseColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		d.ID, d.OwnerID, d.ServerID, d.NodeID, d.Name, d.Username, d.Password, d.Host, d.Port, d.CreatedAt,
	)
	if isUniqueViolation(err) {
		return ErrDuplicate
	}
	return err
}

func (r *Databases) GetByID(id string) (*models.Database, error) {
	return scanDatabase(r.db.QueryRow(`SELECT `+databaseColumns+` FROM databases WHERE id = ?`, id))
}

func (r *Databases) ListByServer(serverID string) ([]*models.Database, error) {
	rows, err := r.db.Query(`SELECT `+databaseColumns+` FROM databases WHERE server_id = ? ORDER BY created_at`, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDatabaseRows(rows)
}

// ListByOwner returns every database a user owns across all their servers, used
// to drop them on their nodes before the user (and their servers) are deleted.
func (r *Databases) ListByOwner(ownerID string) ([]*models.Database, error) {
	rows, err := r.db.Query(`SELECT `+databaseColumns+` FROM databases WHERE owner_id = ? ORDER BY created_at`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDatabaseRows(rows)
}

func scanDatabaseRows(rows *sql.Rows) ([]*models.Database, error) {
	out := make([]*models.Database, 0)
	for rows.Next() {
		d, err := scanDatabase(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// CountByOwner returns how many databases a user owns across all their servers,
// used to enforce the databases quota.
func (r *Databases) CountByOwner(ownerID string) (int, error) {
	var n int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM databases WHERE owner_id = ?`, ownerID).Scan(&n)
	return n, err
}

// NameExistsOnNode reports whether a database name is already taken on a node —
// databases share one MariaDB per node, so names must be globally unique there.
func (r *Databases) NameExistsOnNode(nodeID, name string) (bool, error) {
	var n int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM databases WHERE node_id = ? AND name = ?`, nodeID, name).Scan(&n)
	return n > 0, err
}

func (r *Databases) Delete(id string) error {
	return checkRowsAffected(r.db.Exec(`DELETE FROM databases WHERE id = ?`, id))
}
