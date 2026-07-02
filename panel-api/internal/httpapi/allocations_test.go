package httpapi

import (
	"net/http"
	"testing"
)

func listAllocations(t *testing.T, router http.Handler, admin, nodeID string) []allocationResponse {
	t.Helper()
	rec := authedRequest(t, router, http.MethodGet, "/api/v1/admin/nodes/"+nodeID+"/allocations", admin)
	if rec.Code != http.StatusOK {
		t.Fatalf("list allocations: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var out []allocationResponse
	decodeBody(t, rec, &out)
	return out
}

func TestNodeAutoSeedsDefaultAllocations(t *testing.T) {
	router, _, adminAccess, _, server := setupServerWithFakeAgent(t)

	allocs := listAllocations(t, router, adminAccess, server.NodeID)
	if len(allocs) != DefaultAllocationCount {
		t.Fatalf("expected %d auto-seeded allocations, got %d", DefaultAllocationCount, len(allocs))
	}

	// Exactly one should be in use — the server created by the helper — and it
	// should carry that server's name.
	inUse := 0
	for _, a := range allocs {
		if a.ServerID != "" {
			inUse++
			if a.ServerName != "My Server" {
				t.Errorf("in-use allocation missing server name, got %q", a.ServerName)
			}
		}
	}
	if inUse != 1 {
		t.Errorf("expected exactly 1 in-use allocation, got %d", inUse)
	}
}

func TestAdminCreateAndDeleteAllocations(t *testing.T) {
	router, _, adminAccess, _, server := setupServerWithFakeAgent(t)
	nodeID := server.NodeID

	// Add a range; all ten are new.
	rec := authedJSON(t, router, http.MethodPost, "/api/v1/admin/nodes/"+nodeID+"/allocations", adminAccess, createAllocationsRequest{
		PortStart: 26000, PortEnd: 26009,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create range: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var created struct {
		Created int `json:"created"`
	}
	decodeBody(t, rec, &created)
	if created.Created != 10 {
		t.Fatalf("expected 10 new allocations, got %d", created.Created)
	}

	// Re-adding an overlapping single port creates nothing (skipped).
	rec = authedJSON(t, router, http.MethodPost, "/api/v1/admin/nodes/"+nodeID+"/allocations", adminAccess, createAllocationsRequest{Port: 26000})
	decodeBody(t, rec, &created)
	if created.Created != 0 {
		t.Errorf("expected 0 new allocations for an existing port, got %d", created.Created)
	}

	// Find a free allocation and delete it.
	var freeID, inUseID string
	for _, a := range listAllocations(t, router, adminAccess, nodeID) {
		if a.ServerID == "" && freeID == "" {
			freeID = a.ID
		}
		if a.ServerID != "" {
			inUseID = a.ID
		}
	}
	if freeID == "" || inUseID == "" {
		t.Fatal("expected both a free and an in-use allocation")
	}

	if del := authedRequest(t, router, http.MethodDelete, "/api/v1/admin/allocations/"+freeID, adminAccess); del.Code != http.StatusNoContent {
		t.Fatalf("delete free allocation: expected 204, got %d: %s", del.Code, del.Body.String())
	}

	// Deleting an in-use allocation is refused.
	if del := authedRequest(t, router, http.MethodDelete, "/api/v1/admin/allocations/"+inUseID, adminAccess); del.Code != http.StatusConflict {
		t.Fatalf("delete in-use allocation: expected 409, got %d: %s", del.Code, del.Body.String())
	}
}

func TestAdminCreateAllocationsRejectsBadRange(t *testing.T) {
	router, _, adminAccess, _, server := setupServerWithFakeAgent(t)

	rec := authedJSON(t, router, http.MethodPost, "/api/v1/admin/nodes/"+server.NodeID+"/allocations", adminAccess, createAllocationsRequest{
		PortStart: 1, PortEnd: 70000,
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for an out-of-bounds/huge range, got %d: %s", rec.Code, rec.Body.String())
	}
}
