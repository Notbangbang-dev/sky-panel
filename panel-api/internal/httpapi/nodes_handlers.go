package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/auth"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/repo"
)

// DefaultNodeTokenTTL is how long a freshly (re)issued node token is valid
// before it must be rotated.
const DefaultNodeTokenTTL = 90 * 24 * time.Hour

type createNodeRequest struct {
	Name         string `json:"name"`
	Address      string `json:"address"`
	DockerSocket string `json:"docker_socket,omitempty"`
}

type nodeResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Address      string `json:"address"`
	DockerSocket string `json:"docker_socket"`
	ExpiresAt    string `json:"expires_at"`
}

// createNodeResponse additionally carries the plaintext node token, shown
// exactly once, that the operator pastes into that node's SKY_NODE_TOKEN.
type createNodeResponse struct {
	nodeResponse
	NodeToken string `json:"node_token"`
}

func toNodeResponse(n *models.Node) nodeResponse {
	return nodeResponse{
		ID: n.ID, Name: n.Name, Address: n.Address, DockerSocket: n.DockerSocket, ExpiresAt: n.ExpiresAt.Format(rfc3339),
	}
}

func (d Deps) CreateNode(w http.ResponseWriter, r *http.Request) {
	var req createNodeRequest
	if err := decodeJSON(r, &req); err != nil || req.Name == "" || req.Address == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "name and address are required")
		return
	}

	dockerSocket := req.DockerSocket
	if dockerSocket == "" {
		dockerSocket = "/var/run/docker.sock"
	}

	rawToken, err := auth.NewOpaqueToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to generate node token")
		return
	}

	now := time.Now().UTC()
	node := &models.Node{
		ID:           uuid.NewString(),
		Name:         req.Name,
		TokenHash:    auth.HashToken(rawToken),
		Token:        rawToken,
		ExpiresAt:    now.Add(DefaultNodeTokenTTL),
		Address:      req.Address,
		DockerSocket: dockerSocket,
		CreatedAt:    now,
	}

	if err := d.Nodes.Create(node); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to create node")
		return
	}
	d.audit(r, "node.create", node.ID, node.Name)

	// Seed a default block of port allocations so servers can be created on
	// this node immediately, out of the box. Best-effort: a seeding hiccup
	// shouldn't fail node registration — the operator can add ports later in
	// the Allocations tab.
	if _, err := d.Allocations.CreateRange(node.ID, DefaultAllocationStart, DefaultAllocationStart+DefaultAllocationCount-1); err == nil {
		d.audit(r, "allocation.seed", node.ID, fmt.Sprintf("%d-%d", DefaultAllocationStart, DefaultAllocationStart+DefaultAllocationCount-1))
	}

	writeJSON(w, http.StatusCreated, createNodeResponse{nodeResponse: toNodeResponse(node), NodeToken: rawToken})
}

func (d Deps) ListNodes(w http.ResponseWriter, r *http.Request) {
	nodes, err := d.Nodes.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list nodes")
		return
	}

	out := make([]nodeResponse, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, toNodeResponse(n))
	}
	writeJSON(w, http.StatusOK, out)
}

type nodeSummary struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Address   string `json:"address"`
	Connected bool   `json:"connected"`
}

// ListNodesSlim is the non-admin node listing: just enough for a regular
// user to pick a node when creating a server (no docker_socket/token/expiry,
// which stay admin-only via ListNodes).
func (d Deps) ListNodesSlim(w http.ResponseWriter, r *http.Request) {
	nodes, err := d.Nodes.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list nodes")
		return
	}

	out := make([]nodeSummary, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, nodeSummary{ID: n.ID, Name: n.Name, Address: n.Address, Connected: d.AgentHub.Registry.Connected(n.ID)})
	}
	writeJSON(w, http.StatusOK, out)
}

func (d Deps) DeleteNode(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "nodeID")

	// Refuse to delete a node that still hosts servers. Deleting it would
	// CASCADE those server (and database/allocation) rows away, orphaning the
	// real Docker containers and MariaDB databases on the box with no pointer
	// left to reclaim them. The operator must delete or move the servers first.
	if n, err := d.Servers.CountByNode(id); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to check node servers")
		return
	} else if n > 0 {
		writeError(w, http.StatusConflict, "node_in_use", fmt.Sprintf("this node still hosts %d server(s); delete or move them before removing the node", n))
		return
	}

	if err := d.Nodes.Delete(id); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "node not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete node")
		return
	}
	// Sever any live daemon connection so its now-deleted token stops working.
	d.AgentHub.Registry.Close(id)
	d.audit(r, "node.delete", id, "")
	w.WriteHeader(http.StatusNoContent)
}

// RotateNodeToken issues a fresh token (and expiry) for a node, immediately
// invalidating the old one. The node's SKY_NODE_TOKEN env/config must be
// updated with the new value and the daemon restarted.
func (d Deps) RotateNodeToken(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "nodeID")

	if _, err := d.Nodes.GetByID(id); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "node not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load node")
		return
	}

	rawToken, err := auth.NewOpaqueToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to generate node token")
		return
	}
	expiresAt := time.Now().UTC().Add(DefaultNodeTokenTTL)

	if err := d.Nodes.RotateToken(id, auth.HashToken(rawToken), rawToken, expiresAt); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to rotate node token")
		return
	}
	// Drop the live connection authenticated with the old token so it can't keep
	// operating until it happens to reconnect — the daemon must re-hello with the
	// new token.
	d.AgentHub.Registry.Close(id)
	d.audit(r, "node.rotate_token", id, "")

	writeJSON(w, http.StatusOK, map[string]string{"node_token": rawToken, "expires_at": expiresAt.Format(rfc3339)})
}
