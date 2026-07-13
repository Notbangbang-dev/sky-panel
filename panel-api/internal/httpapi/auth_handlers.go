package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/auth"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/repo"
)

type registerRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	TOTPCode string `json:"totp_code,omitempty"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type tokenPairResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	User         userResponse `json:"user"`
}

type userResponse struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	Username    string `json:"username"`
	Role        string `json:"role"`
	TOTPEnabled bool   `json:"totp_enabled"`
	Coins       int64  `json:"coins"`
}

func toUserResponse(u *models.User) userResponse {
	return userResponse{
		ID:          u.ID,
		Email:       u.Email,
		Username:    u.Username,
		Role:        string(u.Role),
		TOTPEnabled: u.TOTPEnabled,
		Coins:       u.Coins,
	}
}

func (d Deps) Register(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	var req registerRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	// Normalize identifiers: emails are lower-cased + trimmed so case variants
	// can't create duplicate accounts (and so login isn't case-sensitive);
	// usernames keep their display case but are trimmed and length-bounded.
	req.Email = normalizeEmail(req.Email)
	req.Username = strings.TrimSpace(req.Username)
	if req.Email == "" || req.Username == "" || len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "invalid_input", "email, username and a password of at least 8 characters are required")
		return
	}
	if len(req.Email) > 254 || len(req.Username) > 32 || len(req.Password) > 1024 {
		writeError(w, http.StatusBadRequest, "invalid_input", "email, username, or password is too long")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to hash password")
		return
	}

	count, err := d.Users.Count()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to check existing users")
		return
	}

	// The very first account always gets through regardless of the toggle —
	// otherwise a freshly deployed panel with registration disabled by
	// default settings would have no way to ever create an admin.
	if count > 0 && !d.registrationEnabled() {
		writeError(w, http.StatusForbidden, "registration_disabled", "registration is currently disabled")
		return
	}

	role := models.RoleUser
	if count == 0 {
		role = models.RoleAdmin
	}

	now := time.Now().UTC()
	user := &models.User{
		ID:           uuid.NewString(),
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: hash,
		Role:         role,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := d.Users.Create(user); err != nil {
		if errors.Is(err, repo.ErrDuplicate) {
			writeError(w, http.StatusConflict, "already_exists", "email or username already taken")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to create user")
		return
	}

	d.issueTokenPair(w, user)
}

// registrationEnabled treats an unset "registration_enabled" setting as
// enabled, so upgrading an existing install never silently locks out new
// signups it never opted into blocking.
func (d Deps) registrationEnabled() bool {
	value, found, err := d.Settings.Get("registration_enabled")
	if err != nil || !found {
		return true
	}
	return value != "false"
}

// RegistrationStatus is a public, unauthenticated endpoint so the frontend
// can hide the register flow proactively instead of letting someone fill
// out the form and hit a 403.
func (d Deps) RegistrationStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"enabled": d.registrationEnabled()})
}

func (d Deps) Login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	req.Email = normalizeEmail(req.Email)

	user, err := d.Users.GetByEmail(req.Email)
	if errors.Is(err, repo.ErrNotFound) {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to look up user")
		return
	}

	if !auth.VerifyPassword(user.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
		return
	}

	if user.TOTPEnabled {
		if req.TOTPCode == "" {
			writeError(w, http.StatusUnauthorized, "totp_required", "two-factor code required")
			return
		}
		if !auth.VerifyTOTPCode(user.TOTPSecret, req.TOTPCode) {
			writeError(w, http.StatusUnauthorized, "invalid_totp", "invalid two-factor code")
			return
		}
	}

	d.issueTokenPair(w, user)
}

func (d Deps) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := decodeJSON(r, &req); err != nil || req.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "refresh_token is required")
		return
	}

	oldHash := auth.HashToken(req.RefreshToken)
	userID, err := d.RefreshTokens.UserIDForValidToken(oldHash)
	if errors.Is(err, repo.ErrNotFound) {
		writeError(w, http.StatusUnauthorized, "invalid_refresh_token", "refresh token is invalid or expired")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to validate refresh token")
		return
	}

	// Rotate first: the presented refresh token is single-use. Deleting it
	// before any further step that can fail (e.g. loading the user) guarantees
	// the "single-use" invariant holds even on an error path — otherwise a
	// transient failure would leave the token valid and infinitely replayable.
	_ = d.RefreshTokens.DeleteByHash(oldHash)

	user, err := d.Users.GetByID(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load user")
		return
	}

	d.issueTokenPair(w, user)
}

func (d Deps) Logout(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := decodeJSON(r, &req); err == nil && req.RefreshToken != "" {
		_ = d.RefreshTokens.DeleteByHash(auth.HashToken(req.RefreshToken))
	}
	w.WriteHeader(http.StatusNoContent)
}

func (d Deps) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	user, err := d.Users.GetByID(claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load user")
		return
	}

	writeJSON(w, http.StatusOK, toUserResponse(user))
}

func (d Deps) issueTokenPair(w http.ResponseWriter, user *models.User) {
	access, err := d.JWT.NewAccessToken(user.ID, string(user.Role))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to issue access token")
		return
	}

	rawRefresh, err := auth.NewOpaqueToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to issue refresh token")
		return
	}

	expiresAt := time.Now().UTC().Add(d.RefreshTTL)
	if err := d.RefreshTokens.Create(uuid.NewString(), user.ID, auth.HashToken(rawRefresh), expiresAt); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to persist refresh token")
		return
	}

	writeJSON(w, http.StatusOK, tokenPairResponse{
		AccessToken:  access,
		RefreshToken: rawRefresh,
		User:         toUserResponse(user),
	})
}
