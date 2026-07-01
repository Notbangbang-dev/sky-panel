package runtime

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Fake is an in-memory ContainerRuntime used by tests (and available for
// local dev without Docker installed). It tracks enough state to make
// dispatch-logic tests meaningful without touching a real container engine.
type Fake struct {
	mu         sync.Mutex
	containers map[string]*fakeContainer
}

type fakeContainer struct {
	spec    ContainerSpec
	running bool
	console *fakeConsole
}

func NewFake() *Fake {
	return &Fake{containers: make(map[string]*fakeContainer)}
}

func (f *Fake) Create(_ context.Context, spec ContainerSpec) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	id := uuid.NewString()
	f.containers[id] = &fakeContainer{spec: spec}
	return id, nil
}

func (f *Fake) Start(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	c, ok := f.containers[id]
	if !ok {
		return fmt.Errorf("fake runtime: container %q not found", id)
	}
	c.running = true
	return nil
}

func (f *Fake) Stop(_ context.Context, id string, _ time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	c, ok := f.containers[id]
	if !ok {
		return fmt.Errorf("fake runtime: container %q not found", id)
	}
	c.running = false
	return nil
}

func (f *Fake) Kill(ctx context.Context, id string) error {
	return f.Stop(ctx, id, 0)
}

func (f *Fake) Remove(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, ok := f.containers[id]; !ok {
		return fmt.Errorf("fake runtime: container %q not found", id)
	}
	delete(f.containers, id)
	return nil
}

func (f *Fake) Inspect(_ context.Context, id string) (ContainerState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	c, ok := f.containers[id]
	if !ok {
		return ContainerState{}, fmt.Errorf("fake runtime: container %q not found", id)
	}
	return ContainerState{ID: id, Running: c.running}, nil
}

func (f *Fake) Stats(_ context.Context, id string) (Stats, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, ok := f.containers[id]; !ok {
		return Stats{}, fmt.Errorf("fake runtime: container %q not found", id)
	}
	// Deterministic non-zero fixture values so callers can assert plumbing
	// works without depending on real container metrics.
	return Stats{
		CPUPercent:       12.5,
		MemoryUsedBytes:  128 * 1024 * 1024,
		MemoryLimitBytes: 512 * 1024 * 1024,
		NetworkRxBytes:   1024,
		NetworkTxBytes:   2048,
		DiskReadBytes:    4096,
		DiskWriteBytes:   8192,
	}, nil
}

func (f *Fake) Attach(_ context.Context, id string) (Console, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	c, ok := f.containers[id]
	if !ok {
		return nil, fmt.Errorf("fake runtime: container %q not found", id)
	}
	if c.console == nil {
		c.console = newFakeConsole()
	}
	return c.console, nil
}

// ConsoleWrites returns everything written to id's stdin via Attach, for
// test assertions.
func (f *Fake) ConsoleWrites(id string) []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	c, ok := f.containers[id]
	if !ok || c.console == nil {
		return nil
	}
	return c.console.Writes
}

// fakeConsole lets tests both feed the "output" side and inspect what was
// written to "stdin".
type fakeConsole struct {
	mu     sync.Mutex
	output chan string
	closed bool
	Writes []string
}

func newFakeConsole() *fakeConsole {
	return &fakeConsole{output: make(chan string, 64)}
}

func (c *fakeConsole) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Writes = append(c.Writes, string(p))
	return len(p), nil
}

func (c *fakeConsole) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		close(c.output)
		c.closed = true
	}
	return nil
}

func (c *fakeConsole) Output() <-chan string {
	return c.output
}

// Emit pushes a fake output line, for use by tests driving Attach.
func (c *fakeConsole) Emit(line string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.output <- line
	}
}
