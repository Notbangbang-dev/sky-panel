package httpapi

import "net/http"

type leaderboardEntry struct {
	Rank     int    `json:"rank"`
	Username string `json:"username"`
	Coins    int64  `json:"coins"`
}

// Leaderboard returns the top coin balances.
func (d Deps) Leaderboard(w http.ResponseWriter, r *http.Request) {
	users, err := d.Users.TopByCoins(25)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load leaderboard")
		return
	}
	out := make([]leaderboardEntry, 0, len(users))
	for i, u := range users {
		out = append(out, leaderboardEntry{Rank: i + 1, Username: u.Username, Coins: u.Coins})
	}
	writeJSON(w, http.StatusOK, out)
}
