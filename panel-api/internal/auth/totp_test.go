package auth

import (
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

func TestVerifyTOTPCode(t *testing.T) {
	key, err := NewTOTPSecret("test@example.com")
	if err != nil {
		t.Fatalf("NewTOTPSecret: %v", err)
	}

	code, err := totp.GenerateCode(key.Secret(), time.Now())
	if err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}

	if !VerifyTOTPCode(key.Secret(), code) {
		t.Error("expected freshly generated code to validate")
	}
	if VerifyTOTPCode(key.Secret(), "000000") {
		t.Error("expected an arbitrary wrong code to be rejected (astronomically unlikely to collide)")
	}
	if VerifyTOTPCode("", code) {
		t.Error("expected empty secret to be rejected")
	}
}
