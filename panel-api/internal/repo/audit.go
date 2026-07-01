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

func (r *Audit) List(limit int) ([]*models.AuditEntry, error) {
	rows, err := r.db.Query(
		`SELECT id, actor_id, action, target, metadata, created_at FROM audit_log ORDER BY created_at DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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
