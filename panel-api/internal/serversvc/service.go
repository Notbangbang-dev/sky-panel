// Package serversvc contains the business logic for provisioning and
// controlling servers: resolving egg variables into a concrete container
// spec, claiming a port allocation, and dispatching commands to the owning
// node over agenthub.
package serversvc

import (
	"encoding/json"
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
	SendCommandTimeout(nodeID string, cmd agenthub.CommandPayload, timeout time.Duration) (agenthub.AckPayload, error)
}

const (
	// defaultProvisionTimeout bounds an ordinary dispatch (image already
	// present on the node).
	defaultProvisionTimeout = 15 * time.Second
	// ProvisionCreateTimeout bounds a first-time container create, which may
	// pull a large image from a registry (minutes). CreateServer provisions in
	// the background so this long wait never blocks the HTTP request.
	ProvisionCreateTimeout = 10 * time.Minute
)

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
func (s *Service) CreateServer(ownerID, nodeID, eggID, name string, memoryBytes int64, cpuLimit int, diskBytes int64, overrides map[string]string) (*models.Server, error) {
	if _, err := s.Eggs.GetByID(eggID); err != nil {
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
		CPULimit:    cpuLimit,
		DiskBytes:   diskBytes,
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

	// The server row + port are ready; the container itself is provisioned
	// separately (see Provision), because a first-time create may pull a large
	// image and take minutes. The server is returned "installing".
	return server, nil
}

// Provision creates and starts a prepared server's container on its node and
// records the resulting status (running on success, errored on failure).
// It's normally run in a background goroutine right after CreateServer, since
// the create step may pull a large image. createTimeout bounds that wait.
func (s *Service) Provision(serverID string, createTimeout time.Duration) error {
	server, err := s.Servers.GetByID(serverID)
	if err != nil {
		return fmt.Errorf("load server: %w", err)
	}
	egg, err := s.Eggs.GetByID(server.EggID)
	if err != nil {
		return fmt.Errorf("load egg: %w", err)
	}
	if err := s.provision(server, egg, createTimeout); err != nil {
		_ = s.Servers.SetError(serverID, err.Error())
		return err
	}
	return s.Servers.SetStatus(serverID, models.StatusRunning)
}

// UpdateServer applies edited settings (name, resource limits, egg
// variables, backup schedule) and re-provisions the container so the new
// spec takes effect. The server's volume (and therefore its data) is
// preserved — only the container is recreated.
func (s *Service) UpdateServer(serverID, name string, memoryBytes int64, cpuLimit int, diskBytes int64, overrides map[string]string, backupIntervalHours int) (*models.Server, error) {
	if err := s.Servers.UpdateSettings(serverID, name, memoryBytes, cpuLimit, diskBytes, overrides, backupIntervalHours); err != nil {
		return nil, fmt.Errorf("persist settings: %w", err)
	}

	server, err := s.Servers.GetByID(serverID)
	if err != nil {
		return nil, fmt.Errorf("reload server: %w", err)
	}
	egg, err := s.Eggs.GetByID(server.EggID)
	if err != nil {
		return nil, fmt.Errorf("load egg: %w", err)
	}

	// Recreate the container with the new spec (remove is best-effort — the
	// container may not exist if it was never started or the node was down).
	_, _ = s.dispatch(server.NodeID, agenthub.ActionRemove, serverID, nil)
	if err := s.provision(server, egg, defaultProvisionTimeout); err != nil {
		_ = s.Servers.SetError(serverID, err.Error())
		return server, err
	}
	_ = s.Servers.SetStatus(serverID, models.StatusRunning)
	server.Status = models.StatusRunning
	return server, nil
}

// ReinstallServer recreates the container from its egg, re-running the
// image's install/startup against the (preserved) volume — a fresh
// container without wiping the server's files.
// If eggID is non-empty and differs from the server's current egg, the server
// is switched onto that egg (new image/startup) before the reinstall — its
// volume is still preserved, though the new software may not understand the
// old data. Pass "" to reinstall onto the same egg.
func (s *Service) ReinstallServer(serverID, eggID string) error {
	server, err := s.Servers.GetByID(serverID)
	if err != nil {
		return fmt.Errorf("load server: %w", err)
	}

	if eggID != "" && eggID != server.EggID {
		if _, err := s.Eggs.GetByID(eggID); err != nil {
			return fmt.Errorf("load target egg: %w", err)
		}
		if err := s.Servers.SetEgg(serverID, eggID); err != nil {
			return fmt.Errorf("switch egg: %w", err)
		}
		server.EggID = eggID
	}

	egg, err := s.Eggs.GetByID(server.EggID)
	if err != nil {
		return fmt.Errorf("load egg: %w", err)
	}

	// Show "installing" while we re-pull/recreate (may take minutes), and use
	// the long create timeout so a retry of a failed install can finish.
	_ = s.Servers.SetStatus(serverID, models.StatusInstalling)
	_, _ = s.dispatch(server.NodeID, agenthub.ActionRemove, serverID, nil)
	if err := s.provision(server, egg, ProvisionCreateTimeout); err != nil {
		_ = s.Servers.SetError(serverID, err.Error())
		return err
	}
	return s.Servers.SetStatus(serverID, models.StatusRunning)
}

// SuspendServer flags a server as suspended and stops its container. The stop
// is best-effort (the node may be offline); the flag is what actually blocks
// the owner from starting it again.
func (s *Service) SuspendServer(serverID string) error {
	server, err := s.Servers.GetByID(serverID)
	if err != nil {
		return fmt.Errorf("load server: %w", err)
	}
	if err := s.Servers.SetSuspended(serverID, true); err != nil {
		return err
	}
	_, _ = s.dispatch(server.NodeID, agenthub.ActionStop, serverID, nil)
	_ = s.Servers.SetStatus(serverID, models.StatusOffline)
	return nil
}

// UnsuspendServer clears the suspension flag, letting the owner control the
// server again. It does not auto-start it.
func (s *Service) UnsuspendServer(serverID string) error {
	if _, err := s.Servers.GetByID(serverID); err != nil {
		return fmt.Errorf("load server: %w", err)
	}
	return s.Servers.SetSuspended(serverID, false)
}

// provision builds the container spec for a server and dispatches
// create + start to its node. The create dispatch gets createTimeout (it may
// pull an image); the start that follows uses the ordinary timeout.
func (s *Service) provision(server *models.Server, egg *models.Egg, createTimeout time.Duration) error {
	spec := s.buildSpec(server, egg)
	if _, err := s.dispatchTimeout(server.NodeID, agenthub.ActionCreate, server.ID, spec, createTimeout); err != nil {
		return fmt.Errorf("dispatch create: %w", err)
	}
	if _, err := s.dispatch(server.NodeID, agenthub.ActionStart, server.ID, nil); err != nil {
		return fmt.Errorf("dispatch start: %w", err)
	}
	return nil
}

// buildSpec turns a server + its egg into a concrete container spec:
// resolves variables, injects the built-in SERVER_* and MEMORY env vars,
// substitutes the startup command, and maps resource limits.
func (s *Service) buildSpec(server *models.Server, egg *models.Egg) *agenthub.ContainerSpec {
	mib := server.MemoryBytes / 1024 / 1024

	resolved := resolveVariables(egg.Variables, server.Variables)
	// MEMORY defaults to the server's memory limit (itzg images size their
	// JVM heap from it) unless the egg explicitly defines its own.
	if _, ok := resolved["MEMORY"]; !ok {
		resolved["MEMORY"] = fmt.Sprintf("%dM", mib)
	}
	resolved["SERVER_MEMORY"] = strconv.FormatInt(mib, 10)
	resolved["SERVER_PORT"] = strconv.Itoa(server.PrimaryPort)
	resolved["SERVER_UUID"] = server.ID

	env := make([]string, 0, len(resolved))
	for k, v := range resolved {
		env = append(env, k+"="+v)
	}

	// CPU limit is a percentage of one core; Docker wants nano-CPUs
	// (1 core = 1e9). 0 leaves it unlimited.
	nanoCPUs := int64(server.CPULimit) * 10_000_000

	return &agenthub.ContainerSpec{
		Name:        "sky-" + server.ID,
		Image:       egg.DockerImage,
		Cmd:         tokenizeCommand(substitute(egg.Startup, resolved)),
		Env:         env,
		WorkingDir:  "/home/container",
		Binds:       []string{fmt.Sprintf("/srv/sky-panel/volumes/%s:/home/container", server.ID)},
		MemoryBytes: server.MemoryBytes,
		NanoCPUs:    nanoCPUs,
		PortBindings: []agenthub.PortBinding{
			{ContainerPort: fmt.Sprintf("%d/tcp", server.PrimaryPort), HostPort: strconv.Itoa(server.PrimaryPort)},
			{ContainerPort: fmt.Sprintf("%d/udp", server.PrimaryPort), HostPort: strconv.Itoa(server.PrimaryPort)},
		},
		Labels: map[string]string{"sky-panel.server_id": server.ID},
	}
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

// Backup dispatches a backup command and, on success, records the time so
// the scheduler knows when the next scheduled backup is due.
func (s *Service) Backup(serverID string) (agenthub.BackupResult, error) {
	server, err := s.Servers.GetByID(serverID)
	if err != nil {
		return agenthub.BackupResult{}, fmt.Errorf("load server: %w", err)
	}
	ack, err := s.dispatch(server.NodeID, agenthub.ActionBackup, serverID, nil)
	if err != nil {
		return agenthub.BackupResult{}, err
	}
	var result agenthub.BackupResult
	if len(ack.Result) > 0 {
		_ = json.Unmarshal(ack.Result, &result)
	}
	_ = s.Servers.MarkBackedUp(serverID, time.Now().UTC())
	return result, nil
}

// ListBackups asks the node for the backups it holds for a server.
func (s *Service) ListBackups(serverID string) ([]agenthub.BackupEntry, error) {
	server, err := s.Servers.GetByID(serverID)
	if err != nil {
		return nil, fmt.Errorf("load server: %w", err)
	}
	ack, err := s.dispatch(server.NodeID, agenthub.ActionListBackups, serverID, nil)
	if err != nil {
		return nil, err
	}
	var result agenthub.ListBackupsResult
	if len(ack.Result) > 0 {
		if err := json.Unmarshal(ack.Result, &result); err != nil {
			return nil, fmt.Errorf("decode backup list: %w", err)
		}
	}
	return result.Backups, nil
}

// RestoreBackup / DeleteBackup act on a single backup archive by filename.
func (s *Service) RestoreBackup(serverID, filename string) error {
	return s.dispatchFile(serverID, agenthub.ActionRestoreBackup, filename)
}

func (s *Service) DeleteBackup(serverID, filename string) error {
	return s.dispatchFile(serverID, agenthub.ActionDeleteBackup, filename)
}

func (s *Service) dispatchFile(serverID, action, path string) error {
	server, err := s.Servers.GetByID(serverID)
	if err != nil {
		return fmt.Errorf("load server: %w", err)
	}
	ack, err := s.Hub.SendCommand(server.NodeID, agenthub.CommandPayload{
		CommandID: uuid.NewString(),
		Action:    action,
		ServerID:  serverID,
		Path:      path,
	})
	if err != nil {
		return err
	}
	if !ack.OK {
		return fmt.Errorf("%w: %s", ErrCommandFailed, ack.Error)
	}
	return nil
}

// SendConsole writes a line to a running server's console (used by scheduled
// "command" automations and reused by the HTTP console handler's needs).
func (s *Service) SendConsole(serverID, input string) error {
	server, err := s.Servers.GetByID(serverID)
	if err != nil {
		return fmt.Errorf("load server: %w", err)
	}
	ack, err := s.Hub.SendCommand(server.NodeID, agenthub.CommandPayload{
		CommandID: uuid.NewString(),
		Action:    agenthub.ActionConsole,
		ServerID:  serverID,
		Input:     input,
	})
	if err != nil {
		return err
	}
	if !ack.OK {
		return fmt.Errorf("%w: %s", ErrCommandFailed, ack.Error)
	}
	return nil
}

// RunScheduleAction executes one automation action against a server.
func (s *Service) RunScheduleAction(serverID, action, payload string) error {
	switch action {
	case models.ScheduleStart:
		return s.PowerAction(serverID, agenthub.ActionStart)
	case models.ScheduleStop:
		return s.PowerAction(serverID, agenthub.ActionStop)
	case models.ScheduleKill:
		return s.PowerAction(serverID, agenthub.ActionKill)
	case models.ScheduleRestart:
		if err := s.PowerAction(serverID, agenthub.ActionStop); err != nil {
			return err
		}
		return s.PowerAction(serverID, agenthub.ActionStart)
	case models.ScheduleBackup:
		_, err := s.Backup(serverID)
		return err
	case models.ScheduleCommand:
		return s.SendConsole(serverID, payload)
	default:
		return fmt.Errorf("unknown schedule action %q", action)
	}
}

func (s *Service) dispatch(nodeID, action, serverID string, spec *agenthub.ContainerSpec) (agenthub.AckPayload, error) {
	return s.dispatchTimeout(nodeID, action, serverID, spec, defaultProvisionTimeout)
}

func (s *Service) dispatchTimeout(nodeID, action, serverID string, spec *agenthub.ContainerSpec, timeout time.Duration) (agenthub.AckPayload, error) {
	ack, err := s.Hub.SendCommandTimeout(nodeID, agenthub.CommandPayload{
		CommandID: uuid.NewString(),
		Action:    action,
		ServerID:  serverID,
		Spec:      spec,
	}, timeout)
	if err != nil {
		return ack, err
	}
	if !ack.OK {
		return ack, fmt.Errorf("%w: %s", ErrCommandFailed, ack.Error)
	}
	return ack, nil
}
