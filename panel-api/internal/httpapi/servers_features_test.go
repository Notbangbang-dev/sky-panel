package httpapi

import (
	"net/http"
	"testing"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/agenthub"
)

func TestUpdateServerSettings(t *testing.T) {
	router, _, _, ownerAccess, server := setupServerWithFakeAgent(t)

	rec := authedJSON(t, router, http.MethodPatch, "/api/v1/servers/"+server.ID, ownerAccess, updateServerRequest{
		Name:                "Renamed Server",
		MemoryBytes:         2048 * 1024 * 1024,
		CPULimit:            150,
		Variables:           map[string]string{"EULA": "TRUE"},
		BackupIntervalHours: 24,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("update settings: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var updated serverResponse
	decodeBody(t, rec, &updated)
	if updated.Name != "Renamed Server" || updated.CPULimit != 150 || updated.BackupIntervalHours != 24 {
		t.Fatalf("settings not applied: %+v", updated)
	}
	if updated.MemoryBytes != 2048*1024*1024 {
		t.Errorf("memory not updated, got %d", updated.MemoryBytes)
	}
}

func TestReinstallServer(t *testing.T) {
	router, _, _, ownerAccess, server := setupServerWithFakeAgent(t)

	rec := authedRequest(t, router, http.MethodPost, "/api/v1/servers/"+server.ID+"/reinstall", ownerAccess)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("reinstall: expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestServerActivityRecordsActions(t *testing.T) {
	router, _, _, ownerAccess, server := setupServerWithFakeAgent(t)

	// Generate an auditable action against the server.
	authedJSON(t, router, http.MethodPost, "/api/v1/servers/"+server.ID+"/power", ownerAccess, powerActionRequest{Action: "stop"})

	rec := authedRequest(t, router, http.MethodGet, "/api/v1/servers/"+server.ID+"/activity", ownerAccess)
	if rec.Code != http.StatusOK {
		t.Fatalf("activity: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var entries []auditEntryResponse
	decodeBody(t, rec, &entries)
	if len(entries) == 0 {
		t.Fatal("expected at least one activity entry for the server")
	}
	foundPower := false
	for _, e := range entries {
		if e.Target != server.ID {
			t.Errorf("activity entry targets %q, expected %q", e.Target, server.ID)
		}
		if e.Action == "server.power.stop" {
			foundPower = true
		}
	}
	if !foundPower {
		t.Errorf("expected a server.power.stop entry, got %+v", entries)
	}
}

func TestBackupNowAndList(t *testing.T) {
	router, _, _, ownerAccess, server := setupServerWithFakeAgent(t)

	createRec := authedRequest(t, router, http.MethodPost, "/api/v1/servers/"+server.ID+"/backups", ownerAccess)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("backup now: expected 201, got %d: %s", createRec.Code, createRec.Body.String())
	}
	var result agenthub.BackupResult
	decodeBody(t, createRec, &result)
	if result.Filename == "" {
		t.Errorf("expected a backup filename in the response")
	}

	listRec := authedRequest(t, router, http.MethodGet, "/api/v1/servers/"+server.ID+"/backups", ownerAccess)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list backups: expected 200, got %d: %s", listRec.Code, listRec.Body.String())
	}
	var listBody struct {
		Backups []agenthub.BackupEntry `json:"backups"`
	}
	decodeBody(t, listRec, &listBody)
	if len(listBody.Backups) != 1 || listBody.Backups[0].Filename == "" {
		t.Fatalf("unexpected backup list: %+v", listBody.Backups)
	}
}
