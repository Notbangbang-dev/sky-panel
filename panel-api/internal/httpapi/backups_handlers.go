package httpapi

import (
	"fmt"
	"net/http"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
)

// Backups are gated behind the "files" permission — a backup is a copy of
// the server's files, so anyone allowed to touch files may manage them.

// maxBackupsPerServer caps how many backup archives a server may hold, so a
// user (or a runaway schedule) can't fill a node's disk with unlimited
// backups. When at the cap, delete an old one before making a new one.
const maxBackupsPerServer = 15

func (d Deps) ListBackups(w http.ResponseWriter, r *http.Request) {
	server := d.loadServerWithPermission(w, r, models.PermFiles)
	if server == nil {
		return
	}
	backups, err := d.ServerSvc.ListBackups(server.ID)
	if err != nil {
		writeError(w, http.StatusBadGateway, "node_dispatch_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"backups": backups})
}

func (d Deps) CreateBackup(w http.ResponseWriter, r *http.Request) {
	server := d.loadServerForWrite(w, r, models.PermFiles)
	if server == nil {
		return
	}
	// Enforce a per-server backup cap so a user or schedule can't fill the
	// node's disk with unbounded archives. Best-effort: if we can't list them
	// (node offline), let the create proceed rather than hard-blocking.
	if existing, err := d.ServerSvc.ListBackups(server.ID); err == nil && len(existing) >= maxBackupsPerServer {
		writeError(w, http.StatusConflict, "backup_limit", fmt.Sprintf("this server already has the maximum of %d backups; delete one before creating another", maxBackupsPerServer))
		return
	}
	result, err := d.ServerSvc.Backup(server.ID)
	if err != nil {
		writeError(w, http.StatusBadGateway, "node_dispatch_failed", err.Error())
		return
	}
	d.audit(r, "server.backup", server.ID, result.Filename)
	writeJSON(w, http.StatusCreated, result)
}

type backupFileRequest struct {
	Filename string `json:"filename"`
}

func (d Deps) RestoreBackup(w http.ResponseWriter, r *http.Request) {
	server := d.loadServerForWrite(w, r, models.PermFiles)
	if server == nil {
		return
	}
	var req backupFileRequest
	if err := decodeJSON(r, &req); err != nil || req.Filename == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "filename is required")
		return
	}
	if !validBackupFilename(req.Filename) {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid backup filename")
		return
	}
	if err := d.ServerSvc.RestoreBackup(server.ID, req.Filename); err != nil {
		writeError(w, http.StatusBadGateway, "node_dispatch_failed", err.Error())
		return
	}
	d.audit(r, "server.backup.restore", server.ID, req.Filename)
	w.WriteHeader(http.StatusNoContent)
}

func (d Deps) DeleteBackup(w http.ResponseWriter, r *http.Request) {
	server := d.loadServerWithPermission(w, r, models.PermFiles)
	if server == nil {
		return
	}
	filename := r.URL.Query().Get("filename")
	if filename == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "filename is required")
		return
	}
	if !validBackupFilename(filename) {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid backup filename")
		return
	}
	if err := d.ServerSvc.DeleteBackup(server.ID, filename); err != nil {
		writeError(w, http.StatusBadGateway, "node_dispatch_failed", err.Error())
		return
	}
	d.audit(r, "server.backup.delete", server.ID, filename)
	w.WriteHeader(http.StatusNoContent)
}
