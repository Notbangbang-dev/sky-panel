package httpapi

import (
	"errors"
	"net/http"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/auth"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/repo"
)

func (d Deps) AdminListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := d.Users.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list users")
		return
	}

	out := make([]userResponse, 0, len(users))
	for _, u := range users {
		out = append(out, toUserResponse(u))
	}
	writeJSON(w, http.StatusOK, out)
}

type setUserRoleRequest struct {
	Role string `json:"role"`
}

func (d Deps) AdminSetUserRole(w http.ResponseWriter, r *http.Request) {
	var req setUserRoleRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	role := models.Role(req.Role)
	if role != models.RoleAdmin && role != models.RoleUser {
		writeError(w, http.StatusBadRequest, "bad_request", "role must be 'admin' or 'user'")
		return
	}

	userID := pathParam(r, "userID")
	if err := d.Users.SetRole(userID, role); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to update role")
		return
	}

	d.audit(r, "user.set_role", userID, req.Role)
	w.WriteHeader(http.StatusNoContent)
}

func (d Deps) AdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	userID := pathParam(r, "userID")
	if userID == claims.UserID {
		writeError(w, http.StatusBadRequest, "bad_request", "you cannot delete your own account")
		return
	}

	if err := d.Users.Delete(userID); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete user")
		return
	}

	d.audit(r, "user.delete", userID, "")
	w.WriteHeader(http.StatusNoContent)
}
