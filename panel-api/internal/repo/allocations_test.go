package repo

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
)

// newAllocTestFixture creates the user → node → egg → server graph the FK
// constraints require before an allocation can point at a server, and returns
// the allocations repo plus the node and server ids to work against.
func newAllocTestFixture(t *testing.T) (allocs *Allocations, nodeID, serverID string) {
	t.Helper()
	db := newTestDB(t)
	users := NewUsers(db)
	nodes := NewNodes(db)
	eggs := NewEggs(db)
	servers := NewServers(db)
	allocs = NewAllocations(db)

	owner := newTestUser()
	if err := users.Create(owner); err != nil {
		t.Fatalf("seed owner: %v", err)
	}
	node := &models.Node{ID: uuid.NewString(), Name: "node", TokenHash: "h", Token: "t", Address: "1.2.3.4", DockerSocket: "/x", CreatedAt: owner.CreatedAt, ExpiresAt: owner.CreatedAt}
	if err := nodes.Create(node); err != nil {
		t.Fatalf("seed node: %v", err)
	}
	egg := &models.Egg{ID: uuid.NewString(), Name: "egg", DockerImage: "img", Startup: "run", CreatedAt: owner.CreatedAt}
	if err := eggs.Create(egg); err != nil {
		t.Fatalf("seed egg: %v", err)
	}
	server := &models.Server{ID: uuid.NewString(), OwnerID: owner.ID, NodeID: node.ID, EggID: egg.ID, Name: "s1", Status: models.StatusRunning, CreatedAt: owner.CreatedAt, UpdatedAt: owner.CreatedAt}
	if err := servers.Create(server); err != nil {
		t.Fatalf("seed server: %v", err)
	}
	return allocs, node.ID, server.ID
}

func TestAllocationsClaimSpecificAndListByServer(t *testing.T) {
	allocs, nodeID, serverID := newAllocTestFixture(t)

	if _, err := allocs.CreateRange(nodeID, 25565, 25567); err != nil {
		t.Fatalf("CreateRange: %v", err)
	}

	// Grab the three free allocations' ids in port order.
	free, err := allocs.ListByNode(nodeID)
	if err != nil {
		t.Fatalf("ListByNode: %v", err)
	}
	if len(free) != 3 {
		t.Fatalf("expected 3 allocations, got %d", len(free))
	}

	// Claim the first as the primary and the second as an additional port.
	primary := free[0]
	extra := free[1]
	port, err := allocs.ClaimSpecific(primary.ID, serverID)
	if err != nil {
		t.Fatalf("ClaimSpecific(primary): %v", err)
	}
	if port != 25565 {
		t.Errorf("expected primary port 25565, got %d", port)
	}
	if _, err := allocs.ClaimSpecific(extra.ID, serverID); err != nil {
		t.Fatalf("ClaimSpecific(extra): %v", err)
	}

	// The server now holds two ports, ordered by port.
	held, err := allocs.ListByServer(serverID)
	if err != nil {
		t.Fatalf("ListByServer: %v", err)
	}
	if len(held) != 2 || held[0].Port != 25565 || held[1].Port != 25566 {
		t.Fatalf("unexpected held allocations: %+v", held)
	}

	// Claiming an already-held allocation reports it in use, not free.
	if _, err := allocs.ClaimSpecific(extra.ID, serverID); !errors.Is(err, ErrAllocationInUse) {
		t.Errorf("expected ErrAllocationInUse re-claiming, got %v", err)
	}

	// Claiming a non-existent allocation reports not found.
	if _, err := allocs.ClaimSpecific(uuid.NewString(), serverID); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound for unknown alloc, got %v", err)
	}
}

func TestAllocationsReleaseOne(t *testing.T) {
	allocs, nodeID, serverID := newAllocTestFixture(t)
	if _, err := allocs.CreateRange(nodeID, 25565, 25566); err != nil {
		t.Fatalf("CreateRange: %v", err)
	}
	free, _ := allocs.ListByNode(nodeID)
	primary, extra := free[0], free[1]

	if _, err := allocs.ClaimSpecific(primary.ID, serverID); err != nil {
		t.Fatalf("claim primary: %v", err)
	}
	if _, err := allocs.ClaimSpecific(extra.ID, serverID); err != nil {
		t.Fatalf("claim extra: %v", err)
	}

	// Release just the extra port; the primary stays.
	if err := allocs.ReleaseOne(extra.ID, serverID); err != nil {
		t.Fatalf("ReleaseOne: %v", err)
	}
	held, _ := allocs.ListByServer(serverID)
	if len(held) != 1 || held[0].ID != primary.ID {
		t.Fatalf("expected only the primary to remain, got %+v", held)
	}

	// Releasing a port another server (or nobody) holds is a safe no-op.
	if err := allocs.ReleaseOne(extra.ID, "someone-else"); err != nil {
		t.Fatalf("ReleaseOne no-op: %v", err)
	}
	gotExtra, err := allocs.GetByID(extra.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if gotExtra.ServerID != nil {
		t.Errorf("expected extra to be free, got server_id %v", *gotExtra.ServerID)
	}
}

func TestAllocationsGetByIDNotFound(t *testing.T) {
	allocs, _, _ := newAllocTestFixture(t)
	if _, err := allocs.GetByID(uuid.NewString()); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
