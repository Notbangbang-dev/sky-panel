package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type broadcastRequest struct {
	Message string `json:"message"`
}

type broadcastMessage struct {
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
}

// AdminBroadcast pushes a message to every browser client subscribed to the
// "broadcast" topic (any authenticated user may subscribe — see
// Deps.authorizedForTopic).
func (d Deps) AdminBroadcast(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	var req broadcastRequest
	if err := decodeJSON(r, &req); err != nil || req.Message == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "message is required")
		return
	}
	req.Message = strings.TrimSpace(req.Message)
	if req.Message == "" || len(req.Message) > 2000 {
		writeError(w, http.StatusBadRequest, "bad_request", "message must be 1-2000 characters")
		return
	}

	msg, err := json.Marshal(broadcastMessage{Message: req.Message, CreatedAt: time.Now().UTC().Format(rfc3339)})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to encode broadcast")
		return
	}

	d.Hub.Broadcast("broadcast", msg)
	d.audit(r, "broadcast.send", "", req.Message)
	w.WriteHeader(http.StatusNoContent)
}
