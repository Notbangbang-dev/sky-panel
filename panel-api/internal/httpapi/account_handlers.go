package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/auth"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/repo"
)

// apiKeyPrefix marks a personal API key so RequireAuth can tell it apart from
// a JWT access token before doing a DB lookup.
const apiKeyPrefix = "sky_"

// resolveAPIKey is the auth.ClaimsResolver: it maps a raw personal API key to
// its owner's Claims. Wired into RequireAuth so API keys authenticate the same
// endpoints as a logged-in session.
func (d Deps) resolveAPIKey(raw string) (*auth.Claims, bool) {
	if !strings.HasPrefix(raw, apiKeyPrefix) {
		return nil, false
	}
	hash := auth.HashToken(raw)
	userID, err := d.APIKeys.UserIDForKeyHash(hash)
	if err != nil {
		return nil, false
	}
	user, err := d.Users.GetByID(userID)
	if err != nil {
		return nil, false
	}
	d.APIKeys.TouchLastUsed(hash)
	return &auth.Claims{UserID: user.ID, Role: string(user.Role)}, true
}

// ---- Change password ----

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (d Deps) ChangePassword(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	var req changePasswordRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	if len(req.NewPassword) < 8 {
		writeError(w, http.StatusBadRequest, "bad_request", "new password must be at least 8 characters")
		return
	}

	user, err := d.Users.GetByID(claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load user")
		return
	}
	if !auth.VerifyPassword(user.PasswordHash, req.CurrentPassword) {
		writeError(w, http.StatusForbidden, "wrong_password", "current password is incorrect")
		return
	}

	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to hash password")
		return
	}
	if err := d.Users.SetPasswordHash(claims.UserID, hash); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to update password")
		return
	}

	// Log every other session out — a password change should invalidate them.
	_ = d.RefreshTokens.DeleteAllForUser(claims.UserID)
	d.audit(r, "account.password_change", claims.UserID, "")
	w.WriteHeader(http.StatusNoContent)
}

// ---- Active sessions ----

type sessionResponse struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	ExpiresAt string `json:"expires_at"`
	Current   bool   `json:"current"`
}

func (d Deps) ListSessions(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}
	// The caller can pass its own refresh token so we can flag "this device".
	currentHash := ""
	if rt := r.URL.Query().Get("current"); rt != "" {
		currentHash = auth.HashToken(rt)
	}

	sessions, err := d.RefreshTokens.ListByUser(claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list sessions")
		return
	}
	out := make([]sessionResponse, 0, len(sessions))
	for _, s := range sessions {
		resp := sessionResponse{ID: s.ID, CreatedAt: s.CreatedAt.Format(rfc3339), ExpiresAt: s.ExpiresAt.Format(rfc3339)}
		if currentHash != "" {
			if h, err := d.RefreshTokens.HashForID(s.ID, claims.UserID); err == nil && h == currentHash {
				resp.Current = true
			}
		}
		out = append(out, resp)
	}
	writeJSON(w, http.StatusOK, out)
}

func (d Deps) RevokeSession(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}
	if err := d.RefreshTokens.DeleteByIDForUser(pathParam(r, "sessionID"), claims.UserID); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "session not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to revoke session")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type revokeOthersRequest struct {
	CurrentRefreshToken string `json:"current_refresh_token"`
}

func (d Deps) RevokeOtherSessions(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}
	var req revokeOthersRequest
	if err := decodeJSON(r, &req); err != nil || req.CurrentRefreshToken == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "current_refresh_token is required")
		return
	}
	if err := d.RefreshTokens.DeleteOthersForUser(claims.UserID, auth.HashToken(req.CurrentRefreshToken)); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to revoke sessions")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- API keys ----

type apiKeyResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	LastUsedAt string `json:"last_used_at,omitempty"`
	CreatedAt  string `json:"created_at"`
}

func (d Deps) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}
	keys, err := d.APIKeys.ListByUser(claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list API keys")
		return
	}
	out := make([]apiKeyResponse, 0, len(keys))
	for _, k := range keys {
		resp := apiKeyResponse{ID: k.ID, Name: k.Name, CreatedAt: k.CreatedAt.Format(rfc3339)}
		if k.LastUsedAt != nil {
			resp.LastUsedAt = k.LastUsedAt.Format(rfc3339)
		}
		out = append(out, resp)
	}
	writeJSON(w, http.StatusOK, out)
}

type createAPIKeyRequest struct {
	Name string `json:"name"`
}

func (d Deps) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}
	var req createAPIKeyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "API key"
	}

	raw, err := auth.NewOpaqueToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to generate key")
		return
	}
	key := apiKeyPrefix + raw
	if err := d.APIKeys.Create(uuid.NewString(), claims.UserID, name, auth.HashToken(key)); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to create key")
		return
	}
	d.audit(r, "account.api_key_create", claims.UserID, name)
	// The raw key is returned exactly once.
	writeJSON(w, http.StatusCreated, map[string]string{"name": name, "key": key})
}

func (d Deps) DeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}
	if err := d.APIKeys.DeleteByIDForUser(pathParam(r, "keyID"), claims.UserID); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "API key not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete key")
		return
	}
	d.audit(r, "account.api_key_delete", claims.UserID, "")
	w.WriteHeader(http.StatusNoContent)
}
