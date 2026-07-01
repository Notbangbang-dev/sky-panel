// Package agentclient implements node-agent's side of the persistent
// outbound WebSocket connection to panel-api: it dials out (so nodes behind
// NAT/firewalls need no inbound ports), sends a hello + periodic heartbeats,
// and dispatches incoming commands to a ContainerRuntime.
package agentclient

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Notbangbang-dev/sky-panel/node-agent/internal/protocol"
	"github.com/Notbangbang-dev/sky-panel/node-agent/internal/runtime"
)

// Dispatcher executes commands from panel-api against a ContainerRuntime and
// tracks which container backs which server so heartbeats can report stats
// for all of them.
type Dispatcher struct {
	rt runtime.ContainerRuntime

	mu      sync.Mutex
	tracked map[string]string // serverID -> containerID
}

func NewDispatcher(rt runtime.ContainerRuntime) *Dispatcher {
	return &Dispatcher{rt: rt, tracked: make(map[string]string)}
}

// Handle executes one command and returns the Ack to send back. It never
// returns an error itself — failures are reported inside AckPayload so the
// caller always has something to send upstream.
func (d *Dispatcher) Handle(ctx context.Context, cmd protocol.CommandPayload) protocol.AckPayload {
	err := d.handle(ctx, cmd)
	ack := protocol.AckPayload{CommandID: cmd.CommandID, OK: err == nil}
	if err != nil {
		ack.Error = err.Error()
	}
	return ack
}

func (d *Dispatcher) handle(ctx context.Context, cmd protocol.CommandPayload) error {
	switch cmd.Action {
	case protocol.ActionCreate:
		if cmd.Spec == nil {
			return fmt.Errorf("create command missing spec")
		}
		id, err := d.rt.Create(ctx, toRuntimeSpec(*cmd.Spec))
		if err != nil {
			return err
		}
		d.track(cmd.ServerID, id)
		return nil

	case protocol.ActionStart:
		id, err := d.containerFor(cmd)
		if err != nil {
			return err
		}
		return d.rt.Start(ctx, id)

	case protocol.ActionStop:
		id, err := d.containerFor(cmd)
		if err != nil {
			return err
		}
		return d.rt.Stop(ctx, id, 15*time.Second)

	case protocol.ActionKill:
		id, err := d.containerFor(cmd)
		if err != nil {
			return err
		}
		return d.rt.Kill(ctx, id)

	case protocol.ActionRemove:
		id, err := d.containerFor(cmd)
		if err != nil {
			return err
		}
		if err := d.rt.Remove(ctx, id); err != nil {
			return err
		}
		d.untrack(cmd.ServerID)
		return nil

	case protocol.ActionConsole:
		id, err := d.containerFor(cmd)
		if err != nil {
			return err
		}
		console, err := d.rt.Attach(ctx, id)
		if err != nil {
			return err
		}
		defer console.Close()
		_, err = console.Write([]byte(cmd.Input + "\n"))
		return err

	default:
		return fmt.Errorf("unknown command action %q", cmd.Action)
	}
}

func (d *Dispatcher) track(serverID, containerID string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.tracked[serverID] = containerID
}

func (d *Dispatcher) untrack(serverID string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.tracked, serverID)
}

func (d *Dispatcher) containerFor(cmd protocol.CommandPayload) (string, error) {
	if cmd.ContainerID != "" {
		return cmd.ContainerID, nil
	}

	d.mu.Lock()
	id, ok := d.tracked[cmd.ServerID]
	d.mu.Unlock()
	if !ok {
		return "", fmt.Errorf("no known container for server %q", cmd.ServerID)
	}
	return id, nil
}

// Heartbeat reports live stats for every tracked container. Containers that
// fail to report (e.g. mid-removal) are skipped rather than failing the
// whole heartbeat.
func (d *Dispatcher) Heartbeat(ctx context.Context) protocol.HeartbeatPayload {
	d.mu.Lock()
	snapshot := make(map[string]string, len(d.tracked))
	for k, v := range d.tracked {
		snapshot[k] = v
	}
	d.mu.Unlock()

	payload := protocol.HeartbeatPayload{}
	for serverID, containerID := range snapshot {
		state, err := d.rt.Inspect(ctx, containerID)
		if err != nil {
			continue
		}

		hb := protocol.ContainerHeartbeat{ServerID: serverID, Running: state.Running}
		if state.Running {
			if stats, err := d.rt.Stats(ctx, containerID); err == nil {
				hb.CPU = stats.CPUPercent
				hb.MemUsed = stats.MemoryUsedBytes
				hb.MemLimit = stats.MemoryLimitBytes
				hb.NetRx = stats.NetworkRxBytes
				hb.NetTx = stats.NetworkTxBytes
			}
		}
		payload.Containers = append(payload.Containers, hb)
	}
	return payload
}

func toRuntimeSpec(s protocol.ContainerSpec) runtime.ContainerSpec {
	bindings := make(map[string]string, len(s.PortBindings))
	for _, pb := range s.PortBindings {
		bindings[pb.ContainerPort] = pb.HostPort
	}

	return runtime.ContainerSpec{
		Name:         s.Name,
		Image:        s.Image,
		Cmd:          s.Cmd,
		Env:          s.Env,
		WorkingDir:   s.WorkingDir,
		Binds:        s.Binds,
		PortBindings: bindings,
		MemoryBytes:  s.MemoryBytes,
		NanoCPUs:     s.NanoCPUs,
		Labels:       s.Labels,
	}
}
