package httpapi

import (
	"errors"
	"net/http"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/auth"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/repo"
)

type adminResetPasswordRequest struct {
	Password string `json:"password"`
}

// AdminResetUserPassword sets a new password for any user (for support/recovery)
// and logs out all of that user's sessions, so a leaked or forgotten password
// can be rotated without the old one.
func (d Deps) AdminResetUserPassword(w http.ResponseWriter, r *http.Request) {
	userID := pathParam(r, "userID")

	var req adminResetPasswordRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "bad_request", "password must be at least 8 characters")
		return
	}

	if _, err := d.Users.GetByID(userID); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load user")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to hash password")
		return
	}
	if err := d.Users.SetPasswordHash(userID, hash); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to update password")
		return
	}
	// Invalidate every existing session for that user.
	_ = d.RefreshTokens.DeleteAllForUser(userID)
	d.audit(r, "admin.user.password_reset", userID, "")
	w.WriteHeader(http.StatusNoContent)
}

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
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

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

	// Guard against locking the instance out of the admin console. A demotion
	// (admin -> user) may not target yourself, and may not remove the last
	// remaining admin.
	if role == models.RoleUser {
		target, err := d.Users.GetByID(userID)
		if err != nil {
			if errors.Is(err, repo.ErrNotFound) {
				writeError(w, http.StatusNotFound, "not_found", "user not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to load user")
			return
		}
		if target.Role == models.RoleAdmin {
			if userID == claims.UserID {
				writeError(w, http.StatusConflict, "last_admin", "you cannot demote yourself; ask another admin to do it")
				return
			}
			admins, err := d.Users.CountAdmins()
			if err != nil {
				writeError(w, http.StatusInternalServerError, "internal_error", "failed to count admins")
				return
			}
			if admins <= 1 {
				writeError(w, http.StatusConflict, "last_admin", "cannot demote the last remaining admin")
				return
			}
		}
	}

	if err := d.Users.SetRole(userID, role); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to update role")
		return
	}

	// A role change must take effect promptly: revoke the target's sessions so
	// their next request re-authenticates with a fresh token carrying the new
	// role, instead of retaining the old role until the access token expires.
	_ = d.RefreshTokens.DeleteAllForUser(userID)

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

	// Refuse to delete the last remaining admin — otherwise the instance is
	// left with no way into the admin console short of editing the database.
	if target, err := d.Users.GetByID(userID); err == nil && target.Role == models.RoleAdmin {
		admins, err := d.Users.CountAdmins()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to count admins")
			return
		}
		if admins <= 1 {
			writeError(w, http.StatusConflict, "last_admin", "cannot delete the last remaining admin")
			return
		}
	}

	// Drop the user's databases on their nodes first — deleting the user
	// CASCADE-removes the server and database rows, which would otherwise strand
	// the real MariaDB databases with no pointer left to reclaim them.
	if d.ServerSvc != nil {
		d.ServerSvc.DeleteUserDatabases(userID)
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
