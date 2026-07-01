// Package serversvc contains the business logic for provisioning and
// controlling servers: resolving egg variables into a concrete container
// spec, claiming a port allocation, and dispatching commands to the owning
// node over agenthub.
package serversvc

import (
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/agenthub"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/repo"
)

// CommandSender abstracts agenthub.Registry so this package can be unit
// tested without a real node connection.
type CommandSender interface {
	SendCommand(nodeID string, cmd agenthub.CommandPayload) (agenthub.AckPayload, error)
}

type Service struct {
	Servers     *repo.Servers
	Eggs        *repo.Eggs
	Nodes       *repo.Nodes
	Allocations *repo.Allocations
	Hub         CommandSender
}

func NewService(servers *repo.Servers, eggs *repo.Eggs, nodes *repo.Nodes, allocations *repo.Allocations, hub CommandSender) *Service {
	return &Service{Servers: servers, Eggs: eggs, Nodes: nodes, Allocations: allocations, Hub: hub}
}

var ErrCommandFailed = fmt.Errorf("node reported command failure")

// CreateServer provisions a new server: claims a free port on the node,
// resolves the egg's startup command against the requested variables, and
// asks the node to create + start the container. The server row is
// persisted regardless of whether the node ack succeeds, so a failure is
// visible/retryable rather than silently vanishing.
func (s *Service) CreateServer(ownerID, nodeID, eggID, name string, memoryBytes int64, overrides map[string]string) (*models.Server, error) {
	egg, err := s.Eggs.GetByID(eggID)
	if err != nil {
		return nil, fmt.Errorf("load egg: %w", err)
	}
	if _, err := s.Nodes.GetByID(nodeID); err != nil {
		return nil, fmt.Errorf("load node: %w", err)
	}

	serverID := uuid.NewString()
	now := time.Now().UTC()

	// The server row must exist before an allocation can be claimed for it
	// (allocations.server_id has a foreign key into servers), so it's
	// created up front with a placeholder port and updated once one is
	// claimed.
	server := &models.Server{
		ID:          serverID,
		OwnerID:     ownerID,
		NodeID:      nodeID,
		EggID:       eggID,
		Name:        name,
		Status:      models.StatusInstalling,
		MemoryBytes: memoryBytes,
		Variables:   overrides,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.Servers.Create(server); err != nil {
		return nil, fmt.Errorf("persist server: %w", err)
	}

	port, err := s.Allocations.ClaimFree(nodeID, serverID)
	if err != nil {
		_ = s.Servers.Delete(serverID)
		return nil, fmt.Errorf("claim allocation: %w", err)
	}
	if err := s.Servers.SetPrimaryPort(serverID, port); err != nil {
		return nil, fmt.Errorf("persist allocated port: %w", err)
	}
	server.PrimaryPort = port

	resolved := resolveVariables(egg.Variables, overrides)
	resolved["SERVER_MEMORY"] = strconv.FormatInt(memoryBytes/1024/1024, 10)
	resolved["SERVER_PORT"] = strconv.Itoa(port)
	resolved["SERVER_UUID"] = serverID

	env := make([]string, 0, len(resolved))
	for k, v := range resolved {
		env = append(env, k+"="+v)
	}

	cmd := tokenizeCommand(substitute(egg.Startup, resolved))

	spec := &agenthub.ContainerSpec{
		Name:        "sky-" + serverID,
		Image:       egg.DockerImage,
		Cmd:         cmd,
		Env:         env,
		WorkingDir:  "/home/container",
		Binds:       []string{fmt.Sprintf("/srv/sky-panel/volumes/%s:/home/container", serverID)},
		MemoryBytes: memoryBytes,
		PortBindings: []agenthub.PortBinding{
			{ContainerPort: fmt.Sprintf("%d/tcp", port), HostPort: strconv.Itoa(port)},
			{ContainerPort: fmt.Sprintf("%d/udp", port), HostPort: strconv.Itoa(port)},
		},
		Labels: map[string]string{"sky-panel.server_id": serverID},
	}

	if _, err := s.dispatch(nodeID, agenthub.ActionCreate, serverID, spec); err != nil {
		_ = s.Servers.SetStatus(serverID, models.StatusErrored)
		return server, fmt.Errorf("dispatch create: %w", err)
	}
	if _, err := s.dispatch(nodeID, agenthub.ActionStart, serverID, nil); err != nil {
		_ = s.Servers.SetStatus(serverID, models.StatusErrored)
		return server, fmt.Errorf("dispatch start: %w", err)
	}

	_ = s.Servers.SetStatus(serverID, models.StatusRunning)
	server.Status = models.StatusRunning
	return server, nil
}

// PowerAction sends a start/stop/kill command for an existing server.
func (s *Service) PowerAction(serverID, action string) error {
	server, err := s.Servers.GetByID(serverID)
	if err != nil {
		return fmt.Errorf("load server: %w", err)
	}

	if _, err := s.dispatch(server.NodeID, action, serverID, nil); err != nil {
		return err
	}

	switch action {
	case agenthub.ActionStart:
		return s.Servers.SetStatus(serverID, models.StatusRunning)
	case agenthub.ActionStop, agenthub.ActionKill:
		return s.Servers.SetStatus(serverID, models.StatusOffline)
	default:
		return nil
	}
}

// DeleteServer removes the container on its node, frees the port
// allocation, and deletes the server row.
func (s *Service) DeleteServer(serverID string) error {
	server, err := s.Servers.GetByID(serverID)
	if err != nil {
		return fmt.Errorf("load server: %w", err)
	}

	// Best-effort: if the node is offline the container is gone anyway from
	// the panel's point of view, so proceed with cleanup regardless.
	_, _ = s.dispatch(server.NodeID, agenthub.ActionRemove, serverID, nil)

	if err := s.Allocations.ReleaseByServerID(serverID); err != nil {
		return fmt.Errorf("release allocation: %w", err)
	}
	return s.Servers.Delete(serverID)
}

func (s *Service) dispatch(nodeID, action, serverID string, spec *agenthub.ContainerSpec) (agenthub.AckPayload, error) {
	ack, err := s.Hub.SendCommand(nodeID, agenthub.CommandPayload{
		CommandID: uuid.NewString(),
		Action:    action,
		ServerID:  serverID,
		Spec:      spec,
	})
	if err != nil {
		return ack, err
	}
	if !ack.OK {
		return ack, fmt.Errorf("%w: %s", ErrCommandFailed, ack.Error)
	}
	return ack, nil
}
