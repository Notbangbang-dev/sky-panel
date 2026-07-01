package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/auth"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/repo"
)

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
}

// createNodeResponse additionally carries the plaintext node token, shown
// exactly once, that the operator pastes into that node's SKY_NODE_TOKEN.
type createNodeResponse struct {
	nodeResponse
	NodeToken string `json:"node_token"`
}

func toNodeResponse(n *models.Node) nodeResponse {
	return nodeResponse{ID: n.ID, Name: n.Name, Address: n.Address, DockerSocket: n.DockerSocket}
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

	node := &models.Node{
		ID:           uuid.NewString(),
		Name:         req.Name,
		TokenHash:    auth.HashToken(rawToken),
		Address:      req.Address,
		DockerSocket: dockerSocket,
		CreatedAt:    time.Now().UTC(),
	}

	if err := d.Nodes.Create(node); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to create node")
		return
	}
	d.audit(r, "node.create", node.ID, node.Name)

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

func (d Deps) DeleteNode(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "nodeID")
	if err := d.Nodes.Delete(id); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "node not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete node")
		return
	}
	d.audit(r, "node.delete", id, "")
	w.WriteHeader(http.StatusNoContent)
}
