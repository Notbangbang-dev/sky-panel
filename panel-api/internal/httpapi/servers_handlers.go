package httpapi

import (
	"errors"
	"log"
	"net/http"

	"github.com/google/uuid"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/agenthub"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/auth"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/repo"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/serversvc"
)

type createServerRequest struct {
	NodeID      string            `json:"node_id"`
	EggID       string            `json:"egg_id"`
	Name        string            `json:"name"`
	MemoryBytes int64             `json:"memory_bytes"`
	CPULimit    int               `json:"cpu_limit"`
	DiskBytes   int64             `json:"disk_bytes"`
	Variables   map[string]string `json:"variables,omitempty"`
}

type serverResponse struct {
	ID                  string            `json:"id"`
	OwnerID             string            `json:"owner_id"`
	NodeID              string            `json:"node_id"`
	EggID               string            `json:"egg_id"`
	Name                string            `json:"name"`
	Status              string            `json:"status"`
	MemoryBytes         int64             `json:"memory_bytes"`
	CPULimit            int               `json:"cpu_limit"`
	DiskBytes           int64             `json:"disk_bytes"`
	PrimaryPort         int               `json:"primary_port"`
	Variables           map[string]string `json:"variables"`
	BackupIntervalHours int               `json:"backup_interval_hours"`
	LastBackupAt        string            `json:"last_backup_at,omitempty"`
	Suspended           bool              `json:"suspended"`
	StatusMessage       string            `json:"status_message,omitempty"`
	Description         string            `json:"description"`
}

func toServerResponse(s *models.Server) serverResponse {
	resp := serverResponse{
		ID: s.ID, OwnerID: s.OwnerID, NodeID: s.NodeID, EggID: s.EggID, Name: s.Name,
		Status: string(s.Status), MemoryBytes: s.MemoryBytes, CPULimit: s.CPULimit, DiskBytes: s.DiskBytes,
		PrimaryPort: s.PrimaryPort, Variables: s.Variables, BackupIntervalHours: s.BackupIntervalHours,
		Suspended: s.Suspended, StatusMessage: s.StatusMessage, Description: s.Description,
	}
	if s.LastBackupAt != nil {
		resp.LastBackupAt = s.LastBackupAt.Format(rfc3339)
	}
	return resp
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

	// Serialize this user's create/clone requests so the quota check and the
	// row insert are atomic per user (see keyedMutex): concurrent creates must
	// not each pass the check against the same pre-insert snapshot.
	unlock := serverCreateLocks.lock(claims.UserID)
	defer unlock()

	// Enforce the user's resource quota (admins are unmetered).
	if claims.Role != string(models.RoleAdmin) {
		if err := d.QuotaSvc.CheckCreate(claims.UserID, req.MemoryBytes, req.CPULimit, req.DiskBytes); err != nil {
			d.writeQuotaError(w, err)
			return
		}
	}

	server, err := d.ServerSvc.CreateServer(claims.UserID, req.NodeID, req.EggID, req.Name, req.MemoryBytes, req.CPULimit, req.DiskBytes, req.Variables)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	d.audit(r, "server.create", server.ID, server.Name)

	// Provision (create + start the container, which may pull a large image on
	// first use) in the background so the request returns immediately with an
	// "installing" server rather than timing out. The recover keeps a
	// provisioning panic from taking down the API.
	go func(serverID, name string) {
		defer func() {
			if p := recover(); p != nil {
				log.Printf("server %s provisioning panicked: %v", serverID, p)
			}
		}()
		if err := d.ServerSvc.Provision(serverID, serversvc.ProvisionCreateTimeout); err != nil {
			log.Printf("server %s (%s) provisioning failed: %v", serverID, name, err)
		}
	}(server.ID, server.Name)

	writeJSON(w, http.StatusCreated, toServerResponse(server))
}

