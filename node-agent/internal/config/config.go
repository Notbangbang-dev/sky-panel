package config

import (
	"os"
	"time"
)

type Config struct {
	PanelWSURL        string
	NodeToken         string
	DockerSocket      string
	HeartbeatInterval time.Duration
}

func Load() Config {
	return Config{
		PanelWSURL:        getEnv("SKY_PANEL_WS_URL", "ws://127.0.0.1:8080/agent/ws"),
		NodeToken:         os.Getenv("SKY_NODE_TOKEN"),
		DockerSocket:      getEnv("SKY_DOCKER_SOCKET", "/var/run/docker.sock"),
		HeartbeatInterval: getEnvDuration("SKY_HEARTBEAT_INTERVAL", 5*time.Second),
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
