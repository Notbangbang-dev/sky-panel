package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

func registerAndGetAccessToken(t *testing.T, r http.Handler, email, username string) string {
	t.Helper()
	rec := doJSON(t, r, http.MethodPost, "/api/v1/auth/register", registerRequest{
		Email: email, Username: username, Password: "password123",
	})
	var tokens tokenPairResponse
	decodeBody(t, rec, &tokens)
	return tokens.AccessToken
}

func TestTOTPSetupConfirmAndLoginFlow(t *testing.T) {
	r := newTestRouter(t)

	access := registerAndGetAccessToken(t, r, "totp@example.com", "totpuser")

	setupReq := httptest.NewRequest(http.MethodPost, "/api/v1/me/totp/setup", nil)
	setupReq.Header.Set("Authorization", "Bearer "+access)
	setupRec := httptest.NewRecorder()
	r.ServeHTTP(setupRec, setupReq)
	if setupRec.Code != http.StatusOK {
		t.Fatalf("expected 200 from setup, got %d: %s", setupRec.Code, setupRec.Body.String())
	}

	var setup totpSetupResponse
	decodeBody(t, setupRec, &setup)
	if setup.Secret == "" {
		t.Fatal("expected a non-empty TOTP secret")
	}

	code, err := totp.GenerateCode(setup.Secret, time.Now())
	if err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}

	confirmReq := jsonRequest(t, http.MethodPost, "/api/v1/me/totp/confirm", totpCodeRequest{Code: code})
	confirmReq.Header.Set("Authorization", "Bearer "+access)
	confirmRec := httptest.NewRecorder()
	r.ServeHTTP(confirmRec, confirmReq)
	if confirmRec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 from confirm, got %d: %s", confirmRec.Code, confirmRec.Body.String())
	}

	// Login without a TOTP code should now be rejected as totp_required.
	loginRec := doJSON(t, r, http.MethodPost, "/api/v1/auth/login", loginRequest{
		Email: "totp@example.com", Password: "password123",
	})
	if loginRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without totp code, got %d", loginRec.Code)
	}

	// Login with a fresh valid code should succeed.
	freshCode, err := totp.GenerateCode(setup.Secret, time.Now())
	if err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}
	loginWithTOTPRec := doJSON(t, r, http.MethodPost, "/api/v1/auth/login", loginRequest{
		Email: "totp@example.com", Password: "password123", TOTPCode: freshCode,
	})
	if loginWithTOTPRec.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid totp code, got %d: %s", loginWithTOTPRec.Code, loginWithTOTPRec.Body.String())
	}
}
