package httpapi

import (
	"errors"
	"net/http"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/repo"
)

// AdminSuspendServer suspends a server: it's stopped and the owner can no
// longer start or use its console until an admin unsuspends it.
func (d Deps) AdminSuspendServer(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "serverID")
	if err := d.ServerSvc.SuspendServer(id); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "server not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	d.audit(r, "server.suspend", id, "")
	w.WriteHeader(http.StatusNoContent)
}

// AdminUnsuspendServer lifts a suspension, letting the owner control the
// server again (it is not auto-started).
func (d Deps) AdminUnsuspendServer(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "serverID")
	if err := d.ServerSvc.UnsuspendServer(id); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "server not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	d.audit(r, "server.unsuspend", id, "")
	w.WriteHeader(http.StatusNoContent)
}
