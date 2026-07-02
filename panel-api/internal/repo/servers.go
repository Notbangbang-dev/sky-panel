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

const serverColumns = `id, owner_id, node_id, egg_id, name, container_id, status, memory_bytes, cpu_limit, disk_bytes, variables_json, primary_port, backup_interval_hours, last_backup_at, suspended, status_message, created_at, updated_at`

func (r *Servers) Create(s *models.Server) error {
	varsJSON, err := json.Marshal(s.Variables)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(
		`INSERT INTO servers (id, owner_id, node_id, egg_id, name, container_id, status, memory_bytes, cpu_limit, disk_bytes, variables_json, primary_port, backup_interval_hours, last_backup_at, suspended, status_message, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.OwnerID, s.NodeID, s.EggID, s.Name, s.ContainerID, string(s.Status), s.MemoryBytes, s.CPULimit, s.DiskBytes, varsJSON, s.PrimaryPort, s.BackupIntervalHours, s.LastBackupAt, s.Suspended, s.StatusMessage, s.CreatedAt, s.UpdatedAt,
	)
	return err
}

func (r *Servers) GetByID(id string) (*models.Server, error) {
	row := r.db.QueryRow(`SELECT `+serverColumns+` FROM servers WHERE id = ?`, id)
	return scanServer(row)
}

func (r *Servers) ListByOwner(ownerID string) ([]*models.Server, error) {
	rows, err := r.db.Query(`SELECT `+serverColumns+` FROM servers WHERE owner_id = ? ORDER BY created_at`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanServerRows(rows)
}

func (r *Servers) ListAll() ([]*models.Server, error) {
	rows, err := r.db.Query(`SELECT ` + serverColumns + ` FROM servers ORDER BY created_at`)
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

// SetStatus updates the status and clears any stale status message (a healthy
// transition means the previous error, if any, no longer applies).
func (r *Servers) SetStatus(id string, status models.ServerStatus) error {
	res, err := r.db.Exec(`UPDATE servers SET status = ?, status_message = '', updated_at = ? WHERE id = ?`, string(status), time.Now().UTC(), id)
	return checkRowsAffected(res, err)
}

// SetEgg changes which egg (and therefore image/startup) a server runs. Used
// when reinstalling onto a different software stack.
func (r *Servers) SetEgg(id, eggID string) error {
	res, err := r.db.Exec(`UPDATE servers SET egg_id = ?, updated_at = ? WHERE id = ?`, eggID, time.Now().UTC(), id)
	return checkRowsAffected(res, err)
}

// SetStatusMessage updates only the human-readable status message, leaving the
// status itself unchanged — used to surface live provisioning phases (e.g.
// "Pulling image…", "Creating container…") while a server is still installing.
func (r *Servers) SetStatusMessage(id, message string) error {
	res, err := r.db.Exec(`UPDATE servers SET status_message = ?, updated_at = ? WHERE id = ?`, message, time.Now().UTC(), id)
	return checkRowsAffected(res, err)
}

// SetError marks a server errored and records why, so the UI can explain it.
func (r *Servers) SetError(id, message string) error {
	res, err := r.db.Exec(
		`UPDATE servers SET status = ?, status_message = ?, updated_at = ? WHERE id = ?`,
		string(models.StatusErrored), message, time.Now().UTC(), id,
	)
	return checkRowsAffected(res, err)
}

func (r *Servers) SetSuspended(id string, suspended bool) error {
	res, err := r.db.Exec(`UPDATE servers SET suspended = ?, updated_at = ? WHERE id = ?`, suspended, time.Now().UTC(), id)
	return checkRowsAffected(res, err)
}

// UpdateSettings applies user-editable settings (name, resource limits,
// egg variable overrides, and backup schedule) to an existing server.
func (r *Servers) UpdateSettings(id, name string, memoryBytes int64, cpuLimit int, diskBytes int64, variables map[string]string, backupIntervalHours int) error {
	varsJSON, err := json.Marshal(variables)
	if err != nil {
		return err
	}
	res, err := r.db.Exec(
		`UPDATE servers SET name = ?, memory_bytes = ?, cpu_limit = ?, disk_bytes = ?, variables_json = ?, backup_interval_hours = ?, updated_at = ? WHERE id = ?`,
		name, memoryBytes, cpuLimit, diskBytes, varsJSON, backupIntervalHours, time.Now().UTC(), id,
	)
	return checkRowsAffected(res, err)
}

// MarkBackedUp records that a backup completed for a server just now, so the
// scheduler knows when the next one is due.
func (r *Servers) MarkBackedUp(id string, at time.Time) error {
	res, err := r.db.Exec(`UPDATE servers SET last_backup_at = ? WHERE id = ?`, at, id)
	return checkRowsAffected(res, err)
}

// DueForBackup returns servers whose scheduled backup interval has elapsed
// (or which have never been backed up), for the background scheduler.
func (r *Servers) DueForBackup(now time.Time) ([]*models.Server, error) {
	rows, err := r.db.Query(
		`SELECT `+serverColumns+` FROM servers
		 WHERE backup_interval_hours > 0
		   AND (last_backup_at IS NULL OR last_backup_at <= ?)`,
		now.Add(-time.Hour), // coarse pre-filter; exact due-check is done per-row below
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	all, err := scanServerRows(rows)
	if err != nil {
		return nil, err
	}
	var due []*models.Server
	for _, s := range all {
		if s.LastBackupAt == nil || now.Sub(*s.LastBackupAt) >= time.Duration(s.BackupIntervalHours)*time.Hour {
			due = append(due, s)
		}
	}
	return due, nil
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
	var lastBackup sql.NullTime

	if err := row.Scan(&s.ID, &s.OwnerID, &s.NodeID, &s.EggID, &s.Name, &s.ContainerID, &status, &s.MemoryBytes, &s.CPULimit, &s.DiskBytes, &varsJSON, &s.PrimaryPort, &s.BackupIntervalHours, &lastBackup, &s.Suspended, &s.StatusMessage, &s.CreatedAt, &s.UpdatedAt); err != nil {
		return nil, err
	}

	s.Status = models.ServerStatus(status)
	if lastBackup.Valid {
		t := lastBackup.Time
		s.LastBackupAt = &t
	}
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
