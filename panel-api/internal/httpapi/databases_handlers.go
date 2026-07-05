package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/repo"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/serversvc"
)

// databaseCreateLocks serializes per-owner database creation so the quota
// check and the row insert can't interleave across concurrent requests (a
// TOCTOU that would otherwise let a burst of parallel creates overshoot the
// quota). Same pattern as serverCreateLocks.
var databaseCreateLocks = newKeyedMutex()

// Databases are gated behind the "databases" permission. Credentials are stored
// by the panel and shown to whoever can manage the server's databases.

func (d Deps) ListDatabases(w http.ResponseWriter, r *http.Request) {
	server := d.loadServerWithPermission(w, r, models.PermDatabases)
	if server == nil {
		return
	}
	dbs, err := d.Databases.ListByServer(server.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list databases")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"databases": dbs})
}

type createDatabaseRequest struct {
	Name string `json:"name"`
}

func (d Deps) CreateDatabase(w http.ResponseWriter, r *http.Request) {
	server := d.loadServerWithPermission(w, r, models.PermDatabases)
	if server == nil {
		return
	}
	var req createDatabaseRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid body")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || len(req.Name) > 32 {
		writeError(w, http.StatusBadRequest, "bad_request", "name must be 1-32 characters")
		return
	}

	// Serialize per owner so concurrent creates can't each pass the quota check
	// against the same pre-insert count and collectively overshoot it.
	unlock := databaseCreateLocks.lock(server.OwnerID)
	defer unlock()

	// Databases count against the server owner's quota (bought from the store).
	if err := d.QuotaSvc.CheckDatabaseCreate(server.OwnerID); err != nil {
		d.writeQuotaError(w, err)
		return
	}

	creds, err := d.ServerSvc.CreateDatabase(server.ID, req.Name)
	if err != nil {
		if errors.Is(err, serversvc.ErrDatabasesUnavailable) {
			writeError(w, http.StatusConflict, "databases_unavailable", err.Error())
			return
		}
		writeError(w, http.StatusBadGateway, "node_dispatch_failed", err.Error())
		return
	}

	db := &models.Database{
		ID:        uuid.NewString(),
		OwnerID:   server.OwnerID,
		ServerID:  server.ID,
		NodeID:    server.NodeID,
		Name:      creds.Name,
		Username:  creds.Username,
		Password:  creds.Password,
		Host:      creds.Host,
		Port:      creds.Port,
		CreatedAt: time.Now().UTC(),
	}
	if err := d.Databases.Create(db); err != nil {
		if errors.Is(err, repo.ErrDuplicate) {
			// The name is already tracked (astronomically unlikely given the
			// random suffix + pre-check). Do NOT drop it on the node — it may
			// belong to another record. Leave it and ask the caller to retry.
			writeError(w, http.StatusConflict, "name_conflict", "database name collision, please try again")
			return
		}
		// The database exists on the node under our unique name but we couldn't
		// persist it — roll it back so we don't strand an untracked database.
		_ = d.ServerSvc.DeleteDatabase(server.NodeID, creds.Name, creds.Username)
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to save database")
		return
	}
	d.audit(r, "database.create", db.ID, db.Name)
	writeJSON(w, http.StatusCreated, db)
}

func (d Deps) DeleteDatabase(w http.ResponseWriter, r *http.Request) {
	server := d.loadServerWithPermission(w, r, models.PermDatabases)
	if server == nil {
		return
	}
	db, err := d.Databases.GetByID(pathParam(r, "databaseID"))
	if errors.Is(err, repo.ErrNotFound) || (err == nil && db.ServerID != server.ID) {
		writeError(w, http.StatusNotFound, "not_found", "database not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load database")
		return
	}
	// Best-effort drop on the node; remove the panel row regardless so a broken
	// node can't strand the record.
	_ = d.ServerSvc.DeleteDatabase(db.NodeID, db.Name, db.Username)
	if err := d.Databases.Delete(db.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete database")
		return
	}
	d.audit(r, "database.delete", db.ID, db.Name)
	w.WriteHeader(http.StatusNoContent)
}
