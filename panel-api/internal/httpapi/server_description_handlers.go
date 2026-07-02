package httpapi

import (
	"net/http"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
)

const maxDescriptionLen = 500

type setDescriptionRequest struct {
	Description string `json:"description"`
}

// SetServerDescription updates a server's free-text note. Unlike UpdateServer
// it does NOT re-provision the container — it's just metadata. Requires the
// settings permission (owner, admin, or a subuser granted it).
func (d Deps) SetServerDescription(w http.ResponseWriter, r *http.Request) {
	server := d.loadServerWithPermission(w, r, models.PermSettings)
	if server == nil {
		return
	}

	var req setDescriptionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	if len(req.Description) > maxDescriptionLen {
		writeError(w, http.StatusBadRequest, "bad_request", "description is too long")
		return
	}

	if err := d.Servers.SetDescription(server.ID, req.Description); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to update description")
		return
	}
	d.audit(r, "server.description", server.ID, "")
	w.WriteHeader(http.StatusNoContent)
}
