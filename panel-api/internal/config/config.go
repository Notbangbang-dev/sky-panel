package config

import (
	"os"
	"time"
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
}

func Load() Config {
	return Config{
		HTTPAddr:         getEnv("SKY_HTTP_ADDR", ":8080"),
		DBPath:           getEnv("SKY_DB_PATH", "sky-panel.db"),
		JWTAccessSecret:  getEnv("SKY_JWT_ACCESS_SECRET", "dev-access-secret-change-me"),
		JWTRefreshSecret: getEnv("SKY_JWT_REFRESH_SECRET", "dev-refresh-secret-change-me"),
		AccessTTL:        getEnvDuration("SKY_ACCESS_TTL", 15*time.Minute),
		RefreshTTL:       getEnvDuration("SKY_REFRESH_TTL", 30*24*time.Hour),
	}
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
