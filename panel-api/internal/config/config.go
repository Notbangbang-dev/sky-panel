package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Known insecure development defaults. A production boot must not run on these.
const (
	defaultAccessSecret  = "dev-access-secret-change-me"
	defaultRefreshSecret = "dev-refresh-secret-change-me"
	minSecretLen         = 32
)

// Config holds panel-api runtime configuration, sourced from environment
// variables so the same binary works unconfigured in dev and via systemd
// EnvironmentFile on a VPS.
type Config struct {
	HTTPAddr         string
	DBPath           string
	JWTAccessSecret  string
	JWTRefreshSecret string
	AccessTTL        time.Duration
	RefreshTTL       time.Duration

	// DevMode relaxes production safety checks (e.g. allows the built-in
	// default JWT secrets). Enabled with SKY_DEV_MODE=1/true.
	DevMode bool
	// CORSOrigin, when set (SKY_CORS_ORIGIN), pins Access-Control-Allow-Origin
	// to that exact origin instead of "*".
	CORSOrigin string
}

func Load() Config {
	return Config{
		HTTPAddr:         getEnv("SKY_HTTP_ADDR", ":8080"),
		DBPath:           getEnv("SKY_DB_PATH", "sky-panel.db"),
		JWTAccessSecret:  getEnv("SKY_JWT_ACCESS_SECRET", defaultAccessSecret),
		JWTRefreshSecret: getEnv("SKY_JWT_REFRESH_SECRET", defaultRefreshSecret),
		AccessTTL:        getEnvDuration("SKY_ACCESS_TTL", 15*time.Minute),
		RefreshTTL:       getEnvDuration("SKY_REFRESH_TTL", 30*24*time.Hour),
		DevMode:          getEnvBool("SKY_DEV_MODE", false),
		CORSOrigin:       strings.TrimSpace(getEnv("SKY_CORS_ORIGIN", "")),
	}
}

// Validate fails a production boot when the JWT secrets are missing, left at
// their public source-controlled defaults, or too short to be safe — closing
// the "admin JWT forgeable with a well-known secret" deployment footgun. In dev
// mode (SKY_DEV_MODE=1) these checks are relaxed with a warning left to the
// caller. It returns a descriptive error listing every problem found.
func (c Config) Validate() error {
	if c.DevMode {
		return nil
	}

	var problems []string
	check := func(name, val, def string) {
		switch {
		case val == "" || val == def:
			problems = append(problems, fmt.Sprintf("%s is unset or using the insecure built-in default — set it to a long random secret", name))
		case len(val) < minSecretLen:
			problems = append(problems, fmt.Sprintf("%s is too short (%d chars); use at least %d random characters", name, len(val), minSecretLen))
		}
	}
	check("SKY_JWT_ACCESS_SECRET", c.JWTAccessSecret, defaultAccessSecret)
	check("SKY_JWT_REFRESH_SECRET", c.JWTRefreshSecret, defaultRefreshSecret)

	if len(problems) > 0 {
		return fmt.Errorf("insecure configuration:\n  - %s\n\nGenerate secrets with e.g. `openssl rand -hex 32`, or set SKY_DEV_MODE=1 for local development only", strings.Join(problems, "\n  - "))
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}
