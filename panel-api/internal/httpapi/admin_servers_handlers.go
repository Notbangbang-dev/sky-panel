package httpapi

import (
	"errors"
	"fmt"
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

// AdminDeleteServer deletes any server (regardless of owner): the node is told
// to remove the container, the port allocation is freed, and the row is dropped.
func (d Deps) AdminDeleteServer(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "serverID")
	if _, err := d.Servers.GetByID(id); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "server not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load server")
		return
	}
	if err := d.ServerSvc.DeleteServer(id); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	d.audit(r, "server.admin_delete", id, "")
	w.WriteHeader(http.StatusNoContent)
}

type purgeServersRequest struct {
	ServerIDs []string `json:"server_ids"`
}

type purgeServersResult struct {
	Deleted int      `json:"deleted"`
	Failed  []string `json:"failed"`
}

// AdminPurgeServers bulk-deletes the given servers — a data-wipe tool. Each
// deletion is best-effort; the response reports how many were removed and which
// ids failed, so one bad server doesn't abort the whole purge.
func (d Deps) AdminPurgeServers(w http.ResponseWriter, r *http.Request) {
	var req purgeServersRequest
	if err := decodeJSON(r, &req); err != nil || len(req.ServerIDs) == 0 {
		writeError(w, http.StatusBadRequest, "bad_request", "server_ids is required")
		return
	}
	// Bound the batch so one request can't try to churn the whole fleet at once.
	if len(req.ServerIDs) > 200 {
		writeError(w, http.StatusBadRequest, "bad_request", "too many servers in one purge (max 200)")
		return
	}

	result := purgeServersResult{Failed: []string{}}
	for _, id := range req.ServerIDs {
		if err := d.ServerSvc.DeleteServer(id); err != nil {
			result.Failed = append(result.Failed, id)
			continue
		}
		result.Deleted++
	}
	d.audit(r, "server.purge", "", fmt.Sprintf("deleted %d, failed %d", result.Deleted, len(result.Failed)))
	writeJSON(w, http.StatusOK, result)
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
