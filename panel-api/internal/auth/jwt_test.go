package auth

import (
	"testing"
	"time"
)

func TestAccessTokenRoundTrip(t *testing.T) {
	mgr := NewManager("test-secret", time.Minute)

	token, err := mgr.NewAccessToken("user-1", "admin")
	if err != nil {
		t.Fatalf("NewAccessToken: %v", err)
	}

	claims, err := mgr.ParseAccessToken(token)
	if err != nil {
		t.Fatalf("ParseAccessToken: %v", err)
	}
	if claims.UserID != "user-1" || claims.Role != "admin" {
		t.Errorf("unexpected claims: %+v", claims)
	}
}

func TestAccessTokenExpired(t *testing.T) {
	mgr := NewManager("test-secret", -time.Minute)

	token, err := mgr.NewAccessToken("user-1", "user")
	if err != nil {
		t.Fatalf("NewAccessToken: %v", err)
	}

	if _, err := mgr.ParseAccessToken(token); err == nil {
		t.Error("expected expired token to fail parsing")
	}
}

func TestAccessTokenWrongSecret(t *testing.T) {
	mgr := NewManager("test-secret", time.Minute)
	other := NewManager("other-secret", time.Minute)

	token, err := mgr.NewAccessToken("user-1", "user")
	if err != nil {
		t.Fatalf("NewAccessToken: %v", err)
	}

	if _, err := other.ParseAccessToken(token); err == nil {
		t.Error("expected token signed with a different secret to fail parsing")
	}
}
