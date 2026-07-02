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

type adminServerResponse struct {
	serverResponse
	OwnerUsername string `json:"owner_username"`
}

// AdminListServers returns every server across all owners, with the owner's
// username, for the admin console's fleet view.
func (d Deps) AdminListServers(w http.ResponseWriter, r *http.Request) {
	servers, err := d.Servers.ListAll()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list servers")
		return
	}
	users, err := d.Users.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list users")
		return
	}
	usernames := make(map[string]string, len(users))
	for _, u := range users {
		usernames[u.ID] = u.Username
	}

	out := make([]adminServerResponse, 0, len(servers))
	for _, s := range servers {
		out = append(out, adminServerResponse{serverResponse: toServerResponse(s), OwnerUsername: usernames[s.OwnerID]})
	}
	writeJSON(w, http.StatusOK, out)
}

type transferOwnerRequest struct {
	OwnerID string `json:"owner_id"`
}

// AdminTransferServer reassigns a server to a different owner.
func (d Deps) AdminTransferServer(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "serverID")

	var req transferOwnerRequest
	if err := decodeJSON(r, &req); err != nil || req.OwnerID == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "owner_id is required")
		return
	}

	if _, err := d.Servers.GetByID(id); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "server not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load server")
		return
	}
	if _, err := d.Users.GetByID(req.OwnerID); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusBadRequest, "bad_request", "target user does not exist")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load target user")
		return
	}

	if err := d.Servers.SetOwner(id, req.OwnerID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to transfer server")
		return
	}
	d.audit(r, "server.transfer", id, req.OwnerID)
	w.WriteHeader(http.StatusNoContent)
}
