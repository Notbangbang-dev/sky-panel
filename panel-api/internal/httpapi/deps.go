package httpapi

import (
	"time"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/agenthub"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/auth"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/coinsvc"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/quotasvc"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/repo"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/serversvc"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/storesvc"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/wshub"
)

// Deps wires every dependency handlers need. It is intentionally a flat
// struct rather than an interface-heavy DI container: this is a monolith
// control plane, and handlers are simple enough to take Deps by value.
type Deps struct {
	Users         *repo.Users
	RefreshTokens *repo.RefreshTokens
	JWT           *auth.Manager
	Hub           *wshub.Hub

	APIKeys     *repo.APIKeys
	Nodes       *repo.Nodes
	Eggs        *repo.Eggs
	Servers     *repo.Servers
	Allocations *repo.Allocations
	Subusers    *repo.Subusers
	Quotas      *repo.Quotas
	Schedules   *repo.Schedules
	ServerSvc   *serversvc.Service
	AgentHub    *agenthub.Handler
	CoinSvc     *coinsvc.Service
	QuotaSvc    *quotasvc.Service
	StoreSvc    *storesvc.Service
	Settings    *repo.Settings
	Audit       *repo.Audit

	RefreshTTL time.Duration
}
