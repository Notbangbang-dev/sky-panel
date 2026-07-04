package httpapi

import (
	"log"
	"net/http"
	"unicode/utf8"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/auth"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/serversvc"
)

// CloneServer creates a new server that mirrors an existing one's egg, node,
// resource limits and variables. It's a fresh server (new ID, new port, empty
// volume) owned by the caller — files are NOT copied, only the configuration.
func (d Deps) CloneServer(w http.ResponseWriter, r *http.Request) {
	src := d.loadOwnedServer(w, r)
	if src == nil {
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	// Serialize with the user's other create/clone requests so the quota check
	// and the insert below are atomic per user (see keyedMutex).
	unlock := serverCreateLocks.lock(claims.UserID)
	defer unlock()

	// The clone counts against the caller's quota (admins are unmetered).
	if claims.Role != string(models.RoleAdmin) {
		if err := d.QuotaSvc.CheckCreate(claims.UserID, src.MemoryBytes, src.CPULimit, src.DiskBytes); err != nil {
			d.writeQuotaError(w, err)
			return
		}
	}

	// Truncate on a rune boundary, not a byte boundary — a byte slice can cut
	// through a multibyte UTF-8 rune and persist an invalid name.
	name := "Copy of " + src.Name
	if utf8.RuneCountInString(name) > 60 {
		name = string([]rune(name)[:60])
	}
	server, err := d.ServerSvc.CreateServer(claims.UserID, src.NodeID, src.EggID, name, src.MemoryBytes, src.CPULimit, src.DiskBytes, src.Variables)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	d.audit(r, "server.clone", server.ID, src.ID)

	go func(serverID, name string) {
		defer func() {
			if p := recover(); p != nil {
				log.Printf("server %s clone provisioning panicked: %v", serverID, p)
			}
		}()
		if err := d.ServerSvc.Provision(serverID, serversvc.ProvisionCreateTimeout); err != nil {
			log.Printf("server %s (%s) clone provisioning failed: %v", serverID, name, err)
		}
	}(server.ID, server.Name)

	writeJSON(w, http.StatusCreated, toServerResponse(server))
}
