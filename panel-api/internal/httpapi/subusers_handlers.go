package httpapi

import (
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/repo"
)

type subuserResponse struct {
	UserID      string   `json:"user_id"`
	Permissions []string `json:"permissions"`
}

func toSubuserResponse(s *models.Subuser) subuserResponse {
	return subuserResponse{UserID: s.UserID, Permissions: s.Permissions}
}

// ListSubusers, AddSubuser, and RemoveSubuser are all owner/admin-only —
// sharing management is intentionally not delegable to a subuser (avoids a
// subuser escalating by inviting themselves broader access).

func (d Deps) ListSubusers(w http.ResponseWriter, r *http.Request) {
	server := d.loadOwnedServer(w, r)
	if server == nil {
		return
	}

	subusers, err := d.Subusers.ListByServer(server.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list subusers")
		return
	}

	out := make([]subuserResponse, 0, len(subusers))
	for _, s := range subusers {
		out = append(out, toSubuserResponse(s))
	}
	writeJSON(w, http.StatusOK, out)
}

type addSubuserRequest struct {
	Username    string   `json:"username"`
	Permissions []string `json:"permissions"`
}

func (d Deps) AddSubuser(w http.ResponseWriter, r *http.Request) {
	server := d.loadOwnedServer(w, r)
	if server == nil {
		return
	}

	var req addSubuserRequest
	if err := decodeJSON(r, &req); err != nil || req.Username == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "username is required")
		return
	}
	for _, p := range req.Permissions {
		if !models.AllPermissions[p] {
			writeError(w, http.StatusBadRequest, "bad_request", "unknown permission: "+p)
			return
		}
	}

	invitee, err := d.Users.GetByUsername(req.Username)
	if errors.Is(err, repo.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "no user with that username")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to look up user")
		return
	}
	if invitee.ID == server.OwnerID {
		writeError(w, http.StatusBadRequest, "bad_request", "the owner already has full access")
		return
	}

	if err := d.Subusers.Create(uuid.NewString(), server.ID, invitee.ID, req.Permissions); err != nil {
		if errors.Is(err, repo.ErrDuplicate) {
			writeError(w, http.StatusConflict, "already_exists", "that user already has access to this server")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to add subuser")
		return
	}
	d.audit(r, "server.subuser.add", server.ID, req.Username)

	writeJSON(w, http.StatusCreated, subuserResponse{UserID: invitee.ID, Permissions: req.Permissions})
}

func (d Deps) RemoveSubuser(w http.ResponseWriter, r *http.Request) {
	server := d.loadOwnedServer(w, r)
	if server == nil {
		return
	}

	userID := pathParam(r, "userID")
	if err := d.Subusers.Delete(server.ID, userID); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "subuser not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to remove subuser")
		return
	}
	d.audit(r, "server.subuser.remove", server.ID, userID)
	w.WriteHeader(http.StatusNoContent)
}
