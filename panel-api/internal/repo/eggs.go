package repo

import (
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
)

type Eggs struct {
	db *sql.DB
}

func NewEggs(db *sql.DB) *Eggs {
	return &Eggs{db: db}
}

func (r *Eggs) Create(e *models.Egg) error {
	varsJSON, err := json.Marshal(e.Variables)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(
		`INSERT INTO eggs (id, name, category, description, docker_image, startup, stop_command, variables_json, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.Name, e.Category, e.Description, e.DockerImage, e.Startup, e.StopCommand, varsJSON, e.CreatedAt,
	)
	return err
}

func (r *Eggs) GetByID(id string) (*models.Egg, error) {
	row := r.db.QueryRow(
		`SELECT id, name, category, description, docker_image, startup, stop_command, variables_json, created_at
		 FROM eggs WHERE id = ?`, id)
	return scanEgg(row)
}

func (r *Eggs) List() ([]*models.Egg, error) {
	rows, err := r.db.Query(
		`SELECT id, name, category, description, docker_image, startup, stop_command, variables_json, created_at
		 FROM eggs ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*models.Egg
	for rows.Next() {
		e, err := scanEggRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *Eggs) Delete(id string) error {
	res, err := r.db.Exec(`DELETE FROM eggs WHERE id = ?`, id)
	return checkRowsAffected(res, err)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanEgg(row rowScanner) (*models.Egg, error) {
	e, err := scanEggRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return e, err
}

func scanEggRow(row rowScanner) (*models.Egg, error) {
	var e models.Egg
	var varsJSON string

	if err := row.Scan(&e.ID, &e.Name, &e.Category, &e.Description, &e.DockerImage, &e.Startup, &e.StopCommand, &varsJSON, &e.CreatedAt); err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(varsJSON), &e.Variables); err != nil {
		return nil, err
	}
	return &e, nil
}
