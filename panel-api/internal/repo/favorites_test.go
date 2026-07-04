package repo

import (
	"testing"

	"github.com/google/uuid"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
)

func TestFavoritesAddListRemove(t *testing.T) {
	db := newTestDB(t)
	users := NewUsers(db)
	nodes := NewNodes(db)
	eggs := NewEggs(db)
	servers := NewServers(db)
	favorites := NewFavorites(db)

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

	// Add is idempotent (ON CONFLICT DO NOTHING).
	if err := favorites.Add(owner.ID, server.ID); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := favorites.Add(owner.ID, server.ID); err != nil {
		t.Fatalf("Add (again): %v", err)
	}

	ids, err := favorites.ListByUser(owner.ID)
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if len(ids) != 1 || ids[0] != server.ID {
		t.Fatalf("expected [%s], got %v", server.ID, ids)
	}

	if err := favorites.Remove(owner.ID, server.ID); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	ids, err = favorites.ListByUser(owner.ID)
	if err != nil {
		t.Fatalf("ListByUser after remove: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("expected no favorites, got %v", ids)
	}
}

// TestFavoriteCascadesOnServerDelete guards the FK-cascade fix: foreign_keys
// must be enabled on the connection so deleting a server drops its favorite
// rows automatically, rather than leaving orphans.
func TestFavoriteCascadesOnServerDelete(t *testing.T) {
	db := newTestDB(t)
	users := NewUsers(db)
	nodes := NewNodes(db)
	eggs := NewEggs(db)
	servers := NewServers(db)
	favorites := NewFavorites(db)

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
	if err := favorites.Add(owner.ID, server.ID); err != nil {
		t.Fatalf("Add: %v", err)
	}

	if err := servers.Delete(server.ID); err != nil {
		t.Fatalf("Delete server: %v", err)
	}

	ids, err := favorites.ListByUser(owner.ID)
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("expected favorite to cascade-delete with its server, got %v (foreign_keys not enabled?)", ids)
	}
}
