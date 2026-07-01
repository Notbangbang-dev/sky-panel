package repo

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
)

func newTestUser() *models.User {
	now := time.Now().UTC()
	return &models.User{
		ID:           uuid.NewString(),
		Email:        "sky@example.com",
		Username:     "sky",
		PasswordHash: "hash",
		Role:         models.RoleUser,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func TestUsersCreateAndGet(t *testing.T) {
	repo := NewUsers(newTestDB(t))
	u := newTestUser()

	if err := repo.Create(u); err != nil {
		t.Fatalf("Create: %v", err)
	}

	byID, err := repo.GetByID(u.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if byID.Email != u.Email || byID.Username != u.Username {
		t.Errorf("GetByID returned unexpected user: %+v", byID)
	}

	byEmail, err := repo.GetByEmail(u.Email)
	if err != nil {
		t.Fatalf("GetByEmail: %v", err)
	}
	if byEmail.ID != u.ID {
		t.Errorf("GetByEmail returned wrong user")
	}

	byUsername, err := repo.GetByUsername(u.Username)
	if err != nil {
		t.Fatalf("GetByUsername: %v", err)
	}
	if byUsername.ID != u.ID {
		t.Errorf("GetByUsername returned wrong user")
	}
}

func TestUsersGetByIDNotFound(t *testing.T) {
	repo := NewUsers(newTestDB(t))

	if _, err := repo.GetByID("nonexistent"); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUsersCreateDuplicateEmail(t *testing.T) {
	repo := NewUsers(newTestDB(t))

	first := newTestUser()
	if err := repo.Create(first); err != nil {
		t.Fatalf("Create first: %v", err)
	}

	second := newTestUser()
	second.ID = uuid.NewString()
	second.Username = "different-username"
	// same email as first

	if err := repo.Create(second); !errors.Is(err, ErrDuplicate) {
		t.Errorf("expected ErrDuplicate for duplicate email, got %v", err)
	}
}

func TestUsersCount(t *testing.T) {
	repo := NewUsers(newTestDB(t))

	n, err := repo.Count()
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected 0 users initially, got %d", n)
	}

	if err := repo.Create(newTestUser()); err != nil {
		t.Fatalf("Create: %v", err)
	}

	n, err = repo.Count()
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 user after create, got %d", n)
	}
}

func TestUsersSetTOTP(t *testing.T) {
	repo := NewUsers(newTestDB(t))
	u := newTestUser()
	if err := repo.Create(u); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.SetTOTP(u.ID, "secret123", true); err != nil {
		t.Fatalf("SetTOTP: %v", err)
	}

	got, err := repo.GetByID(u.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.TOTPSecret != "secret123" || !got.TOTPEnabled {
		t.Errorf("SetTOTP did not persist: %+v", got)
	}
}

func TestUsersSetTOTPNotFound(t *testing.T) {
	repo := NewUsers(newTestDB(t))

	if err := repo.SetTOTP("nonexistent", "secret", true); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
