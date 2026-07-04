package httpapi

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/auth"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/wshub"
)

func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(corsMiddleware)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", d.Register)
			r.Post("/login", d.Login)
			r.Post("/refresh", d.Refresh)
			r.Post("/logout", d.Logout)
		})

		r.Route("/public", func(r chi.Router) {
			r.Get("/registration-status", d.RegistrationStatus)
			r.Get("/appearance", d.PublicAppearance)
			r.Get("/maintenance", d.MaintenanceStatus)
		})

		r.Group(func(r chi.Router) {
			r.Use(auth.RequireAuth(d.JWT, d.resolveAPIKey))
			r.Use(d.maintenanceGate)

			r.Get("/me", d.Me)
			r.Get("/me/quota", d.MyQuota)
			r.Post("/me/password", d.ChangePassword)
			r.Get("/me/sessions", d.ListSessions)
			r.Post("/me/sessions/revoke-others", d.RevokeOtherSessions)
			r.Delete("/me/sessions/{sessionID}", d.RevokeSession)
			r.Get("/me/api-keys", d.ListAPIKeys)
			r.Post("/me/api-keys", d.CreateAPIKey)
			r.Delete("/me/api-keys/{keyID}", d.DeleteAPIKey)
			r.Post("/me/totp/setup", d.TOTPSetup)
			r.Post("/me/totp/confirm", d.TOTPConfirm)
			r.Post("/me/totp/disable", d.TOTPDisable)

			r.Get("/leaderboard", d.Leaderboard)
			r.Get("/achievements", d.Achievements)
			r.Get("/me/favorites", d.ListFavorites)

			r.Get("/modrinth/search", d.ModrinthSearch)
			r.Get("/modrinth/versions", d.ModrinthVersions)

			r.Route("/servers", func(r chi.Router) {
				r.Get("/", d.ListServers)
				r.Post("/", d.CreateServer)
				r.Get("/{serverID}", d.GetServer)
				r.Patch("/{serverID}", d.UpdateServer)
				r.Delete("/{serverID}", d.DeleteServer)
				r.Post("/{serverID}/power", d.PowerAction)
				r.Post("/{serverID}/reinstall", d.ReinstallServer)
				r.Post("/{serverID}/clone", d.CloneServer)
				r.Post("/{serverID}/favorite", d.FavoriteServer)
				r.Delete("/{serverID}/favorite", d.UnfavoriteServer)
				r.Put("/{serverID}/description", d.SetServerDescription)
				r.Post("/{serverID}/console", d.ConsoleInput)
				r.Get("/{serverID}/activity", d.ServerActivity)

				r.Get("/{serverID}/backups", d.ListBackups)
				r.Post("/{serverID}/backups", d.CreateBackup)
				r.Post("/{serverID}/backups/restore", d.RestoreBackup)
				r.Delete("/{serverID}/backups", d.DeleteBackup)

				r.Get("/{serverID}/schedules", d.ListSchedules)
				r.Post("/{serverID}/schedules", d.CreateSchedule)
				r.Post("/{serverID}/schedules/{scheduleID}/toggle", d.ToggleSchedule)
				r.Delete("/{serverID}/schedules/{scheduleID}", d.DeleteSchedule)

				r.Get("/{serverID}/subusers", d.ListSubusers)
				r.Post("/{serverID}/subusers", d.AddSubuser)
				r.Delete("/{serverID}/subusers/{userID}", d.RemoveSubuser)

				r.Get("/{serverID}/files", d.ListFiles)
				r.Get("/{serverID}/files/content", d.ReadFile)
				r.Put("/{serverID}/files/content", d.WriteFile)
				r.Post("/{serverID}/modrinth/install", d.ModrinthInstall)
				r.Post("/{serverID}/files/rename", d.RenameFile)
				r.Delete("/{serverID}/files", d.DeleteFile)
				r.Post("/{serverID}/files/mkdir", d.Mkdir)
			})

			r.Get("/eggs", d.ListEggs)
			r.Get("/eggs/{eggID}", d.GetEgg)
			r.Get("/nodes", d.ListNodesSlim)

			r.Get("/wallet", d.Wallet)
			r.Post("/afk/heartbeat", d.AFKHeartbeat)
			r.Post("/daily-reward/claim", d.ClaimDailyReward)

			r.Get("/store", d.ListStore)
			r.Post("/store/purchase", d.PurchaseStoreItem)

			r.Post("/coins/gift", d.GiftCoins)
			r.Post("/coins/redeem", d.RedeemCode)

			r.Group(func(r chi.Router) {
				r.Use(auth.RequireRole(string(models.RoleAdmin)))

				r.Route("/admin/users", func(r chi.Router) {
					r.Get("/", d.AdminListUsers)
					r.Post("/{userID}/coins/adjust", d.AdminAdjustCoins)
					r.Get("/{userID}/quota", d.AdminGetUserQuota)
					r.Put("/{userID}/quota", d.AdminSetUserQuota)
					r.Post("/{userID}/role", d.AdminSetUserRole)
					r.Post("/{userID}/impersonate", d.AdminImpersonate)
					r.Delete("/{userID}", d.AdminDeleteUser)
				})

				r.Route("/admin/servers", func(r chi.Router) {
					r.Get("/", d.AdminListServers)
					r.Post("/{serverID}/suspend", d.AdminSuspendServer)
					r.Post("/{serverID}/unsuspend", d.AdminUnsuspendServer)
					r.Post("/{serverID}/owner", d.AdminTransferServer)
				})

				r.Route("/admin/redeem-codes", func(r chi.Router) {
					r.Get("/", d.AdminListRedeemCodes)
					r.Post("/", d.AdminCreateRedeemCode)
					r.Delete("/{codeID}", d.AdminDeleteRedeemCode)
				})

				r.Route("/admin/nodes", func(r chi.Router) {
					r.Get("/", d.ListNodes)
					r.Post("/", d.CreateNode)
					r.Delete("/{nodeID}", d.DeleteNode)
					r.Post("/{nodeID}/rotate-token", d.RotateNodeToken)
					r.Get("/{nodeID}/allocations", d.AdminListAllocations)
					r.Post("/{nodeID}/allocations", d.AdminCreateAllocations)
				})
				r.Delete("/admin/allocations/{allocationID}", d.AdminDeleteAllocation)
				r.Route("/admin/eggs", func(r chi.Router) {
					r.Post("/", d.CreateEgg)
					r.Put("/{eggID}", d.UpdateEgg)
					r.Get("/{eggID}/export", d.ExportEgg)
					r.Delete("/{eggID}", d.DeleteEgg)
				})

				r.Route("/admin/settings", func(r chi.Router) {
					r.Get("/", d.AdminGetSettings)
					r.Put("/{key}", d.AdminSetSetting)
				})

				r.Get("/admin/audit-log", d.AdminListAuditLog)
				r.Post("/admin/broadcast", d.AdminBroadcast)
			})
		})
	})

	r.Group(func(r chi.Router) {
		r.Use(auth.RequireAuth(d.JWT, d.resolveAPIKey))
		r.Get("/ws", d.ServeWS)
	})

	// Nodes dial this in from arbitrary VPS IPs, authenticating via the
	// hello message's node token rather than a user JWT (see agenthub).
	r.Get("/agent/ws", d.AgentHub.ServeWS)

	return r
}

