package httpapi

import (
	"net/http"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
)

type adminAnalytics struct {
	Users           int            `json:"users"`
	Admins          int            `json:"admins"`
	Servers         int            `json:"servers"`
	ServersByStatus map[string]int `json:"servers_by_status"`
	Suspended       int            `json:"suspended"`
	ServersByEgg    map[string]int `json:"servers_by_egg"`
	Nodes           int            `json:"nodes"`
	NodesConnected  int            `json:"nodes_connected"`
	Eggs            int            `json:"eggs"`
	CoinsInCirc     int64          `json:"coins_in_circulation"`
}

// AdminAnalytics returns a read-only, at-a-glance rollup of the instance for
// the admin dashboard: user/server/node counts, servers grouped by status and
// egg, and total coins in circulation. All derived live — no separate table.
func (d Deps) AdminAnalytics(w http.ResponseWriter, r *http.Request) {
	users, err := d.Users.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load users")
		return
	}
	servers, err := d.Servers.ListAll()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load servers")
		return
	}
	eggs, err := d.Eggs.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load eggs")
		return
	}
	nodes, err := d.Nodes.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load nodes")
		return
	}

	out := adminAnalytics{
		Users:           len(users),
		Servers:         len(servers),
		ServersByStatus: map[string]int{},
		ServersByEgg:    map[string]int{},
		Eggs:            len(eggs),
		Nodes:           len(nodes),
	}

	for _, u := range users {
		if u.Role == models.RoleAdmin {
			out.Admins++
		}
		out.CoinsInCirc += u.Coins
	}

	eggNames := make(map[string]string, len(eggs))
	for _, e := range eggs {
		eggNames[e.ID] = e.Name
	}
	for _, s := range servers {
		out.ServersByStatus[string(s.Status)]++
		if s.Suspended {
			out.Suspended++
		}
		name := eggNames[s.EggID]
		if name == "" {
			name = "unknown"
		}
		out.ServersByEgg[name]++
	}

	for _, n := range nodes {
		if d.AgentHub.Registry.Connected(n.ID) {
			out.NodesConnected++
		}
	}

	writeJSON(w, http.StatusOK, out)
}
