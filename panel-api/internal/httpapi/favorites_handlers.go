package httpapi

import (
	"net/http"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/auth"
)

// FavoriteServer stars a server for the caller. Requires the caller to be able
// to see the server (owner, admin, or a subuser) so favorites can't be used to
// probe for server IDs.
func (d Deps) FavoriteServer(w http.ResponseWriter, r *http.Request) {
	server := d.loadServerWithPermission(w, r, "")
	if server == nil {
		return
	}
	claims, _ := auth.FromContext(r.Context())
	if err := d.Favorites.Add(claims.UserID, server.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to favorite server")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// UnfavoriteServer removes a star for the caller.
func (d Deps) UnfavoriteServer(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}
	if err := d.Favorites.Remove(claims.UserID, pathParam(r, "serverID")); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to unfavorite server")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListFavorites returns the caller's favorited server IDs.
func (d Deps) ListFavorites(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}
	ids, err := d.Favorites.ListByUser(claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list favorites")
		return
	}
	if ids == nil {
		ids = []string{}
	}
	writeJSON(w, http.StatusOK, ids)
}
