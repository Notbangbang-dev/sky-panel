package repo

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
)

func seedServer(t *testing.T, nodes *Nodes, eggs *Eggs, servers *Servers, owner *models.User) *models.Server {
	t.Helper()
	node := &models.Node{ID: uuid.NewString(), Name: "node", TokenHash: uuid.NewString(), Token: "tok", Address: "1.2.3.4", DockerSocket: "/x", CreatedAt: owner.CreatedAt, ExpiresAt: owner.CreatedAt}
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
	return server
}

func TestDatabasesCRUDAndCount(t *testing.T) {
	db := newTestDB(t)
	users := NewUsers(db)
	nodes := NewNodes(db)
	eggs := NewEggs(db)
	servers := NewServers(db)
	databases := NewDatabases(db)

	owner := newTestUser()
	if err := users.Create(owner); err != nil {
		t.Fatalf("seed owner: %v", err)
	}
	server := seedServer(t, nodes, eggs, servers, owner)

	d := &models.Database{
		ID: uuid.NewString(), OwnerID: owner.ID, ServerID: server.ID, NodeID: server.NodeID,
		Name: "sky_ab12_test", Username: "sky_deadbeef1234", Password: "secret", Host: "1.2.3.4", Port: 3306,
		CreatedAt: time.Now().UTC(),
	}
	if err := databases.Create(d); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := databases.GetByID(d.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != d.Name || got.Username != d.Username || got.Port != 3306 {
		t.Fatalf("round-trip mismatch: %+v", got)
	}

	n, err := databases.CountByOwner(owner.ID)
	if err != nil || n != 1 {
		t.Fatalf("CountByOwner = %d, %v; want 1", n, err)
	}

	exists, err := databases.NameExistsOnNode(server.NodeID, "sky_ab12_test")
	if err != nil || !exists {
		t.Fatalf("NameExistsOnNode = %v, %v; want true", exists, err)
	}

	list, err := databases.ListByServer(server.ID)
	if err != nil || len(list) != 1 {
		t.Fatalf("ListByServer len=%d err=%v; want 1", len(list), err)
	}

	if err := databases.Delete(d.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := databases.GetByID(d.ID); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestQuotasAddDatabases(t *testing.T) {
	db := newTestDB(t)
	users := NewUsers(db)
	quotas := NewQuotas(db)

	owner := newTestUser()
	if err := users.Create(owner); err != nil {
		t.Fatalf("seed owner: %v", err)
	}

	// No row yet → zero bonus.
	b, err := quotas.Get(owner.ID)
	if err != nil || b.Databases != 0 {
		t.Fatalf("initial bonus.Databases = %d, %v; want 0", b.Databases, err)
	}

	if err := quotas.AddDatabases(owner.ID, 3); err != nil {
		t.Fatalf("AddDatabases: %v", err)
	}
	if err := quotas.AddDatabases(owner.ID, 2); err != nil {
		t.Fatalf("AddDatabases: %v", err)
	}

	// Adding memory bonus must not clobber the database bonus.
	if err := quotas.Add(owner.ID, 1024, 0, 0); err != nil {
		t.Fatalf("Add: %v", err)
	}

	b, err = quotas.Get(owner.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if b.Databases != 5 {
		t.Fatalf("bonus.Databases = %d; want 5", b.Databases)
	}
	if b.MemoryBytes != 1024 {
		t.Fatalf("bonus.MemoryBytes = %d; want 1024", b.MemoryBytes)
	}
}
