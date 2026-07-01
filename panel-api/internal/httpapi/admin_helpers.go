package httpapi

import (
	"log"
	"net/http"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/auth"
)

// audit records an admin action against whichever user is authenticated on
// r. Failures are logged rather than surfaced to the caller — a missed
// audit row should never block the underlying action from succeeding.
func (d Deps) audit(r *http.Request, action, target, metadata string) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		return
	}
	if err := d.Audit.Record(claims.UserID, action, target, metadata); err != nil {
		log.Printf("httpapi: failed to record audit entry (action=%s target=%s): %v", action, target, err)
	}
}
