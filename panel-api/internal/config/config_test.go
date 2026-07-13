package config

import "testing"

func TestValidateRejectsDefaultSecrets(t *testing.T) {
	c := Config{
		JWTAccessSecret:  defaultAccessSecret,
		JWTRefreshSecret: defaultRefreshSecret,
	}
	if err := c.Validate(); err == nil {
		t.Fatal("expected Validate to reject the built-in default secrets")
	}
}

func TestValidateRejectsShortSecrets(t *testing.T) {
	c := Config{
		JWTAccessSecret:  "short",
		JWTRefreshSecret: "alsoshort",
	}
	if err := c.Validate(); err == nil {
		t.Fatal("expected Validate to reject too-short secrets")
	}
}

func TestValidateAcceptsStrongSecrets(t *testing.T) {
	strong := "a3f9c1e7b2d84056a1c93e7f0b6d2481aa11bb22cc33dd44"
	c := Config{
		JWTAccessSecret:  strong,
		JWTRefreshSecret: strong + "x",
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("expected strong secrets to validate, got: %v", err)
	}
}

func TestValidateDevModeBypass(t *testing.T) {
	c := Config{
		DevMode:          true,
		JWTAccessSecret:  defaultAccessSecret,
		JWTRefreshSecret: defaultRefreshSecret,
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("dev mode should bypass secret checks, got: %v", err)
	}
}
