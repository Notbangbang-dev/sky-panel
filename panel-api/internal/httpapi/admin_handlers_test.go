package httpapi

import (
	"net/http"
	"testing"
)

func TestAdminUserManagement(t *testing.T) {
	router, _ := newFullTestRouter(t)

	adminAccess := registerAndGetAccessToken(t, router, "admin@example.com", "admin")
	registerAndGetAccessToken(t, router, "user@example.com", "regular")

	listRec := authedRequest(t, router, http.MethodGet, "/api/v1/admin/users", adminAccess)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", listRec.Code, listRec.Body.String())
	}
	var users []userResponse
	decodeBody(t, listRec, &users)
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}

	var regularUserID string
	for _, u := range users {
		if u.Username == "regular" {
			regularUserID = u.ID
		}
	}
	if regularUserID == "" {
		t.Fatal("could not find regular user in list")
	}

	roleRec := authedJSON(t, router, http.MethodPost, "/api/v1/admin/users/"+regularUserID+"/role", adminAccess, setUserRoleRequest{Role: "admin"})
	if roleRec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 from role change, got %d: %s", roleRec.Code, roleRec.Body.String())
	}

	// An admin cannot delete their own account.
	var adminUserID string
	for _, u := range users {
		if u.Username == "admin" {
			adminUserID = u.ID
		}
	}
	selfDeleteRec := authedRequest(t, router, http.MethodDelete, "/api/v1/admin/users/"+adminUserID, adminAccess)
	if selfDeleteRec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for self-delete, got %d", selfDeleteRec.Code)
	}

	deleteRec := authedRequest(t, router, http.MethodDelete, "/api/v1/admin/users/"+regularUserID, adminAccess)
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 from delete, got %d: %s", deleteRec.Code, deleteRec.Body.String())
	}
}

func TestAdminUserManagementRequiresAdminRole(t *testing.T) {
	router, _ := newFullTestRouter(t)
	registerAndGetAccessToken(t, router, "admin@example.com", "admin") // bootstrap admin
	userAccess := registerAndGetAccessToken(t, router, "user@example.com", "regular")

	rec := authedRequest(t, router, http.MethodGet, "/api/v1/admin/users", userAccess)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for a non-admin, got %d", rec.Code)
	}
}

func TestAdminSettingsRoundTrip(t *testing.T) {
	router, _ := newFullTestRouter(t)
	adminAccess := registerAndGetAccessToken(t, router, "admin@example.com", "admin")

	setRec := authedJSON(t, router, http.MethodPut, "/api/v1/admin/settings/site_name", adminAccess, setSettingRequest{Value: "Sky Panel"})
	if setRec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", setRec.Code, setRec.Body.String())
	}

	getRec := authedRequest(t, router, http.MethodGet, "/api/v1/admin/settings", adminAccess)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", getRec.Code)
	}
	var settings map[string]string
	decodeBody(t, getRec, &settings)
	if settings["site_name"] != "Sky Panel" {
		t.Errorf("expected site_name=Sky Panel, got %+v", settings)
	}
}

func TestAdminAuditLogRecordsActions(t *testing.T) {
	router, _ := newFullTestRouter(t)
	adminAccess := registerAndGetAccessToken(t, router, "admin@example.com", "admin")

	authedJSON(t, router, http.MethodPut, "/api/v1/admin/settings/foo", adminAccess, setSettingRequest{Value: "bar"})

	rec := authedRequest(t, router, http.MethodGet, "/api/v1/admin/audit-log", adminAccess)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var entries []auditEntryResponse
	decodeBody(t, rec, &entries)
	if len(entries) != 1 || entries[0].Action != "settings.set" {
		t.Errorf("expected 1 settings.set audit entry, got %+v", entries)
	}
}

func TestAdminBroadcastReachesSubscribers(t *testing.T) {
	router, _ := newFullTestRouter(t)
	adminAccess := registerAndGetAccessToken(t, router, "admin@example.com", "admin")

	rec := authedJSON(t, router, http.MethodPost, "/api/v1/admin/broadcast", adminAccess, broadcastRequest{Message: "server maintenance at midnight"})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}
