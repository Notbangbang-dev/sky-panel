package repo

import (
	"database/sql"
	"time"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
)

type Schedules struct {
	db *sql.DB
}

func NewSchedules(db *sql.DB) *Schedules {
	return &Schedules{db: db}
}

const scheduleColumns = `id, server_id, name, action, payload, interval_minutes, enabled, last_run_at, created_at`

func (r *Schedules) Create(s *models.Schedule) error {
	_, err := r.db.Exec(
		`INSERT INTO server_schedules (id, server_id, name, action, payload, interval_minutes, enabled, last_run_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.ServerID, s.Name, s.Action, s.Payload, s.IntervalMinutes, s.Enabled, s.LastRunAt, s.CreatedAt,
	)
	return err
}

func (r *Schedules) ListByServer(serverID string) ([]*models.Schedule, error) {
	rows, err := r.db.Query(`SELECT `+scheduleColumns+` FROM server_schedules WHERE server_id = ? ORDER BY created_at`, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSchedules(rows)
}

// Due returns enabled schedules whose interval has elapsed since their last run
// (or which have never run), across all servers, for the background scheduler.
func (r *Schedules) Due(now time.Time) ([]*models.Schedule, error) {
	rows, err := r.db.Query(`SELECT ` + scheduleColumns + ` FROM server_schedules WHERE enabled = 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	all, err := scanSchedules(rows)
	if err != nil {
		return nil, err
	}
	var due []*models.Schedule
	for _, s := range all {
		interval := time.Duration(s.IntervalMinutes) * time.Minute
		if interval <= 0 {
			continue
		}
		if s.LastRunAt == nil || now.Sub(*s.LastRunAt) >= interval {
			due = append(due, s)
		}
	}
	return due, nil
}

func (r *Schedules) MarkRun(id string, at time.Time) error {
	_, err := r.db.Exec(`UPDATE server_schedules SET last_run_at = ? WHERE id = ?`, at, id)
	return err
}

func (r *Schedules) SetEnabled(id, serverID string, enabled bool) error {
	res, err := r.db.Exec(`UPDATE server_schedules SET enabled = ? WHERE id = ? AND server_id = ?`, enabled, id, serverID)
	return checkRowsAffected(res, err)
}

func (r *Schedules) Delete(id, serverID string) error {
	res, err := r.db.Exec(`DELETE FROM server_schedules WHERE id = ? AND server_id = ?`, id, serverID)
	return checkRowsAffected(res, err)
}

func scanSchedules(rows *sql.Rows) ([]*models.Schedule, error) {
	var out []*models.Schedule
	for rows.Next() {
		var s models.Schedule
		var lastRun sql.NullTime
		if err := rows.Scan(&s.ID, &s.ServerID, &s.Name, &s.Action, &s.Payload, &s.IntervalMinutes, &s.Enabled, &lastRun, &s.CreatedAt); err != nil {
			return nil, err
		}
		if lastRun.Valid {
			t := lastRun.Time
			s.LastRunAt = &t
		}
		out = append(out, &s)
	}
	return out, rows.Err()
}
