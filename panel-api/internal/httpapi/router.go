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
		})

		r.Group(func(r chi.Router) {
			r.Use(auth.RequireAuth(d.JWT))

			r.Get("/me", d.Me)
			r.Post("/me/totp/setup", d.TOTPSetup)
			r.Post("/me/totp/confirm", d.TOTPConfirm)
			r.Post("/me/totp/disable", d.TOTPDisable)

			r.Route("/servers", func(r chi.Router) {
				r.Get("/", d.ListServers)
				r.Post("/", d.CreateServer)
				r.Get("/{serverID}", d.GetServer)
				r.Delete("/{serverID}", d.DeleteServer)
				r.Post("/{serverID}/power", d.PowerAction)
				r.Post("/{serverID}/console", d.ConsoleInput)

				r.Get("/{serverID}/subusers", d.ListSubusers)
				r.Post("/{serverID}/subusers", d.AddSubuser)
				r.Delete("/{serverID}/subusers/{userID}", d.RemoveSubuser)

				r.Get("/{serverID}/files", d.ListFiles)
				r.Get("/{serverID}/files/content", d.ReadFile)
				r.Put("/{serverID}/files/content", d.WriteFile)
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

			r.Group(func(r chi.Router) {
				r.Use(auth.RequireRole(string(models.RoleAdmin)))

				r.Route("/admin/users", func(r chi.Router) {
					r.Get("/", d.AdminListUsers)
					r.Post("/{userID}/coins/adjust", d.AdminAdjustCoins)
					r.Post("/{userID}/role", d.AdminSetUserRole)
					r.Delete("/{userID}", d.AdminDeleteUser)
				})

				r.Route("/admin/nodes", func(r chi.Router) {
					r.Get("/", d.ListNodes)
					r.Post("/", d.CreateNode)
					r.Delete("/{nodeID}", d.DeleteNode)
					r.Post("/{nodeID}/rotate-token", d.RotateNodeToken)
				})
				r.Route("/admin/eggs", func(r chi.Router) {
					r.Post("/", d.CreateEgg)
					r.Put("/{eggID}", d.UpdateEgg)
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
		r.Use(auth.RequireAuth(d.JWT))
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
	return server.OwnerID == claims.UserID
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
