package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/agenthub"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
)

// dispatchFileCommand sends a file-manager action to the server's node and
// returns the ack, having already written an error response and returned
// ok=false if anything went wrong (dispatch failure or the daemon itself
// reporting an error).
func (d Deps) dispatchFileCommand(w http.ResponseWriter, server *models.Server, cmd agenthub.CommandPayload) (agenthub.AckPayload, bool) {
	cmd.CommandID = uuid.NewString()
	cmd.ServerID = server.ID

	ack, err := d.AgentHub.Registry.SendCommand(server.NodeID, cmd)
	if err != nil {
		writeError(w, http.StatusBadGateway, "node_dispatch_failed", err.Error())
		return agenthub.AckPayload{}, false
	}
	if !ack.OK {
		writeError(w, http.StatusBadGateway, "node_dispatch_failed", ack.Error)
		return agenthub.AckPayload{}, false
	}
	return ack, true
}

func (d Deps) ListFiles(w http.ResponseWriter, r *http.Request) {
	server := d.loadServerWithPermission(w, r, models.PermFiles)
	if server == nil {
		return
	}

	ack, ok := d.dispatchFileCommand(w, server, agenthub.CommandPayload{
		Action: agenthub.ActionListFiles,
		Path:   r.URL.Query().Get("path"),
	})
	if !ok {
		return
	}

	var result agenthub.ListFilesResult
	if err := json.Unmarshal(ack.Result, &result); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to decode file listing")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (d Deps) ReadFile(w http.ResponseWriter, r *http.Request) {
	server := d.loadServerWithPermission(w, r, models.PermFiles)
	if server == nil {
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "path is required")
		return
	}

	ack, ok := d.dispatchFileCommand(w, server, agenthub.CommandPayload{Action: agenthub.ActionReadFile, Path: path})
	if !ok {
		return
	}

	var result agenthub.ReadFileResult
	if err := json.Unmarshal(ack.Result, &result); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to decode file content")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

type writeFileRequest struct {
	Path          string `json:"path"`
	ContentBase64 string `json:"content_base64"`
}

func (d Deps) WriteFile(w http.ResponseWriter, r *http.Request) {
	server := d.loadServerWithPermission(w, r, models.PermFiles)
	if server == nil {
		return
	}

	var req writeFileRequest
	if err := decodeJSON(r, &req); err != nil || req.Path == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "path and content_base64 are required")
		return
	}

	_, ok := d.dispatchFileCommand(w, server, agenthub.CommandPayload{
		Action: agenthub.ActionWriteFile, Path: req.Path, ContentBase64: req.ContentBase64,
	})
	if !ok {
		return
	}
	d.audit(r, "server.file.write", server.ID, req.Path)
	w.WriteHeader(http.StatusNoContent)
}

type renameFileRequest struct {
	Path    string `json:"path"`
	NewPath string `json:"new_path"`
}

func (d Deps) RenameFile(w http.ResponseWriter, r *http.Request) {
	server := d.loadServerWithPermission(w, r, models.PermFiles)
	if server == nil {
		return
	}

	var req renameFileRequest
	if err := decodeJSON(r, &req); err != nil || req.Path == "" || req.NewPath == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "path and new_path are required")
		return
	}

	_, ok := d.dispatchFileCommand(w, server, agenthub.CommandPayload{
		Action: agenthub.ActionRenameFile, Path: req.Path, NewPath: req.NewPath,
	})
	if !ok {
		return
	}
	d.audit(r, "server.file.rename", server.ID, req.Path+" -> "+req.NewPath)
	w.WriteHeader(http.StatusNoContent)
}

func (d Deps) DeleteFile(w http.ResponseWriter, r *http.Request) {
	server := d.loadServerWithPermission(w, r, models.PermFiles)
	if server == nil {
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "path is required")
		return
	}

	_, ok := d.dispatchFileCommand(w, server, agenthub.CommandPayload{Action: agenthub.ActionDeleteFile, Path: path})
	if !ok {
		return
	}
	d.audit(r, "server.file.delete", server.ID, path)
	w.WriteHeader(http.StatusNoContent)
}

type mkdirRequest struct {
	Path string `json:"path"`
}

func (d Deps) Mkdir(w http.ResponseWriter, r *http.Request) {
	server := d.loadServerWithPermission(w, r, models.PermFiles)
	if server == nil {
		return
	}

	var req mkdirRequest
	if err := decodeJSON(r, &req); err != nil || req.Path == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "path is required")
		return
	}

	_, ok := d.dispatchFileCommand(w, server, agenthub.CommandPayload{Action: agenthub.ActionMkdir, Path: req.Path})
	if !ok {
		return
	}
	d.audit(r, "server.file.mkdir", server.ID, req.Path)
	w.WriteHeader(http.StatusNoContent)
}
