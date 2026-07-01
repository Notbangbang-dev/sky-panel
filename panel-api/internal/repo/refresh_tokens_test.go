package repo

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRefreshTokensCreateAndValidate(t *testing.T) {
	db := newTestDB(t)
	users := NewUsers(db)
	tokens := NewRefreshTokens(db)

	u := newTestUser()
	if err := users.Create(u); err != nil {
		t.Fatalf("Create user: %v", err)
	}

	hash := "hashed-token-value"
	if err := tokens.Create(uuid.NewString(), u.ID, hash, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("Create token: %v", err)
	}

	userID, err := tokens.UserIDForValidToken(hash)
	if err != nil {
		t.Fatalf("UserIDForValidToken: %v", err)
	}
	if userID != u.ID {
		t.Errorf("expected user ID %q, got %q", u.ID, userID)
	}
}

func TestRefreshTokensExpired(t *testing.T) {
	db := newTestDB(t)
	users := NewUsers(db)
	tokens := NewRefreshTokens(db)

	u := newTestUser()
	if err := users.Create(u); err != nil {
		t.Fatalf("Create user: %v", err)
	}

	hash := "expired-token"
	if err := tokens.Create(uuid.NewString(), u.ID, hash, time.Now().Add(-time.Hour)); err != nil {
		t.Fatalf("Create token: %v", err)
	}

	if _, err := tokens.UserIDForValidToken(hash); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound for expired token, got %v", err)
	}
}

func TestRefreshTokensDeleteByHash(t *testing.T) {
	db := newTestDB(t)
	users := NewUsers(db)
	tokens := NewRefreshTokens(db)

	u := newTestUser()
	if err := users.Create(u); err != nil {
		t.Fatalf("Create user: %v", err)
	}

	hash := "to-be-deleted"
	if err := tokens.Create(uuid.NewString(), u.ID, hash, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("Create token: %v", err)
	}
	if err := tokens.DeleteByHash(hash); err != nil {
		t.Fatalf("DeleteByHash: %v", err)
	}

	if _, err := tokens.UserIDForValidToken(hash); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}
