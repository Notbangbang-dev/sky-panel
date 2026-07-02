package httpapi

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/repo"
)

const (
	// DefaultAllocationStart / DefaultAllocationCount define the block of
	// ports auto-created for every new node, so a fresh install can host
	// servers immediately without an operator hand-seeding the database.
	// 25565 is the Minecraft default; 50 ports covers a typical small node.
	DefaultAllocationStart = 25565
	DefaultAllocationCount = 50

	// maxAllocationRange caps a single create request so an accidental huge
	// range can't lock the DB inserting millions of rows.
	maxAllocationRange = 5000
)

type allocationResponse struct {
	ID         string `json:"id"`
	Port       int    `json:"port"`
	ServerID   string `json:"server_id,omitempty"`
	ServerName string `json:"server_name,omitempty"`
}

// AdminListAllocations returns every port allocation on a node, marking which
// are free vs. held by a server (with that server's name for display).
func (d Deps) AdminListAllocations(w http.ResponseWriter, r *http.Request) {
	nodeID := pathParam(r, "nodeID")
	if _, err := d.Nodes.GetByID(nodeID); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "node not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load node")
		return
	}

	allocs, err := d.Allocations.ListByNode(nodeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list allocations")
		return
	}

	// One pass over all servers to resolve owner names for the in-use ports.
	names := map[string]string{}
	if servers, err := d.Servers.ListAll(); err == nil {
		for _, s := range servers {
			names[s.ID] = s.Name
		}
	}

	out := make([]allocationResponse, 0, len(allocs))
	for _, a := range allocs {
		resp := allocationResponse{ID: a.ID, Port: a.Port}
		if a.ServerID != nil {
			resp.ServerID = *a.ServerID
			resp.ServerName = names[*a.ServerID]
		}
		out = append(out, resp)
	}
	writeJSON(w, http.StatusOK, out)
}

type createAllocationsRequest struct {
	Port      int `json:"port"`
	PortStart int `json:"port_start"`
	PortEnd   int `json:"port_end"`
}

// AdminCreateAllocations adds a single port ({port}) or an inclusive range
// ({port_start, port_end}) of free allocations to a node, skipping any that
// already exist. Returns how many were newly created.
func (d Deps) AdminCreateAllocations(w http.ResponseWriter, r *http.Request) {
	nodeID := pathParam(r, "nodeID")
	if _, err := d.Nodes.GetByID(nodeID); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "node not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load node")
		return
	}

	var req createAllocationsRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	start, end := req.PortStart, req.PortEnd
	if req.Port != 0 {
		start, end = req.Port, req.Port
	}
	if start <= 0 || end <= 0 {
		writeError(w, http.StatusBadRequest, "bad_request", "provide a port, or port_start and port_end")
		return
	}
	if start > end {
		start, end = end, start
	}
	if start < 1 || end > 65535 {
		writeError(w, http.StatusBadRequest, "bad_request", "ports must be between 1 and 65535")
		return
	}
	if end-start+1 > maxAllocationRange {
		writeError(w, http.StatusBadRequest, "bad_request", fmt.Sprintf("range too large (max %d ports at once)", maxAllocationRange))
		return
	}

	created, err := d.Allocations.CreateRange(nodeID, start, end)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to create allocations")
		return
	}
	d.audit(r, "allocation.create", nodeID, fmt.Sprintf("%d-%d (%d new)", start, end, created))
	writeJSON(w, http.StatusCreated, map[string]int{"created": created})
}

// AdminDeleteAllocation removes a free allocation; a port held by a server
// can't be deleted until that server is removed.
func (d Deps) AdminDeleteAllocation(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "allocationID")
	err := d.Allocations.Delete(id)
	if errors.Is(err, repo.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "allocation not found")
		return
	}
	if errors.Is(err, repo.ErrAllocationInUse) {
		writeError(w, http.StatusConflict, "allocation_in_use", "this port is in use by a server; delete the server first")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete allocation")
		return
	}
	d.audit(r, "allocation.delete", id, "")
	w.WriteHeader(http.StatusNoContent)
}
