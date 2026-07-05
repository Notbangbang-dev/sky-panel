package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/repo"
)

type setPublicStatusRequest struct {
	Public bool `json:"public"`
}

// SetServerPublicStatus toggles whether a server exposes its public status page.
func (d Deps) SetServerPublicStatus(w http.ResponseWriter, r *http.Request) {
	server := d.loadServerWithPermission(w, r, models.PermSettings)
	if server == nil {
		return
	}
	var req setPublicStatusRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	if err := d.Servers.SetPublicStatus(server.ID, req.Public); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to update")
		return
	}
	d.audit(r, "server.public_status", server.ID, map[bool]string{true: "on", false: "off"}[req.Public])
	w.WriteHeader(http.StatusNoContent)
}

type publicServerStatus struct {
	Name          string   `json:"name"`
	Online        bool     `json:"online"`
	Players       []string `json:"players"`
	PlayerCount   int      `json:"player_count"`
	MaxPlayers    int      `json:"max_players"`
	Version       string   `json:"version"`
	CPUPercent    float64  `json:"cpu_percent"`
	MemUsedBytes  uint64   `json:"mem_used_bytes"`
	MemLimitBytes uint64   `json:"mem_limit_bytes"`
}

// PublicServerStatus is an unauthenticated, read-only status page feed for a
// server whose owner opted in. Servers that don't opt in return 404 so this
// can't be used to probe which server ids exist.
func (d Deps) PublicServerStatus(w http.ResponseWriter, r *http.Request) {
	server, err := d.Servers.GetByID(pathParam(r, "serverID"))
	if errors.Is(err, repo.ErrNotFound) || (err == nil && !server.PublicStatus) {
		writeError(w, http.StatusNotFound, "not_found", "no public status for this server")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load server")
		return
	}

	online := server.Status == models.StatusRunning
	resp := publicServerStatus{
		Name:    server.Name,
		Online:  online,
		Players: []string{},
	}

	if online {
		info := d.AgentHub.Players(server.ID)
		if info.Players != nil {
			resp.Players = info.Players
		}
		resp.PlayerCount = len(resp.Players)
		resp.MaxPlayers = info.Max
		resp.Version = info.Version

		if raw, ok := d.AgentHub.LatestStats(server.ID); ok {
			var hb struct {
				CPUPercent    float64 `json:"cpu_percent"`
				MemUsedBytes  uint64  `json:"mem_used_bytes"`
				MemLimitBytes uint64  `json:"mem_limit_bytes"`
			}
			if json.Unmarshal(raw, &hb) == nil {
				resp.CPUPercent = hb.CPUPercent
				resp.MemUsedBytes = hb.MemUsedBytes
				resp.MemLimitBytes = hb.MemLimitBytes
			}
		}
	}

	writeJSON(w, http.StatusOK, resp)
}
