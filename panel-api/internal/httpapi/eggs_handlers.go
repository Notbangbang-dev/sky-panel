package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/repo"
)

type createEggRequest struct {
	Name        string               `json:"name"`
	Category    string               `json:"category,omitempty"`
	Description string               `json:"description,omitempty"`
	DockerImage string               `json:"docker_image"`
	Startup     string               `json:"startup"`
	StopCommand string               `json:"stop_command,omitempty"`
	Variables   []models.EggVariable `json:"variables,omitempty"`
}

func toEggResponse(e *models.Egg) *models.Egg { return e }

func (d Deps) CreateEgg(w http.ResponseWriter, r *http.Request) {
	var req createEggRequest
	if err := decodeJSON(r, &req); err != nil || req.Name == "" || req.DockerImage == "" || req.Startup == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "name, docker_image and startup are required")
		return
	}

	egg := &models.Egg{
		ID:          uuid.NewString(),
		Name:        req.Name,
		Category:    req.Category,
		Description: req.Description,
		DockerImage: req.DockerImage,
		Startup:     req.Startup,
		StopCommand: req.StopCommand,
		Variables:   req.Variables,
		CreatedAt:   time.Now().UTC(),
	}

	if err := d.Eggs.Create(egg); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to create egg")
		return
	}
	d.audit(r, "egg.create", egg.ID, egg.Name)

	writeJSON(w, http.StatusCreated, toEggResponse(egg))
}

func (d Deps) ListEggs(w http.ResponseWriter, r *http.Request) {
	eggs, err := d.Eggs.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list eggs")
		return
	}
	writeJSON(w, http.StatusOK, eggs)
}

func (d Deps) GetEgg(w http.ResponseWriter, r *http.Request) {
	egg, err := d.Eggs.GetByID(pathParam(r, "eggID"))
	if errors.Is(err, repo.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "egg not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load egg")
		return
	}
	writeJSON(w, http.StatusOK, toEggResponse(egg))
}

func (d Deps) DeleteEgg(w http.ResponseWriter, r *http.Request) {
	eggID := pathParam(r, "eggID")
	if err := d.Eggs.Delete(eggID); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "egg not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete egg")
		return
	}
	d.audit(r, "egg.delete", eggID, "")
	w.WriteHeader(http.StatusNoContent)
}
