package httpapi

import (
	"net/http"
	"testing"
)

func TestAdminSuspendBlocksOwnerStart(t *testing.T) {
	router, _, adminAccess, ownerAccess, server := setupServerWithFakeAgent(t)

	// Admin suspends the server.
	susp := authedRequest(t, router, http.MethodPost, "/api/v1/admin/servers/"+server.ID+"/suspend", adminAccess)
	if susp.Code != http.StatusNoContent {
		t.Fatalf("suspend: expected 204, got %d: %s", susp.Code, susp.Body.String())
	}

	// The server now reports suspended.
	var got serverResponse
	decodeBody(t, authedRequest(t, router, http.MethodGet, "/api/v1/servers/"+server.ID, ownerAccess), &got)
	if !got.Suspended {
		t.Error("expected server to be marked suspended")
	}

	// Owner can't start it.
	start := authedJSON(t, router, http.MethodPost, "/api/v1/servers/"+server.ID+"/power", ownerAccess, powerActionRequest{Action: "start"})
	if start.Code != http.StatusForbidden {
		t.Fatalf("owner start while suspended: expected 403, got %d: %s", start.Code, start.Body.String())
	}

	// Owner also can't use the console.
	con := authedJSON(t, router, http.MethodPost, "/api/v1/servers/"+server.ID+"/console", ownerAccess, consoleInputRequest{Input: "say hi"})
	if con.Code != http.StatusForbidden {
		t.Fatalf("owner console while suspended: expected 403, got %d: %s", con.Code, con.Body.String())
	}

	// ...nor can they use settings-save or reinstall as a back door to restart.
	upd := authedJSON(t, router, http.MethodPatch, "/api/v1/servers/"+server.ID, ownerAccess, updateServerRequest{Name: "Nice Try"})
	if upd.Code != http.StatusForbidden {
		t.Fatalf("owner update while suspended: expected 403, got %d: %s", upd.Code, upd.Body.String())
	}
	re := authedRequest(t, router, http.MethodPost, "/api/v1/servers/"+server.ID+"/reinstall", ownerAccess)
	if re.Code != http.StatusForbidden {
		t.Fatalf("owner reinstall while suspended: expected 403, got %d: %s", re.Code, re.Body.String())
	}

	// Admin unsuspends; the owner can start again.
	unsusp := authedRequest(t, router, http.MethodPost, "/api/v1/admin/servers/"+server.ID+"/unsuspend", adminAccess)
	if unsusp.Code != http.StatusNoContent {
		t.Fatalf("unsuspend: expected 204, got %d: %s", unsusp.Code, unsusp.Body.String())
	}
	start = authedJSON(t, router, http.MethodPost, "/api/v1/servers/"+server.ID+"/power", ownerAccess, powerActionRequest{Action: "start"})
	if start.Code != http.StatusNoContent {
		t.Fatalf("owner start after unsuspend: expected 204, got %d: %s", start.Code, start.Body.String())
	}
}

func TestAdminGetUserQuota(t *testing.T) {
	router, _ := newFullTestRouter(t)
	adminAccess := registerAndGetAccessToken(t, router, "admin@example.com", "admin")
	userID := meID(t, router, adminAccess)

	// Grant a bonus, then read it back.
	set := authedJSON(t, router, http.MethodPut, "/api/v1/admin/users/"+userID+"/quota", adminAccess, adminQuotaRequest{
		MemoryBytes: 1024 * 1024 * 1024, CPUPercent: 50, DiskBytes: 2 * 1024 * 1024 * 1024,
	})
	if set.Code != http.StatusOK {
		t.Fatalf("set quota: expected 200, got %d: %s", set.Code, set.Body.String())
	}

	rec := authedRequest(t, router, http.MethodGet, "/api/v1/admin/users/"+userID+"/quota", adminAccess)
	if rec.Code != http.StatusOK {
		t.Fatalf("get quota: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var q adminQuotaResponse
	decodeBody(t, rec, &q)
	if q.Bonus.CPUPercent != 50 || q.Bonus.MemoryBytes != 1024*1024*1024 {
		t.Errorf("unexpected bonus: %+v", q.Bonus)
	}
}
