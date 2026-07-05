package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/agenthub"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/auth"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/backupsvc"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/coinsvc"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/config"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/httpapi"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/quotasvc"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/repo"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/schedulesvc"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/serversvc"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/store"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/storesvc"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/wshub"
)

func main() {
	cfg := config.Load()

	db, err := store.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer db.Close()

	hub := wshub.NewHub()
	users := repo.NewUsers(db)
	nodes := repo.NewNodes(db)
	eggs := repo.NewEggs(db)
	servers := repo.NewServers(db)
	allocations := repo.NewAllocations(db)
	subusers := repo.NewSubusers(db)
	ledger := repo.NewLedger(db)
	afk := repo.NewAFKState(db)
	dailyRewards := repo.NewDailyRewards(db)
	quotas := repo.NewQuotas(db)
	apiKeys := repo.NewAPIKeys(db)
	schedules := repo.NewSchedules(db)
	redeemCodes := repo.NewRedeemCodes(db)
	favorites := repo.NewFavorites(db)
	settings := repo.NewSettings(db)
	auditLog := repo.NewAudit(db)
	databases := repo.NewDatabases(db)

	agentRegistry := agenthub.NewRegistry()
	agentHandler := agenthub.NewHandler(agentRegistry, nodes, hub)
	// Only accept events/heartbeats from the node that actually hosts the
	// referenced server, so one compromised node can't corrupt another's
	// roster/stats/console stream.
	agentHandler.UseServerLocator(servers)
	serverSvc := serversvc.NewService(servers, eggs, nodes, allocations, agentRegistry)
	serverSvc.Databases = databases
	// Warm each node's Docker image cache as soon as it connects, so the first
	// server create on it is fast instead of blocking on a cold image pull.
	agentHandler.OnNodeConnected = serverSvc.WarmImagesOnNode
	// Drop a deleted server's tracked roster immediately.
	serverSvc.OnServerDeleted = agentHandler.Forget

	quotaSvc := quotasvc.NewService(servers, quotas, settings)
	quotaSvc.Databases = databases

	deps := httpapi.Deps{
		Users:         users,
		RefreshTokens: repo.NewRefreshTokens(db),
		JWT:           auth.NewManager(cfg.JWTAccessSecret, cfg.AccessTTL),
		Hub:           hub,
		APIKeys:       apiKeys,
		Nodes:         nodes,
		Eggs:          eggs,
		Servers:       servers,
		Allocations:   allocations,
		Subusers:      subusers,
		Quotas:        quotas,
		Schedules:     schedules,
		RedeemCodes:   redeemCodes,
		Favorites:     favorites,
		Databases:     databases,
		ServerSvc:     serverSvc,
		AgentHub:      agentHandler,
		CoinSvc:       coinsvc.NewService(users, ledger, afk, dailyRewards, settings),
		QuotaSvc:      quotaSvc,
		StoreSvc:      storesvc.NewService(ledger, quotas),
		Settings:      settings,
		Audit:         auditLog,
		RefreshTTL:    cfg.RefreshTTL,
	}

	// Background loop that runs due scheduled backups.
	go backupsvc.NewScheduler(servers, serverSvc, 15*time.Minute).Run(context.Background())
	// Background loop that runs due per-server automations (schedules).
	go schedulesvc.NewScheduler(schedules, serverSvc, time.Minute).Run(context.Background())

	log.Printf("sky-panel panel-api listening on %s", cfg.HTTPAddr)
	if err := http.ListenAndServe(cfg.HTTPAddr, httpapi.NewRouter(deps)); err != nil {
		log.Fatalf("http server: %v", err)
	}
}
