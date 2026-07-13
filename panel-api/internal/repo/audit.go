package repo

import (
	"database/sql"
	"time"

	"github.com/google/uuid"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
)

type Audit struct {
	db *sql.DB
}

func NewAudit(db *sql.DB) *Audit {
	return &Audit{db: db}
}

func (r *Audit) Record(actorID, action, target, metadata string) error {
	_, err := r.db.Exec(
		`INSERT INTO audit_log (id, actor_id, action, target, metadata, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		uuid.NewString(), actorID, action, target, metadata, time.Now().UTC(),
	)
	return err
}

// PruneOlderThan deletes audit entries older than cutoff, returning the number
// removed. Keeps the audit log (and its created_at index) bounded on the
// single-writer DB while retaining a generous recent window.
func (r *Audit) PruneOlderThan(cutoff time.Time) (int64, error) {
	res, err := r.db.Exec(`DELETE FROM audit_log WHERE created_at < ?`, cutoff)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *Audit) List(limit int) ([]*models.AuditEntry, error) {
	rows, err := r.db.Query(
		`SELECT id, actor_id, action, target, metadata, created_at FROM audit_log ORDER BY created_at DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAuditRows(rows)
}

// ListByTarget returns the most recent audit entries whose target matches
// the given id (used for a server's per-server activity feed).
func (r *Audit) ListByTarget(target string, limit int) ([]*models.AuditEntry, error) {
	rows, err := r.db.Query(
		`SELECT id, actor_id, action, target, metadata, created_at FROM audit_log WHERE target = ? ORDER BY created_at DESC LIMIT ?`,
		target, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAuditRows(rows)
}

func scanAuditRows(rows *sql.Rows) ([]*models.AuditEntry, error) {
	var out []*models.AuditEntry
	for rows.Next() {
		var e models.AuditEntry
		if err := rows.Scan(&e.ID, &e.ActorID, &e.Action, &e.Target, &e.Metadata, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &e)
	}
	return out, rows.Err()
}
