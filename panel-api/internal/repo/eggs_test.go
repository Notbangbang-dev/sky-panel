package repo

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
)

func TestEggsCreateGetUpdateDelete(t *testing.T) {
	db := newTestDB(t)
	eggs := NewEggs(db)

	egg := &models.Egg{
		ID:          uuid.NewString(),
		Name:        "Paper",
		Category:    "Minecraft",
		DockerImage: "itzg/minecraft-server",
		Startup:     "",
		Variables: []models.EggVariable{
			{Name: "EULA", Env: "EULA", Default: "TRUE", UserEditable: true},
		},
		CreatedAt: time.Now().UTC(),
	}
	if err := eggs.Create(egg); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := eggs.GetByID(egg.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Startup != "" {
		t.Errorf("expected empty startup to round-trip as empty, got %q", got.Startup)
	}
	if len(got.Variables) != 1 || got.Variables[0].Env != "EULA" || got.Variables[0].Default != "TRUE" {
		t.Errorf("unexpected variables: %+v", got.Variables)
	}

	got.Name = "Paper (updated)"
	got.Variables = append(got.Variables, models.EggVariable{Name: "VERSION", Env: "VERSION", Default: "LATEST", UserEditable: true})
	if err := eggs.Update(got); err != nil {
		t.Fatalf("Update: %v", err)
	}

	updated, err := eggs.GetByID(egg.ID)
	if err != nil {
		t.Fatalf("GetByID after update: %v", err)
	}
	if updated.Name != "Paper (updated)" {
		t.Errorf("expected updated name, got %q", updated.Name)
	}
	if len(updated.Variables) != 2 {
		t.Errorf("expected 2 variables after update, got %d", len(updated.Variables))
	}

	// The migrations seed a starter catalog of eggs, so List() always
	// contains more than just what this test created — check membership
	// rather than an exact count.
	list, err := eggs.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	found := false
	for _, e := range list {
		if e.ID == egg.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected List() to include the created egg %s", egg.ID)
	}

	if err := eggs.Delete(egg.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := eggs.GetByID(egg.ID); err != ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestEggsUpdateUnknownIDFails(t *testing.T) {
	db := newTestDB(t)
	eggs := NewEggs(db)

	err := eggs.Update(&models.Egg{ID: uuid.NewString(), Name: "ghost", DockerImage: "x"})
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound updating an unknown egg, got %v", err)
	}
}
