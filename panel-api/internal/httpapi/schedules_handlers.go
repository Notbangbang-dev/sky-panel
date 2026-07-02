package httpapi

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
)

type scheduleResponse struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Action          string `json:"action"`
	Payload         string `json:"payload,omitempty"`
	IntervalMinutes int    `json:"interval_minutes"`
	Enabled         bool   `json:"enabled"`
	LastRunAt       string `json:"last_run_at,omitempty"`
}

func toScheduleResponse(s *models.Schedule) scheduleResponse {
	resp := scheduleResponse{
		ID: s.ID, Name: s.Name, Action: s.Action, Payload: s.Payload,
		IntervalMinutes: s.IntervalMinutes, Enabled: s.Enabled,
	}
	if s.LastRunAt != nil {
		resp.LastRunAt = s.LastRunAt.Format(rfc3339)
	}
	return resp
}

var validScheduleActions = map[string]bool{
	models.ScheduleStart: true, models.ScheduleStop: true, models.ScheduleRestart: true,
	models.ScheduleKill: true, models.ScheduleBackup: true, models.ScheduleCommand: true,
}

func (d Deps) ListSchedules(w http.ResponseWriter, r *http.Request) {
	server := d.loadServerWithPermission(w, r, "")
	if server == nil {
		return
	}
	schedules, err := d.Schedules.ListByServer(server.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list schedules")
		return
	}
	out := make([]scheduleResponse, 0, len(schedules))
	for _, s := range schedules {
		out = append(out, toScheduleResponse(s))
	}
	writeJSON(w, http.StatusOK, out)
}

type createScheduleRequest struct {
	Name            string `json:"name"`
	Action          string `json:"action"`
	Payload         string `json:"payload"`
	IntervalMinutes int    `json:"interval_minutes"`
}

func (d Deps) CreateSchedule(w http.ResponseWriter, r *http.Request) {
	server := d.loadServerWithPermission(w, r, models.PermSettings)
	if server == nil {
		return
	}
	var req createScheduleRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	if !validScheduleActions[req.Action] {
		writeError(w, http.StatusBadRequest, "bad_request", "action must be one of: start, stop, restart, kill, backup, command")
		return
	}
	if req.IntervalMinutes < 1 {
		writeError(w, http.StatusBadRequest, "bad_request", "interval_minutes must be at least 1")
		return
	}
	if req.Action == models.ScheduleCommand && req.Payload == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "a command schedule needs a command payload")
		return
	}

	sch := &models.Schedule{
		ID:              uuid.NewString(),
		ServerID:        server.ID,
		Name:            req.Name,
		Action:          req.Action,
		Payload:         req.Payload,
		IntervalMinutes: req.IntervalMinutes,
		Enabled:         true,
		CreatedAt:       time.Now().UTC(),
	}
	if err := d.Schedules.Create(sch); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to create schedule")
		return
	}
	d.audit(r, "server.schedule.create", server.ID, req.Action)
	writeJSON(w, http.StatusCreated, toScheduleResponse(sch))
}

type toggleScheduleRequest struct {
	Enabled bool `json:"enabled"`
}

func (d Deps) ToggleSchedule(w http.ResponseWriter, r *http.Request) {
	server := d.loadServerWithPermission(w, r, models.PermSettings)
	if server == nil {
		return
	}
	var req toggleScheduleRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	if err := d.Schedules.SetEnabled(pathParam(r, "scheduleID"), server.ID, req.Enabled); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to update schedule")
		return
	}
	d.audit(r, "server.schedule.toggle", server.ID, pathParam(r, "scheduleID"))
	w.WriteHeader(http.StatusNoContent)
}

func (d Deps) DeleteSchedule(w http.ResponseWriter, r *http.Request) {
	server := d.loadServerWithPermission(w, r, models.PermSettings)
	if server == nil {
		return
	}
	if err := d.Schedules.Delete(pathParam(r, "scheduleID"), server.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete schedule")
		return
	}
	d.audit(r, "server.schedule.delete", server.ID, "")
	w.WriteHeader(http.StatusNoContent)
}
