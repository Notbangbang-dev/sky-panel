package httpapi

import (
	"errors"
	"net/http"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/auth"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/repo"
)

// AdminImpersonate ("view as user") issues a fresh session for the target user
// so an admin can experience the panel exactly as that user sees it. It's
// admin-only (gated by the route group) and audited. The client is expected to
// stash the admin's own session and restore it on exit — the server just mints
// the target session, the same way a normal login would.
func (d Deps) AdminImpersonate(w http.ResponseWriter, r *http.Request) {
	targetID := pathParam(r, "userID")

	if claims, ok := auth.FromContext(r.Context()); ok && claims.UserID == targetID {
		writeError(w, http.StatusBadRequest, "bad_request", "you are already yourself")
		return
	}

	target, err := d.Users.GetByID(targetID)
	if errors.Is(err, repo.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "user not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load user")
		return
	}

	d.audit(r, "admin.impersonate", target.ID, target.Username)
	d.issueTokenPair(w, target)
}
