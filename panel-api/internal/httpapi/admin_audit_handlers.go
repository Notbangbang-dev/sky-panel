package httpapi

import "net/http"

type auditEntryResponse struct {
	ActorID   string `json:"actor_id"`
	Action    string `json:"action"`
	Target    string `json:"target,omitempty"`
	Metadata  string `json:"metadata,omitempty"`
	CreatedAt string `json:"created_at"`
}

func (d Deps) AdminListAuditLog(w http.ResponseWriter, r *http.Request) {
	entries, err := d.Audit.List(200)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load audit log")
		return
	}

	out := make([]auditEntryResponse, 0, len(entries))
	for _, e := range entries {
		out = append(out, auditEntryResponse{
			ActorID: e.ActorID, Action: e.Action, Target: e.Target, Metadata: e.Metadata, CreatedAt: e.CreatedAt.Format(rfc3339),
		})
	}
	writeJSON(w, http.StatusOK, out)
}
