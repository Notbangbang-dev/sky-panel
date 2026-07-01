package httpapi

import (
	"net/http"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/auth"
)

type totpSetupResponse struct {
	Secret string `json:"secret"`
	URL    string `json:"url"`
}

type totpCodeRequest struct {
	Code string `json:"code"`
}

// TOTPSetup generates a fresh secret and stores it disabled until confirmed
// via TOTPConfirm. Calling setup again before confirming simply replaces the
// pending secret.
func (d Deps) TOTPSetup(w http.ResponseWriter, r *http.Request) {
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

	key, err := auth.NewTOTPSecret(user.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to generate totp secret")
		return
	}

	if err := d.Users.SetTOTP(user.ID, key.Secret(), false); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to store totp secret")
		return
	}

	writeJSON(w, http.StatusOK, totpSetupResponse{Secret: key.Secret(), URL: key.URL()})
}

func (d Deps) TOTPConfirm(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	var req totpCodeRequest
	if err := decodeJSON(r, &req); err != nil || req.Code == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "code is required")
		return
	}

	user, err := d.Users.GetByID(claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load user")
		return
	}

	if user.TOTPSecret == "" || !auth.VerifyTOTPCode(user.TOTPSecret, req.Code) {
		writeError(w, http.StatusUnauthorized, "invalid_totp", "invalid two-factor code")
		return
	}

	if err := d.Users.SetTOTP(user.ID, user.TOTPSecret, true); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to enable totp")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (d Deps) TOTPDisable(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	var req totpCodeRequest
	if err := decodeJSON(r, &req); err != nil || req.Code == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "code is required")
		return
	}

	user, err := d.Users.GetByID(claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load user")
		return
	}

	if !user.TOTPEnabled || !auth.VerifyTOTPCode(user.TOTPSecret, req.Code) {
		writeError(w, http.StatusUnauthorized, "invalid_totp", "invalid two-factor code")
		return
	}

	if err := d.Users.SetTOTP(user.ID, "", false); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to disable totp")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
