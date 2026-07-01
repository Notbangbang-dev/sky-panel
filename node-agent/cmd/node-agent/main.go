package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/Notbangbang-dev/sky-panel/node-agent/internal/agentclient"
	"github.com/Notbangbang-dev/sky-panel/node-agent/internal/config"
	"github.com/Notbangbang-dev/sky-panel/node-agent/internal/runtime"
)

func main() {
	cfg := config.Load()
	if cfg.NodeToken == "" {
		log.Fatal("SKY_NODE_TOKEN is required")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dockerRuntime := runtime.NewDocker(cfg.DockerSocket)
	dispatch := agentclient.NewDispatcher(dockerRuntime)

	log.Printf("sky-panel node-agent %s connecting to %s", agentclient.AgentVersion, cfg.PanelWSURL)
	agentclient.Run(ctx, cfg.PanelWSURL, cfg.NodeToken, dispatch, cfg.HeartbeatInterval)
}
