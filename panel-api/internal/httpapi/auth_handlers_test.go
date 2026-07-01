package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/auth"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/repo"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/store"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/wshub"
)

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()

	name := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", name)

	db, err := store.Open(dsn)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	deps := Deps{
		Users:         repo.NewUsers(db),
		RefreshTokens: repo.NewRefreshTokens(db),
		JWT:           auth.NewManager("test-secret", 15*time.Minute),
		Hub:           wshub.NewHub(),
		RefreshTTL:    30 * 24 * time.Hour,
	}

	return NewRouter(deps)
}

func doJSON(t *testing.T, r http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}

	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func jsonRequest(t *testing.T, method, path string, body any) *http.Request {
	t.Helper()

	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}

	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), v); err != nil {
		t.Fatalf("decode response body %q: %v", rec.Body.String(), err)
	}
}

func TestRegisterFirstUserBecomesAdmin(t *testing.T) {
	r := newTestRouter(t)

	rec := doJSON(t, r, http.MethodPost, "/api/v1/auth/register", registerRequest{
		Email: "admin@example.com", Username: "admin", Password: "password123",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp tokenPairResponse
	decodeBody(t, rec, &resp)

	if resp.User.Role != "admin" {
		t.Errorf("expected first registered user to be admin, got %q", resp.User.Role)
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Error("expected non-empty access and refresh tokens")
	}
}

func TestRegisterSecondUserIsRegularUser(t *testing.T) {
	r := newTestRouter(t)

	doJSON(t, r, http.MethodPost, "/api/v1/auth/register", registerRequest{
		Email: "admin@example.com", Username: "admin", Password: "password123",
	})

	rec := doJSON(t, r, http.MethodPost, "/api/v1/auth/register", registerRequest{
		Email: "second@example.com", Username: "second", Password: "password123",
	})

	var resp tokenPairResponse
	decodeBody(t, rec, &resp)

	if resp.User.Role != "user" {
		t.Errorf("expected second registered user to be a regular user, got %q", resp.User.Role)
	}
}

func TestRegisterDuplicateEmailConflicts(t *testing.T) {
	r := newTestRouter(t)

	doJSON(t, r, http.MethodPost, "/api/v1/auth/register", registerRequest{
		Email: "dup@example.com", Username: "one", Password: "password123",
	})
	rec := doJSON(t, r, http.MethodPost, "/api/v1/auth/register", registerRequest{
		Email: "dup@example.com", Username: "two", Password: "password123",
	})

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", rec.Code)
	}
}

func TestLoginSuccess(t *testing.T) {
	r := newTestRouter(t)

	doJSON(t, r, http.MethodPost, "/api/v1/auth/register", registerRequest{
		Email: "login@example.com", Username: "login", Password: "password123",
	})

	rec := doJSON(t, r, http.MethodPost, "/api/v1/auth/login", loginRequest{
		Email: "login@example.com", Password: "password123",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestLoginWrongPassword(t *testing.T) {
	r := newTestRouter(t)

	doJSON(t, r, http.MethodPost, "/api/v1/auth/register", registerRequest{
		Email: "login2@example.com", Username: "login2", Password: "password123",
	})

	rec := doJSON(t, r, http.MethodPost, "/api/v1/auth/login", loginRequest{
		Email: "login2@example.com", Password: "wrong-password",
	})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestMeRequiresAuth(t *testing.T) {
	r := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without a token, got %d", rec.Code)
	}
}

func TestMeWithValidToken(t *testing.T) {
	r := newTestRouter(t)

	rec := doJSON(t, r, http.MethodPost, "/api/v1/auth/register", registerRequest{
		Email: "me@example.com", Username: "me", Password: "password123",
	})
	var tokens tokenPairResponse
	decodeBody(t, rec, &tokens)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	meRec := httptest.NewRecorder()
	r.ServeHTTP(meRec, req)

	if meRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", meRec.Code, meRec.Body.String())
	}

	var user userResponse
	decodeBody(t, meRec, &user)
	if user.Email != "me@example.com" {
		t.Errorf("unexpected user in /me response: %+v", user)
	}
}

func TestRefreshRotatesToken(t *testing.T) {
	r := newTestRouter(t)

	rec := doJSON(t, r, http.MethodPost, "/api/v1/auth/register", registerRequest{
		Email: "refresh@example.com", Username: "refresh", Password: "password123",
	})
	var tokens tokenPairResponse
	decodeBody(t, rec, &tokens)

	refreshRec := doJSON(t, r, http.MethodPost, "/api/v1/auth/refresh", refreshRequest{RefreshToken: tokens.RefreshToken})
	if refreshRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", refreshRec.Code, refreshRec.Body.String())
	}

	var refreshed tokenPairResponse
	decodeBody(t, refreshRec, &refreshed)
	if refreshed.RefreshToken == tokens.RefreshToken {
		t.Error("expected a new refresh token to be issued")
	}

	// Old refresh token must now be rejected (rotation).
	reuseRec := doJSON(t, r, http.MethodPost, "/api/v1/auth/refresh", refreshRequest{RefreshToken: tokens.RefreshToken})
	if reuseRec.Code != http.StatusUnauthorized {
		t.Errorf("expected reused old refresh token to be rejected, got %d", reuseRec.Code)
	}
}

func TestLogoutRevokesRefreshToken(t *testing.T) {
	r := newTestRouter(t)

	rec := doJSON(t, r, http.MethodPost, "/api/v1/auth/register", registerRequest{
		Email: "logout@example.com", Username: "logout", Password: "password123",
	})
	var tokens tokenPairResponse
	decodeBody(t, rec, &tokens)

	logoutRec := doJSON(t, r, http.MethodPost, "/api/v1/auth/logout", refreshRequest{RefreshToken: tokens.RefreshToken})
	if logoutRec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", logoutRec.Code)
	}

	refreshRec := doJSON(t, r, http.MethodPost, "/api/v1/auth/refresh", refreshRequest{RefreshToken: tokens.RefreshToken})
	if refreshRec.Code != http.StatusUnauthorized {
		t.Errorf("expected refresh with a logged-out token to be rejected, got %d", refreshRec.Code)
	}
}
