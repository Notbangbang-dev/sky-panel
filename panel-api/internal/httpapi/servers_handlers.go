package httpapi

import (
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/agenthub"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/auth"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/repo"
)

type createServerRequest struct {
	NodeID      string            `json:"node_id"`
	EggID       string            `json:"egg_id"`
	Name        string            `json:"name"`
	MemoryBytes int64             `json:"memory_bytes"`
	Variables   map[string]string `json:"variables,omitempty"`
}

type serverResponse struct {
	ID          string            `json:"id"`
	OwnerID     string            `json:"owner_id"`
	NodeID      string            `json:"node_id"`
	EggID       string            `json:"egg_id"`
	Name        string            `json:"name"`
	Status      string            `json:"status"`
	MemoryBytes int64             `json:"memory_bytes"`
	PrimaryPort int               `json:"primary_port"`
	Variables   map[string]string `json:"variables"`
}

func toServerResponse(s *models.Server) serverResponse {
	return serverResponse{
		ID: s.ID, OwnerID: s.OwnerID, NodeID: s.NodeID, EggID: s.EggID, Name: s.Name,
		Status: string(s.Status), MemoryBytes: s.MemoryBytes, PrimaryPort: s.PrimaryPort, Variables: s.Variables,
	}
}

func (d Deps) CreateServer(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	var req createServerRequest
	if err := decodeJSON(r, &req); err != nil || req.NodeID == "" || req.EggID == "" || req.Name == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "node_id, egg_id and name are required")
		return
	}

	server, err := d.ServerSvc.CreateServer(claims.UserID, req.NodeID, req.EggID, req.Name, req.MemoryBytes, req.Variables)
	if err != nil {
		if server != nil {
			// Provisioned but the node failed to actually create/start it;
			// surface as 502 rather than losing the row from view.
			writeJSON(w, http.StatusBadGateway, map[string]any{
				"error": "node_dispatch_failed", "message": err.Error(), "server": toServerResponse(server),
			})
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, toServerResponse(server))
}

func (d Deps) ListServers(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	var servers []*models.Server
	var err error
	if claims.Role == string(models.RoleAdmin) {
		servers, err = d.Servers.ListAll()
	} else {
		servers, err = d.Servers.ListByOwner(claims.UserID)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list servers")
		return
	}

	out := make([]serverResponse, 0, len(servers))
	for _, s := range servers {
		out = append(out, toServerResponse(s))
	}
	writeJSON(w, http.StatusOK, out)
}

// loadOwnedServer loads a server and enforces that the caller either owns it
// or is an admin, returning nil (after writing the appropriate error
// response) if access should be denied.
func (d Deps) loadOwnedServer(w http.ResponseWriter, r *http.Request) *models.Server {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return nil
	}

	server, err := d.Servers.GetByID(pathParam(r, "serverID"))
	if errors.Is(err, repo.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "server not found")
		return nil
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load server")
		return nil
	}

	if server.OwnerID != claims.UserID && claims.Role != string(models.RoleAdmin) {
		writeError(w, http.StatusForbidden, "forbidden", "you do not own this server")
		return nil
	}
	return server
}

func (d Deps) GetServer(w http.ResponseWriter, r *http.Request) {
	server := d.loadOwnedServer(w, r)
	if server == nil {
		return
	}
	writeJSON(w, http.StatusOK, toServerResponse(server))
}

func (d Deps) DeleteServer(w http.ResponseWriter, r *http.Request) {
	server := d.loadOwnedServer(w, r)
	if server == nil {
		return
	}

	if err := d.ServerSvc.DeleteServer(server.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type powerActionRequest struct {
	Action string `json:"action"`
}

func (d Deps) PowerAction(w http.ResponseWriter, r *http.Request) {
	server := d.loadOwnedServer(w, r)
	if server == nil {
		return
	}

	var req powerActionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	switch req.Action {
	case agenthub.ActionStart, agenthub.ActionStop, agenthub.ActionKill:
	default:
		writeError(w, http.StatusBadRequest, "bad_request", "action must be one of: start, stop, kill")
		return
	}

	if err := d.ServerSvc.PowerAction(server.ID, req.Action); err != nil {
		writeError(w, http.StatusBadGateway, "node_dispatch_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type consoleInputRequest struct {
	Input string `json:"input"`
}

func (d Deps) ConsoleInput(w http.ResponseWriter, r *http.Request) {
	server := d.loadOwnedServer(w, r)
	if server == nil {
		return
	}

	var req consoleInputRequest
	if err := decodeJSON(r, &req); err != nil || req.Input == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "input is required")
		return
	}

	ack, err := d.AgentHub.Registry.SendCommand(server.NodeID, agenthub.CommandPayload{
		CommandID: uuid.NewString(),
		Action:    agenthub.ActionConsole,
		ServerID:  server.ID,
		Input:     req.Input,
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, "node_dispatch_failed", err.Error())
		return
	}
	if !ack.OK {
		writeError(w, http.StatusBadGateway, "node_dispatch_failed", ack.Error)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
