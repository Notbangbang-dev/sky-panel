package httpapi

import (
	"errors"
	"net/http"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/repo"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/serversvc"
)

// serverPort is one port a server currently holds; primary marks its main
// allocation (which can't be removed).
type serverPort struct {
	ID      string `json:"id"`
	Port    int    `json:"port"`
	Primary bool   `json:"primary"`
}

// freePort is an unclaimed allocation on the server's node, offered to the
// admin as a candidate additional port.
type freePort struct {
	ID   string `json:"id"`
	Port int    `json:"port"`
}

type serverAllocationsResponse struct {
	Ports []serverPort `json:"ports"`
	Free  []freePort   `json:"free"`
}

// serverAllocations builds the ports-held + ports-free view for one server so
// the admin UI can render the manager and populate its add-dropdown in a single
// request.
func (d Deps) serverAllocations(server *models.Server) (serverAllocationsResponse, error) {
	held, err := d.Allocations.ListByServer(server.ID)
	if err != nil {
		return serverAllocationsResponse{}, err
	}
	ports := make([]serverPort, 0, len(held))
	for _, a := range held {
		ports = append(ports, serverPort{ID: a.ID, Port: a.Port, Primary: a.Port == server.PrimaryPort})
	}

	nodeAllocs, err := d.Allocations.ListByNode(server.NodeID)
	if err != nil {
		return serverAllocationsResponse{}, err
	}
	free := make([]freePort, 0)
	for _, a := range nodeAllocs {
		if a.ServerID == nil {
			free = append(free, freePort{ID: a.ID, Port: a.Port})
		}
	}
	return serverAllocationsResponse{Ports: ports, Free: free}, nil
}

// AdminListServerAllocations returns a server's ports (primary + additional)
// plus the free ports available on its node.
func (d Deps) AdminListServerAllocations(w http.ResponseWriter, r *http.Request) {
	server, ok := d.loadServerOr404(w, r)
	if !ok {
		return
	}
	resp, err := d.serverAllocations(server)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list server allocations")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

type addServerAllocationRequest struct {
	AllocationID string `json:"allocation_id"`
}

// AdminAddServerAllocation attaches an additional port (a free allocation on the
// server's node) and recreates the container so it's actually published +
// firewalled. Returns the refreshed allocation view.
func (d Deps) AdminAddServerAllocation(w http.ResponseWriter, r *http.Request) {
	server, ok := d.loadServerOr404(w, r)
	if !ok {
		return
	}
	var req addServerAllocationRequest
	if err := decodeJSON(r, &req); err != nil || req.AllocationID == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "provide an allocation_id")
		return
	}

	err := d.ServerSvc.AddServerAllocation(server.ID, req.AllocationID)
	switch {
	case errors.Is(err, repo.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "allocation not found")
		return
	case errors.Is(err, repo.ErrAllocationInUse):
		writeError(w, http.StatusConflict, "allocation_in_use", "that port is already in use by a server")
		return
	case errors.Is(err, serversvc.ErrAllocationWrongNode):
		writeError(w, http.StatusBadRequest, "wrong_node", "that port is on a different node than the server")
		return
	case err != nil:
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to apply the port change (the server's node may be offline)")
		return
	}

	d.audit(r, "allocation.attach", server.ID, req.AllocationID)
	resp, err := d.serverAllocations(server)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "port added but failed to reload")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// AdminRemoveServerAllocation detaches an additional port and recreates the
// container without it. The primary port can't be removed.
func (d Deps) AdminRemoveServerAllocation(w http.ResponseWriter, r *http.Request) {
	server, ok := d.loadServerOr404(w, r)
	if !ok {
		return
	}
	allocID := pathParam(r, "allocationID")

	err := d.ServerSvc.RemoveServerAllocation(server.ID, allocID)
	switch {
	case errors.Is(err, repo.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "that port isn't attached to this server")
		return
	case errors.Is(err, serversvc.ErrPrimaryAllocation):
		writeError(w, http.StatusConflict, "primary_allocation", "the primary port can't be removed")
		return
	case err != nil:
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to apply the port change (the server's node may be offline)")
		return
	}

	d.audit(r, "allocation.detach", server.ID, allocID)
	w.WriteHeader(http.StatusNoContent)
}

// loadServerOr404 resolves the {serverID} path param, writing a 404/500 and
// returning ok=false when it can't.
func (d Deps) loadServerOr404(w http.ResponseWriter, r *http.Request) (*models.Server, bool) {
	server, err := d.Servers.GetByID(pathParam(r, "serverID"))
	if errors.Is(err, repo.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "server not found")
		return nil, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load server")
		return nil, false
	}
	return server, true
}