// ServerStats returns the most recent live stats the panel has received for a
// server (cached from the node's heartbeats), so the UI shows numbers on page
// load and after a brief WebSocket gap instead of a dash. 204 when there's no
// fresh sample (server stopped, or the node hasn't reported yet).
func (d Deps) ServerStats(w http.ResponseWriter, r *http.Request) {
	server := d.loadServerWithPermission(w, r, "")
	if server == nil {
		return
	}
	msg, ok := d.AgentHub.LatestStats(server.ID)
	if !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(msg)
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

// loadServerWithPermission loads a server and enforces that the caller is
// its owner, an admin, or a subuser holding requiredPerm on it. Pass "" for
// requiredPerm to allow any subuser regardless of their specific grants
// (used for read-only access like GetServer).
func (d Deps) loadServerWithPermission(w http.ResponseWriter, r *http.Request, requiredPerm string) *models.Server {
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

	if server.OwnerID == claims.UserID || claims.Role == string(models.RoleAdmin) {
		return server
	}

	sub, err := d.Subusers.Get(server.ID, claims.UserID)
	if err == nil && (requiredPerm == "" || sub.HasPermission(requiredPerm)) {
		return server
	}

	writeError(w, http.StatusForbidden, "forbidden", "you do not have access to this server")
	return nil
}

// isAdmin reports whether the authenticated caller is an admin.
func (d Deps) isAdmin(r *http.Request) bool {
	claims, ok := auth.FromContext(r.Context())
	return ok && claims.Role == string(models.RoleAdmin)
}

func (d Deps) GetServer(w http.ResponseWriter, r *http.Request) {
	server := d.loadServerWithPermission(w, r, "")
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
	d.audit(r, "server.delete", server.ID, server.Name)
	w.WriteHeader(http.StatusNoContent)
}

type powerActionRequest struct {
	Action string `json:"action"`
}

func (d Deps) PowerAction(w http.ResponseWriter, r *http.Request) {
	server := d.loadServerWithPermission(w, r, models.PermPower)
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

	// A suspended server can't be started by its owner (stop/kill stay allowed
	// so it can be shut down). Admins are exempt.
	if server.Suspended && req.Action == agenthub.ActionStart && !d.isAdmin(r) {
		writeError(w, http.StatusForbidden, "server_suspended", "this server is suspended by an administrator")
		return
	}

	if err := d.ServerSvc.PowerAction(server.ID, req.Action); err != nil {
		writeError(w, http.StatusBadGateway, "node_dispatch_failed", err.Error())
		return
	}
	d.audit(r, "server.power."+req.Action, server.ID, "")
	w.WriteHeader(http.StatusNoContent)
}

type consoleInputRequest struct {
	Input string `json:"input"`
}

func (d Deps) ConsoleInput(w http.ResponseWriter, r *http.Request) {
	server := d.loadServerWithPermission(w, r, models.PermConsole)
	if server == nil {
		return
	}

	if server.Suspended && !d.isAdmin(r) {
		writeError(w, http.StatusForbidden, "server_suspended", "this server is suspended by an administrator")
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

type updateServerRequest struct {
	Name                string            `json:"name"`
	MemoryBytes         int64             `json:"memory_bytes"`
	CPULimit            int               `json:"cpu_limit"`
	DiskBytes           int64             `json:"disk_bytes"`
	Variables           map[string]string `json:"variables"`
	BackupIntervalHours int               `json:"backup_interval_hours"`
}

// UpdateServer applies edited settings and re-provisions the container.
// Requires the "settings" permission (owner/admin/settings-subuser).
func (d Deps) UpdateServer(w http.ResponseWriter, r *http.Request) {
	server := d.loadServerWithPermission(w, r, models.PermSettings)
	if server == nil {
		return
	}
	// Saving settings re-provisions (and starts) the container, so a suspended
	// owner must not be able to use it as a back door to restart. Admins may.
	if server.Suspended && !d.isAdmin(r) {
		writeError(w, http.StatusForbidden, "server_suspended", "this server is suspended by an administrator")
		return
	}

	var req updateServerRequest
	if err := decodeJSON(r, &req); err != nil || req.Name == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "name is required")
		return
	}
	// An omitted numeric field decodes to 0; treat that as "keep current"
	// rather than wiping the server's allocation (which would also corrupt
	// quota accounting on the next check).
	if req.MemoryBytes <= 0 {
		req.MemoryBytes = server.MemoryBytes
	}
	if req.CPULimit <= 0 {
		req.CPULimit = server.CPULimit
	}
	if req.DiskBytes <= 0 {
		req.DiskBytes = server.DiskBytes
	}

	// Enforce quota against the new limits, excluding this server's current
	// allocation from the total (admins are unmetered).
	claims, _ := auth.FromContext(r.Context())
	if claims != nil && claims.Role != string(models.RoleAdmin) {
		if err := d.QuotaSvc.CheckUpdate(server.OwnerID, server.ID, req.MemoryBytes, req.CPULimit, req.DiskBytes); err != nil {
			d.writeQuotaError(w, err)
			return
		}
	}

	updated, err := d.ServerSvc.UpdateServer(server.ID, req.Name, req.MemoryBytes, req.CPULimit, req.DiskBytes, req.Variables, req.BackupIntervalHours)
	if err != nil {
		if updated != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{
				"error": "node_dispatch_failed", "message": err.Error(), "server": toServerResponse(updated),
			})
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	d.audit(r, "server.settings", server.ID, req.Name)
	writeJSON(w, http.StatusOK, toServerResponse(updated))
}

type reinstallServerRequest struct {
	// EggID optionally reinstalls onto a different egg (software). Empty keeps
	// the current one.
	EggID string `json:"egg_id"`
}

// ReinstallServer recreates the container from its egg (files preserved),
// optionally switching to a different egg.
func (d Deps) ReinstallServer(w http.ResponseWriter, r *http.Request) {
	server := d.loadServerWithPermission(w, r, models.PermSettings)
	if server == nil {
		return
	}
	if server.Suspended && !d.isAdmin(r) {
		writeError(w, http.StatusForbidden, "server_suspended", "this server is suspended by an administrator")
		return
	}

	// Body is optional (a bare POST reinstalls onto the same egg).
	var req reinstallServerRequest
	_ = decodeJSON(r, &req)
	if req.EggID != "" {
		if _, err := d.Eggs.GetByID(req.EggID); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "unknown egg")
			return
		}
	}
	d.audit(r, "server.reinstall", server.ID, req.EggID)

	// Mark it installing synchronously so a client polling right after this
	// 204 sees "installing" immediately, rather than racing the goroutine.
	_ = d.Servers.SetStatus(server.ID, models.StatusInstalling)

	// Reinstall re-pulls/recreates the container, which can take minutes; run
	// it in the background (like create) so the request returns immediately and
	// the server shows "installing" until it's done.
	go func(serverID, eggID string) {
		defer func() {
			if p := recover(); p != nil {
				log.Printf("server %s reinstall panicked: %v", serverID, p)
			}
		}()
		if err := d.ServerSvc.ReinstallServer(serverID, eggID); err != nil {
			log.Printf("server %s reinstall failed: %v", serverID, err)
		}
	}(server.ID, req.EggID)

	w.WriteHeader(http.StatusNoContent)
}

// ServerActivity returns the recent audit entries scoped to this server.
func (d Deps) ServerActivity(w http.ResponseWriter, r *http.Request) {
	server := d.loadServerWithPermission(w, r, "")
	if server == nil {
		return
	}
	entries, err := d.Audit.ListByTarget(server.ID, 100)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load activity")
		return
	}
	out := make([]auditEntryResponse, 0, len(entries))
	for _, e := range entries {
		out = append(out, toAuditEntryResponse(e))
	}
	writeJSON(w, http.StatusOK, out)
}
