// Package runtime defines the ContainerRuntime abstraction node-agent uses
// to drive game server processes. The real implementation (Docker) talks to
// the Docker Engine API over the node's local unix socket; a fake
// implementation backs unit tests so agent dispatch logic can be verified on
// a machine with no Docker installed.
package runtime

import (
	"context"
	"io"
	"time"
)

// ContainerSpec describes everything needed to create one server's
// container. Cmd is already fully tokenized (egg variable substitution
// happens upstream in panel-api before dispatch).
type ContainerSpec struct {
	Name         string
	Image        string
	Cmd          []string
	Env          []string          // "KEY=VALUE"
	WorkingDir   string            // container-side working directory
	Binds        []string          // "hostPath:containerPath[:ro]"
	PortBindings map[string]string // "port/tcp" or "port/udp" -> host port
	MemoryBytes  int64             // 0 = unlimited
	NanoCPUs     int64             // 0 = unlimited; 1 CPU = 1_000_000_000
	Labels       map[string]string
}

type ContainerState struct {
	ID       string
	Running  bool
	ExitCode int
}

type Stats struct {
	CPUPercent       float64
	MemoryUsedBytes  uint64
	MemoryLimitBytes uint64
	NetworkRxBytes   uint64
	NetworkTxBytes   uint64
	DiskReadBytes    uint64
	DiskWriteBytes   uint64
}

// Console is a live attached session to a running container: Write sends
// stdin, and lines received on Output() are de-multiplexed combined
// stdout+stderr text lines. Close detaches (it does NOT stop the container).
type Console interface {
	io.Writer
	io.Closer
	Output() <-chan string
}

type ContainerRuntime interface {
	Create(ctx context.Context, spec ContainerSpec) (id string, err error)
	Start(ctx context.Context, id string) error
	Stop(ctx context.Context, id string, timeout time.Duration) error
	Kill(ctx context.Context, id string) error
	Remove(ctx context.Context, id string) error
	Inspect(ctx context.Context, id string) (ContainerState, error)
	Stats(ctx context.Context, id string) (Stats, error)
	Attach(ctx context.Context, id string) (Console, error)
}
