package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// postJSONNoAuth sends an unauthenticated JSON POST and returns the recorder.
func postJSONNoAuth(t *testing.T, r http.Handler, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, jsonRequest(t, http.MethodPost, path, body))
	return rec
}

func TestChangePasswordThenLoginWithNew(t *testing.T) {
	router, _ := newFullTestRouter(t)
	access := registerAndGetAccessToken(t, router, "pw@example.com", "pwuser")

	// Wrong current password is rejected.
	bad := authedJSON(t, router, http.MethodPost, "/api/v1/me/password", access, changePasswordRequest{
		CurrentPassword: "nope", NewPassword: "newpassword123",
	})
	if bad.Code != http.StatusForbidden {
		t.Fatalf("wrong current password: expected 403, got %d: %s", bad.Code, bad.Body.String())
	}

	// Correct current password changes it.
	ok := authedJSON(t, router, http.MethodPost, "/api/v1/me/password", access, changePasswordRequest{
		CurrentPassword: "password123", NewPassword: "newpassword123",
	})
	if ok.Code != http.StatusNoContent {
		t.Fatalf("change password: expected 204, got %d: %s", ok.Code, ok.Body.String())
	}

	// The new password now logs in; the old one doesn't.
	newLogin := postJSONNoAuth(t, router, "/api/v1/auth/login", loginRequest{Email: "pw@example.com", Password: "newpassword123"})
	if newLogin.Code != http.StatusOK {
		t.Fatalf("login with new password: expected 200, got %d: %s", newLogin.Code, newLogin.Body.String())
	}
	oldLogin := postJSONNoAuth(t, router, "/api/v1/auth/login", loginRequest{Email: "pw@example.com", Password: "password123"})
	if oldLogin.Code == http.StatusOK {
		t.Error("old password should no longer work")
	}
}

func TestAPIKeyAuthenticatesRequests(t *testing.T) {
	router, _ := newFullTestRouter(t)
	access := registerAndGetAccessToken(t, router, "key@example.com", "keyuser")

	// Create a key.
	rec := authedJSON(t, router, http.MethodPost, "/api/v1/me/api-keys", access, createAPIKeyRequest{Name: "ci"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create key: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var created struct {
		Key string `json:"key"`
	}
	decodeBody(t, rec, &created)
	if created.Key == "" {
		t.Fatal("expected a raw key in the response")
	}

	// The key authenticates an API call (as a bearer token) just like a JWT.
	meRec := authedRequest(t, router, http.MethodGet, "/api/v1/me", created.Key)
	if meRec.Code != http.StatusOK {
		t.Fatalf("me via API key: expected 200, got %d: %s", meRec.Code, meRec.Body.String())
	}
	var me userResponse
	decodeBody(t, meRec, &me)
	if me.Username != "keyuser" {
		t.Errorf("API key resolved to wrong user: %+v", me)
	}

	// It shows up in the list, and a bogus key is rejected.
	list := authedRequest(t, router, http.MethodGet, "/api/v1/me/api-keys", access)
	var keys []apiKeyResponse
	decodeBody(t, list, &keys)
	if len(keys) != 1 {
		t.Fatalf("expected 1 API key, got %d", len(keys))
	}
	if bogus := authedRequest(t, router, http.MethodGet, "/api/v1/me", "sky_deadbeef"); bogus.Code != http.StatusUnauthorized {
		t.Errorf("bogus API key should be 401, got %d", bogus.Code)
	}
}

func TestLeaderboardRanksByCoins(t *testing.T) {
	router, _ := newFullTestRouter(t)
	admin := registerAndGetAccessToken(t, router, "admin@example.com", "admin")
	poorID := meID(t, router, registerAndGetAccessToken(t, router, "poor@example.com", "poor"))
	richID := meID(t, router, registerAndGetAccessToken(t, router, "rich@example.com", "rich"))

	authedJSON(t, router, http.MethodPost, "/api/v1/admin/users/"+poorID+"/coins/adjust", admin, adminAdjustCoinsRequest{Amount: 10})
	authedJSON(t, router, http.MethodPost, "/api/v1/admin/users/"+richID+"/coins/adjust", admin, adminAdjustCoinsRequest{Amount: 9000})

	rec := authedRequest(t, router, http.MethodGet, "/api/v1/leaderboard", admin)
	if rec.Code != http.StatusOK {
		t.Fatalf("leaderboard: expected 200, got %d", rec.Code)
	}
	var board []leaderboardEntry
	decodeBody(t, rec, &board)
	if len(board) < 2 || board[0].Username != "rich" || board[0].Rank != 1 {
		t.Fatalf("expected 'rich' ranked #1, got %+v", board)
	}
}

func TestServerSchedulesCRUD(t *testing.T) {
	router, _, _, ownerAccess, server := setupServerWithFakeAgent(t)

	create := authedJSON(t, router, http.MethodPost, "/api/v1/servers/"+server.ID+"/schedules", ownerAccess, createScheduleRequest{
		Name: "nightly restart", Action: "restart", IntervalMinutes: 1440,
	})
	if create.Code != http.StatusCreated {
		t.Fatalf("create schedule: expected 201, got %d: %s", create.Code, create.Body.String())
	}
	var sched scheduleResponse
	decodeBody(t, create, &sched)
	if sched.Action != "restart" || !sched.Enabled {
		t.Fatalf("unexpected schedule: %+v", sched)
	}

	// A command schedule requires a payload.
	bad := authedJSON(t, router, http.MethodPost, "/api/v1/servers/"+server.ID+"/schedules", ownerAccess, createScheduleRequest{
		Action: "command", IntervalMinutes: 60,
	})
	if bad.Code != http.StatusBadRequest {
		t.Errorf("command schedule without payload should be 400, got %d", bad.Code)
	}

	list := authedRequest(t, router, http.MethodGet, "/api/v1/servers/"+server.ID+"/schedules", ownerAccess)
	var schedules []scheduleResponse
	decodeBody(t, list, &schedules)
	if len(schedules) != 1 {
		t.Fatalf("expected 1 schedule, got %d", len(schedules))
	}

	del := authedRequest(t, router, http.MethodDelete, "/api/v1/servers/"+server.ID+"/schedules/"+sched.ID, ownerAccess)
	if del.Code != http.StatusNoContent {
		t.Fatalf("delete schedule: expected 204, got %d", del.Code)
	}
}