// ServeWS subscribes the caller to the topics given in the ?topics=a,b,c
// query parameter. Any "server:<id>:*" topic is only granted if the caller
// owns that server (or is an admin); other topics are passed through
// unchecked for now.
func (d Deps) ServeWS(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	raw := r.URL.Query().Get("topics")
	var topics []string
	for _, t := range strings.Split(raw, ",") {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if !d.authorizedForTopic(claims, t) {
			continue
		}
		topics = append(topics, t)
	}

	if err := wshub.Upgrade(d.Hub, w, r, topics); err != nil {
		writeError(w, http.StatusBadRequest, "ws_upgrade_failed", err.Error())
	}
}

func (d Deps) authorizedForTopic(claims *auth.Claims, topic string) bool {
	parts := strings.Split(topic, ":")
	if len(parts) != 3 || parts[0] != "server" {
		return true
	}

	if claims.Role == string(models.RoleAdmin) {
		return true
	}

	server, err := d.Servers.GetByID(parts[1])
	if err != nil {
		return false
	}
	if server.OwnerID == claims.UserID {
		return true
	}
	// A subuser granted access to this server may also watch its live topics —
	// otherwise they can control it over HTTP but never see real-time updates.
	if _, err := d.Subusers.Get(parts[1], claims.UserID); err == nil {
		return true
	}
	return false
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
