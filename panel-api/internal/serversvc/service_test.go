package serversvc

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/agenthub"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/repo"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/store"
)

type fakeSender struct {
	commands []agenthub.CommandPayload
	failOn   string // Action to fail, if any
}

func (f *fakeSender) SendCommand(nodeID string, cmd agenthub.CommandPayload) (agenthub.AckPayload, error) {
	f.commands = append(f.commands, cmd)
	if cmd.Action == f.failOn {
		return agenthub.AckPayload{CommandID: cmd.CommandID, OK: false, Error: "boom"}, nil
	}
	return agenthub.AckPayload{CommandID: cmd.CommandID, OK: true}, nil
}

func newTestService(t *testing.T, sender CommandSender) (*Service, *repo.Nodes, *repo.Eggs, *repo.Servers, *repo.Users) {
	t.Helper()

	name := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", name)

	db, err := store.Open(dsn)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	users := repo.NewUsers(db)
	nodes := repo.NewNodes(db)
	eggs := repo.NewEggs(db)
	servers := repo.NewServers(db)
	allocations := repo.NewAllocations(db)

	svc := NewService(servers, eggs, nodes, allocations, sender)
	return svc, nodes, eggs, servers, users
}

func seedUser(t *testing.T, users *repo.Users) *models.User {
	t.Helper()
	now := time.Now().UTC()
	u := &models.User{
		ID:           uuid.NewString(),
		Email:        uuid.NewString() + "@example.com",
		Username:     "owner-" + uuid.NewString(),
		PasswordHash: "hash",
		Role:         models.RoleUser,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := users.Create(u); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return u
}

func seedNode(t *testing.T, nodes *repo.Nodes) *models.Node {
	t.Helper()
	n := &models.Node{ID: uuid.NewString(), Name: "node-1", TokenHash: "hash-" + uuid.NewString(), Address: "127.0.0.1", DockerSocket: "/var/run/docker.sock", CreatedAt: time.Now().UTC()}
	if err := nodes.Create(n); err != nil {
		t.Fatalf("seed node: %v", err)
	}
	return n
}

func seedEgg(t *testing.T, eggs *repo.Eggs) *models.Egg {
	t.Helper()
	e := &models.Egg{
		ID:          uuid.NewString(),
		Name:        "Minecraft",
		DockerImage: "itzg/minecraft-server",
		Startup:     `java -Xmx{{SERVER_MEMORY}}M -jar server.jar --port {{SERVER_PORT}}`,
		Variables: []models.EggVariable{
			{Name: "Memory", Env: "SERVER_MEMORY", Default: "1024", UserEditable: false},
		},
		CreatedAt: time.Now().UTC(),
	}
	if err := eggs.Create(e); err != nil {
		t.Fatalf("seed egg: %v", err)
	}
	return e
}

func TestCreateServerHappyPath(t *testing.T) {
	sender := &fakeSender{}
	svc, nodes, eggs, servers, users := newTestService(t, sender)

	node := seedNode(t, nodes)
	egg := seedEgg(t, eggs)
	owner := seedUser(t, users)

	if err := svc.Allocations.Create(uuid.NewString(), node.ID, 25565); err != nil {
		t.Fatalf("seed allocation: %v", err)
	}

	server, err := svc.CreateServer(owner.ID, node.ID, egg.ID, "My Server", 1024*1024*1024, nil)
	if err != nil {
		t.Fatalf("CreateServer: %v", err)
	}

	if server.Status != models.StatusRunning {
		t.Errorf("expected status running, got %q", server.Status)
	}
	if server.PrimaryPort != 25565 {
		t.Errorf("expected port 25565, got %d", server.PrimaryPort)
	}

	if len(sender.commands) != 2 {
		t.Fatalf("expected 2 dispatched commands (create, start), got %d", len(sender.commands))
	}
	if sender.commands[0].Action != agenthub.ActionCreate || sender.commands[1].Action != agenthub.ActionStart {
		t.Errorf("unexpected command sequence: %+v", sender.commands)
	}

	spec := sender.commands[0].Spec
	if spec == nil {
		t.Fatal("expected a container spec on the create command")
	}
	found := false
	for _, tok := range spec.Cmd {
		if tok == "-Xmx1024M" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected startup command to have memory substituted, got %v", spec.Cmd)
	}

	persisted, err := servers.GetByID(server.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if persisted.Status != models.StatusRunning {
		t.Errorf("expected persisted status running, got %q", persisted.Status)
	}
}

func TestCreateServerNoFreeAllocation(t *testing.T) {
	sender := &fakeSender{}
	svc, nodes, eggs, _, users := newTestService(t, sender)

	node := seedNode(t, nodes)
	egg := seedEgg(t, eggs)
	owner := seedUser(t, users)

	if _, err := svc.CreateServer(owner.ID, node.ID, egg.ID, "My Server", 1024, nil); err == nil {
		t.Error("expected CreateServer to fail with no free allocations")
	}
}

func TestCreateServerDispatchFailureMarksErrored(t *testing.T) {
	sender := &fakeSender{failOn: agenthub.ActionCreate}
	svc, nodes, eggs, servers, users := newTestService(t, sender)

	node := seedNode(t, nodes)
	egg := seedEgg(t, eggs)
	owner := seedUser(t, users)
	if err := svc.Allocations.Create(uuid.NewString(), node.ID, 25565); err != nil {
		t.Fatalf("seed allocation: %v", err)
	}

	server, err := svc.CreateServer(owner.ID, node.ID, egg.ID, "My Server", 1024, nil)
	if err == nil {
		t.Fatal("expected CreateServer to surface the dispatch error")
	}
	if server == nil {
		t.Fatal("expected the server row to still be returned for visibility")
	}

	persisted, err := servers.GetByID(server.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if persisted.Status != models.StatusErrored {
		t.Errorf("expected status errored, got %q", persisted.Status)
	}
}

func TestPowerActionUpdatesStatus(t *testing.T) {
	sender := &fakeSender{}
	svc, nodes, eggs, servers, users := newTestService(t, sender)

	node := seedNode(t, nodes)
	egg := seedEgg(t, eggs)
	owner := seedUser(t, users)
	if err := svc.Allocations.Create(uuid.NewString(), node.ID, 25565); err != nil {
		t.Fatalf("seed allocation: %v", err)
	}

	server, err := svc.CreateServer(owner.ID, node.ID, egg.ID, "My Server", 1024, nil)
	if err != nil {
		t.Fatalf("CreateServer: %v", err)
	}

	if err := svc.PowerAction(server.ID, agenthub.ActionStop); err != nil {
		t.Fatalf("PowerAction stop: %v", err)
	}

	persisted, err := servers.GetByID(server.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if persisted.Status != models.StatusOffline {
		t.Errorf("expected status offline after stop, got %q", persisted.Status)
	}
}

func TestDeleteServerReleasesAllocation(t *testing.T) {
	sender := &fakeSender{}
	svc, nodes, eggs, servers, users := newTestService(t, sender)

	node := seedNode(t, nodes)
	egg := seedEgg(t, eggs)
	owner := seedUser(t, users)
	if err := svc.Allocations.Create(uuid.NewString(), node.ID, 25565); err != nil {
		t.Fatalf("seed allocation: %v", err)
	}

	server, err := svc.CreateServer(owner.ID, node.ID, egg.ID, "My Server", 1024, nil)
	if err != nil {
		t.Fatalf("CreateServer: %v", err)
	}

	if err := svc.DeleteServer(server.ID); err != nil {
		t.Fatalf("DeleteServer: %v", err)
	}

	if _, err := servers.GetByID(server.ID); err != repo.ErrNotFound {
		t.Errorf("expected server to be deleted, got err=%v", err)
	}

	// The allocation should be free again for a new server.
	allocs, err := svc.Allocations.ListByNode(node.ID)
	if err != nil {
		t.Fatalf("ListByNode: %v", err)
	}
	if len(allocs) != 1 || allocs[0].ServerID != nil {
		t.Errorf("expected the allocation to be unclaimed after delete, got %+v", allocs)
	}
}
