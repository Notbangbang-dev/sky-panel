package agentclient

import (
	"context"
	"testing"

	"github.com/Notbangbang-dev/sky-panel/node-agent/internal/protocol"
	"github.com/Notbangbang-dev/sky-panel/node-agent/internal/runtime"
)

func TestDispatcherCreateStartStopRemove(t *testing.T) {
	ctx := context.Background()
	d := NewDispatcher(runtime.NewFake())

	createAck := d.Handle(ctx, protocol.CommandPayload{
		CommandID: "1",
		Action:    protocol.ActionCreate,
		ServerID:  "server-1",
		Spec:      &protocol.ContainerSpec{Image: "itzg/minecraft-server"},
	})
	if !createAck.OK {
		t.Fatalf("create failed: %s", createAck.Error)
	}

	startAck := d.Handle(ctx, protocol.CommandPayload{CommandID: "2", Action: protocol.ActionStart, ServerID: "server-1"})
	if !startAck.OK {
		t.Fatalf("start failed: %s", startAck.Error)
	}

	stopAck := d.Handle(ctx, protocol.CommandPayload{CommandID: "3", Action: protocol.ActionStop, ServerID: "server-1"})
	if !stopAck.OK {
		t.Fatalf("stop failed: %s", stopAck.Error)
	}

	removeAck := d.Handle(ctx, protocol.CommandPayload{CommandID: "4", Action: protocol.ActionRemove, ServerID: "server-1"})
	if !removeAck.OK {
		t.Fatalf("remove failed: %s", removeAck.Error)
	}

	// After remove, the server is no longer tracked, so a follow-up action
	// referencing it by server ID alone must fail.
	startAgainAck := d.Handle(ctx, protocol.CommandPayload{CommandID: "5", Action: protocol.ActionStart, ServerID: "server-1"})
	if startAgainAck.OK {
		t.Error("expected start on a removed server to fail")
	}
}

func TestDispatcherUnknownAction(t *testing.T) {
	ctx := context.Background()
	d := NewDispatcher(runtime.NewFake())

	ack := d.Handle(ctx, protocol.CommandPayload{CommandID: "1", Action: "not-a-real-action", ServerID: "server-1"})
	if ack.OK {
		t.Error("expected unknown action to fail")
	}
}

func TestDispatcherStartWithExplicitContainerID(t *testing.T) {
	ctx := context.Background()
	rt := runtime.NewFake()
	d := NewDispatcher(rt)

	id, err := rt.Create(ctx, runtime.ContainerSpec{Image: "test"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// No prior "create" through the dispatcher, so server-1 isn't tracked
	// yet — but an explicit container_id should still work.
	ack := d.Handle(ctx, protocol.CommandPayload{CommandID: "1", Action: protocol.ActionStart, ServerID: "server-1", ContainerID: id})
	if !ack.OK {
		t.Fatalf("start with explicit container id failed: %s", ack.Error)
	}
}

func TestDispatcherConsoleInput(t *testing.T) {
	ctx := context.Background()
	rt := runtime.NewFake()
	d := NewDispatcher(rt)

	d.Handle(ctx, protocol.CommandPayload{CommandID: "1", Action: protocol.ActionCreate, ServerID: "server-1", Spec: &protocol.ContainerSpec{Image: "test"}})
	d.Handle(ctx, protocol.CommandPayload{CommandID: "2", Action: protocol.ActionStart, ServerID: "server-1"})

	ack := d.Handle(ctx, protocol.CommandPayload{CommandID: "3", Action: protocol.ActionConsole, ServerID: "server-1", Input: "say hello"})
	if !ack.OK {
		t.Fatalf("console input failed: %s", ack.Error)
	}

	id, err := d.containerFor(protocol.CommandPayload{ServerID: "server-1"})
	if err != nil {
		t.Fatalf("containerFor: %v", err)
	}

	writes := rt.ConsoleWrites(id)
	if len(writes) != 1 || writes[0] != "say hello\n" {
		t.Errorf("expected console input to be written, got %v", writes)
	}
}

func TestDispatcherHeartbeatReportsTrackedContainers(t *testing.T) {
	ctx := context.Background()
	d := NewDispatcher(runtime.NewFake())

	d.Handle(ctx, protocol.CommandPayload{CommandID: "1", Action: protocol.ActionCreate, ServerID: "server-1", Spec: &protocol.ContainerSpec{Image: "test"}})
	d.Handle(ctx, protocol.CommandPayload{CommandID: "2", Action: protocol.ActionStart, ServerID: "server-1"})

	hb := d.Heartbeat(ctx)
	if len(hb.Containers) != 1 {
		t.Fatalf("expected 1 container in heartbeat, got %d", len(hb.Containers))
	}
	if hb.Containers[0].ServerID != "server-1" || !hb.Containers[0].Running {
		t.Errorf("unexpected heartbeat entry: %+v", hb.Containers[0])
	}
}
