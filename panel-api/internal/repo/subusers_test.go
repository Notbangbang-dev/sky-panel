package repo

import (
	"testing"

	"github.com/google/uuid"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
)

func TestSubusersCreateGetListDelete(t *testing.T) {
	db := newTestDB(t)
	users := NewUsers(db)
	nodes := NewNodes(db)
	eggs := NewEggs(db)
	servers := NewServers(db)
	subusers := NewSubusers(db)

	owner := newTestUser()
	if err := users.Create(owner); err != nil {
		t.Fatalf("seed owner: %v", err)
	}
	invitee := newTestUser()
	invitee.Email = "invitee@example.com"
	invitee.Username = "invitee"
	if err := users.Create(invitee); err != nil {
		t.Fatalf("seed invitee: %v", err)
	}

	node := &models.Node{ID: uuid.NewString(), Name: "node", TokenHash: "hash", Token: "tok", Address: "1.2.3.4", DockerSocket: "/x", CreatedAt: owner.CreatedAt, ExpiresAt: owner.CreatedAt}
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

	if err := subusers.Create(uuid.NewString(), server.ID, invitee.ID, []string{models.PermConsole, models.PermFiles}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := subusers.Get(server.ID, invitee.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !got.HasPermission(models.PermConsole) || !got.HasPermission(models.PermFiles) {
		t.Errorf("expected console+files permissions, got %+v", got.Permissions)
	}
	if got.HasPermission(models.PermPower) {
		t.Errorf("did not expect power permission")
	}

	list, err := subusers.ListByServer(server.ID)
	if err != nil {
		t.Fatalf("ListByServer: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 subuser, got %d", len(list))
	}

	if err := subusers.Delete(server.ID, invitee.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := subusers.Get(server.ID, invitee.ID); err != ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestSubusersCreateDuplicateFails(t *testing.T) {
	db := newTestDB(t)
	users := NewUsers(db)
	nodes := NewNodes(db)
	eggs := NewEggs(db)
	servers := NewServers(db)
	subusers := NewSubusers(db)

	owner := newTestUser()
	users.Create(owner)
	invitee := newTestUser()
	invitee.Email = "invitee2@example.com"
	invitee.Username = "invitee2"
	users.Create(invitee)

	node := &models.Node{ID: uuid.NewString(), Name: "node", TokenHash: "hash2", Token: "tok2", Address: "1.2.3.4", DockerSocket: "/x", CreatedAt: owner.CreatedAt, ExpiresAt: owner.CreatedAt}
	nodes.Create(node)
	egg := &models.Egg{ID: uuid.NewString(), Name: "egg", DockerImage: "img", Startup: "run", CreatedAt: owner.CreatedAt}
	eggs.Create(egg)
	server := &models.Server{ID: uuid.NewString(), OwnerID: owner.ID, NodeID: node.ID, EggID: egg.ID, Name: "s1", Status: models.StatusRunning, CreatedAt: owner.CreatedAt, UpdatedAt: owner.CreatedAt}
	servers.Create(server)

	if err := subusers.Create(uuid.NewString(), server.ID, invitee.ID, []string{models.PermConsole}); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	if err := subusers.Create(uuid.NewString(), server.ID, invitee.ID, []string{models.PermFiles}); err != ErrDuplicate {
		t.Errorf("expected ErrDuplicate on second invite of the same user, got %v", err)
	}
}
