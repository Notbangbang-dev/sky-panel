package httpapi

import (
	"net/http"
	"testing"
)

func getServerAllocations(t *testing.T, router http.Handler, admin, serverID string) serverAllocationsResponse {
	t.Helper()
	rec := authedRequest(t, router, http.MethodGet, "/api/v1/admin/servers/"+serverID+"/allocations", admin)
	if rec.Code != http.StatusOK {
		t.Fatalf("list server allocations: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var out serverAllocationsResponse
	decodeBody(t, rec, &out)
	return out
}

// TestAdminServerAllocationsLifecycle drives the admin-only multi-port flow end
// to end over the fake node agent: list → attach a free port (which
// re-provisions the container) → confirm the owner sees it → guard the primary
// → detach → and the not-found / in-use error paths.
func TestAdminServerAllocationsLifecycle(t *testing.T) {
	router, _, adminAccess, ownerAccess, server := setupServerWithFakeAgent(t)

	// Fresh server: one primary port, the rest of the node's pool free.
	got := getServerAllocations(t, router, adminAccess, server.ID)
	if len(got.Ports) != 1 || !got.Ports[0].Primary {
		t.Fatalf("expected exactly one primary port, got %+v", got.Ports)
	}
	if len(got.Free) != DefaultAllocationCount-1 {
		t.Fatalf("expected %d free ports, got %d", DefaultAllocationCount-1, len(got.Free))
	}
	primaryID := got.Ports[0].ID
	freeID := got.Free[0].ID

	// Attach a free port; the handler re-provisions synchronously against the
	// fake agent, so a 200 means the container was recreated with the new port.
	rec := authedJSON(t, router, http.MethodPost, "/api/v1/admin/servers/"+server.ID+"/allocations", adminAccess, addServerAllocationRequest{AllocationID: freeID})
	if rec.Code != http.StatusOK {
		t.Fatalf("attach port: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	got = getServerAllocations(t, router, adminAccess, server.ID)
	if len(got.Ports) != 2 {
		t.Fatalf("expected 2 ports after attach, got %+v", got.Ports)
	}
	if len(got.Free) != DefaultAllocationCount-2 {
		t.Fatalf("expected %d free ports after attach, got %d", DefaultAllocationCount-2, len(got.Free))
	}

	// The owner can now see the extra port on the single-server GET (read-only).
	srvRec := authedRequest(t, router, http.MethodGet, "/api/v1/servers/"+server.ID, ownerAccess)
	if srvRec.Code != http.StatusOK {
		t.Fatalf("owner GET server: expected 200, got %d: %s", srvRec.Code, srvRec.Body.String())
	}
	var srv serverResponse
	decodeBody(t, srvRec, &srv)
	if len(srv.AdditionalPorts) != 1 {
		t.Errorf("expected 1 additional port on the server response, got %+v", srv.AdditionalPorts)
	}

	// The primary port can't be removed.
	if del := authedRequest(t, router, http.MethodDelete, "/api/v1/admin/servers/"+server.ID+"/allocations/"+primaryID, adminAccess); del.Code != http.StatusConflict {
		t.Fatalf("remove primary: expected 409, got %d: %s", del.Code, del.Body.String())
	}

	// Removing the additional port succeeds and frees it again.
	if del := authedRequest(t, router, http.MethodDelete, "/api/v1/admin/servers/"+server.ID+"/allocations/"+freeID, adminAccess); del.Code != http.StatusNoContent {
		t.Fatalf("remove extra: expected 204, got %d: %s", del.Code, del.Body.String())
	}
	got = getServerAllocations(t, router, adminAccess, server.ID)
	if len(got.Ports) != 1 {
		t.Fatalf("expected 1 port after detach, got %+v", got.Ports)
	}

	// Error paths: attaching an already-held port (the primary) is a conflict;
	// a bogus allocation id is a 404.
	if rec := authedJSON(t, router, http.MethodPost, "/api/v1/admin/servers/"+server.ID+"/allocations", adminAccess, addServerAllocationRequest{AllocationID: primaryID}); rec.Code != http.StatusConflict {
		t.Fatalf("attach in-use: expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
	if rec := authedJSON(t, router, http.MethodPost, "/api/v1/admin/servers/"+server.ID+"/allocations", adminAccess, addServerAllocationRequest{AllocationID: "does-not-exist"}); rec.Code != http.StatusNotFound {
		t.Fatalf("attach unknown: expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}
