package serversvc

import (
	"encoding/json"
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
	noPull   bool   // simulate a legacy node without pull_image support
}

func (f *fakeSender) SendCommand(nodeID string, cmd agenthub.CommandPayload) (agenthub.AckPayload, error) {
	f.commands = append(f.commands, cmd)
	if cmd.Action == f.failOn {
		return agenthub.AckPayload{CommandID: cmd.CommandID, OK: false, Error: "boom"}, nil
	}
	return agenthub.AckPayload{CommandID: cmd.CommandID, OK: true}, nil
}

func (f *fakeSender) SendCommandTimeout(nodeID string, cmd agenthub.CommandPayload, _ time.Duration) (agenthub.AckPayload, error) {
	return f.SendCommand(nodeID, cmd)
}

func (f *fakeSender) ConnectedNodeIDs() []string { return nil }

func (f *fakeSender) SupportsPullImage(string) bool { return !f.noPull }

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

	server, err := svc.CreateServer(owner.ID, node.ID, egg.ID, "My Server", 1024*1024*1024, 0, 0, nil)
	if err != nil {
		t.Fatalf("CreateServer: %v", err)
	}

	// CreateServer only prepares the row + port; it comes back "installing".
	if server.Status != models.StatusInstalling {
		t.Errorf("expected status installing right after create, got %q", server.Status)
	}
	if server.PrimaryPort != 25565 {
		t.Errorf("expected port 25565, got %d", server.PrimaryPort)
	}

	// Provisioning (run in the background in production) does the dispatch.
	if err := svc.Provision(server.ID, defaultProvisionTimeout); err != nil {
		t.Fatalf("Provision: %v", err)
	}

	if len(sender.commands) != 3 {
		t.Fatalf("expected 3 dispatched commands (pull_image, create, start), got %d", len(sender.commands))
	}
	if sender.commands[0].Action != agenthub.ActionPullImage ||
		sender.commands[1].Action != agenthub.ActionCreate ||
		sender.commands[2].Action != agenthub.ActionStart {
		t.Errorf("unexpected command sequence: %+v", sender.commands)
	}
	if sender.commands[0].Image != egg.DockerImage {
		t.Errorf("expected pull_image to carry the egg image %q, got %q", egg.DockerImage, sender.commands[0].Image)
	}

	spec := sender.commands[1].Spec
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

func TestEmptyStartupSerializesCmdAsEmptyListNotNull(t *testing.T) {
	// Regression: an egg with no startup command must not put "cmd":null on the
	// wire — the daemon can't decode null into a list and drops the connection
	// ("node reported command failure: node disconnected").
	sender := &fakeSender{}
	svc, nodes, eggs, _, users := newTestService(t, sender)

	node := seedNode(t, nodes)
	owner := seedUser(t, users)
	egg := &models.Egg{
		ID: uuid.NewString(), Name: "Paper", DockerImage: "itzg/minecraft-server",
		Startup: "", CreatedAt: time.Now().UTC(), // no startup command
	}
	if err := eggs.Create(egg); err != nil {
		t.Fatalf("seed egg: %v", err)
	}
	if err := svc.Allocations.Create(uuid.NewString(), node.ID, 25565); err != nil {
		t.Fatalf("seed allocation: %v", err)
	}

	server, err := svc.CreateServer(owner.ID, node.ID, egg.ID, "Sky", 1024, 0, 0, nil)
	if err != nil {
		t.Fatalf("CreateServer: %v", err)
	}
	if err := svc.Provision(server.ID, defaultProvisionTimeout); err != nil {
		t.Fatalf("Provision: %v", err)
	}

	var createCmd *agenthub.CommandPayload
	for i := range sender.commands {
		if sender.commands[i].Action == agenthub.ActionCreate {
			createCmd = &sender.commands[i]
		}
	}
	if createCmd == nil || createCmd.Spec == nil {
		t.Fatal("expected a create command with a spec")
	}
	if createCmd.Spec.Cmd == nil {
		t.Error("Cmd must be a non-nil empty slice, not nil (which marshals to null)")
	}
	raw, _ := json.Marshal(createCmd.Spec)
	if !strings.Contains(string(raw), `"cmd":[]`) || strings.Contains(string(raw), `"cmd":null`) {
		t.Errorf("expected \"cmd\":[] on the wire, got: %s", raw)
	}
}

func TestProvisionLegacyNodeSkipsPullImage(t *testing.T) {
	// A node whose daemon predates pull_image must never be sent that command
	// (an old daemon can't decode it and drops the connection). Provisioning
	// falls back to create + start only.
	sender := &fakeSender{noPull: true}
	svc, nodes, eggs, servers, users := newTestService(t, sender)

	node := seedNode(t, nodes)
	egg := seedEgg(t, eggs)
	owner := seedUser(t, users)
	if err := svc.Allocations.Create(uuid.NewString(), node.ID, 25565); err != nil {
		t.Fatalf("seed allocation: %v", err)
	}

	server, err := svc.CreateServer(owner.ID, node.ID, egg.ID, "Legacy", 1024, 0, 0, nil)
	if err != nil {
		t.Fatalf("CreateServer: %v", err)
	}
	if err := svc.Provision(server.ID, defaultProvisionTimeout); err != nil {
		t.Fatalf("Provision: %v", err)
	}

	for _, cmd := range sender.commands {
		if cmd.Action == agenthub.ActionPullImage {
			t.Fatalf("legacy node must not receive pull_image, got commands %+v", sender.commands)
		}
	}
	if len(sender.commands) != 2 ||
		sender.commands[0].Action != agenthub.ActionCreate ||
		sender.commands[1].Action != agenthub.ActionStart {
		t.Errorf("expected legacy sequence [create, start], got %+v", sender.commands)
	}

	if persisted, _ := servers.GetByID(server.ID); persisted.Status != models.StatusRunning {
		t.Errorf("expected running, got %q", persisted.Status)
	}
}

func TestCreateServerNoFreeAllocation(t *testing.T) {
	sender := &fakeSender{}
	svc, nodes, eggs, _, users := newTestService(t, sender)

	node := seedNode(t, nodes)
	egg := seedEgg(t, eggs)
	owner := seedUser(t, users)

	if _, err := svc.CreateServer(owner.ID, node.ID, egg.ID, "My Server", 1024, 0, 0, nil); err == nil {
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

	server, err := svc.CreateServer(owner.ID, node.ID, egg.ID, "My Server", 1024, 0, 0, nil)
	if err != nil {
		t.Fatalf("CreateServer (prepare) should succeed, got: %v", err)
	}

	// Provisioning fails at the create dispatch and must mark the server errored.
	if err := svc.Provision(server.ID, defaultProvisionTimeout); err == nil {
		t.Fatal("expected Provision to surface the dispatch error")
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

	server, err := svc.CreateServer(owner.ID, node.ID, egg.ID, "My Server", 1024, 0, 0, nil)
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

	server, err := svc.CreateServer(owner.ID, node.ID, egg.ID, "My Server", 1024, 0, 0, nil)
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
