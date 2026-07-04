package httpapi

import (
	"net/http"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/auth"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
)

type achievement struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Unlocked    bool   `json:"unlocked"`
}

// Achievements derives the caller's unlocked milestones from existing data, so
// there's no separate award pipeline to keep in sync — the source of truth is
// always the live state (servers owned, coins, 2FA, gifting, redemptions).
func (d Deps) Achievements(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	user, err := d.Users.GetByID(claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load user")
		return
	}
	servers, err := d.Servers.ListByOwner(claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load servers")
		return
	}
	gifted, _ := d.CoinSvc.Ledger.HasReason(claims.UserID, models.ReasonGiftSent)
	redeemed, _ := d.RedeemCodes.HasRedeemed(claims.UserID)
	n := len(servers)

	list := []achievement{
		{ID: "first_server", Name: "First deploy", Description: "Create your first server", Unlocked: n >= 1},
		{ID: "fleet", Name: "Fleet commander", Description: "Own three or more servers at once", Unlocked: n >= 3},
		{ID: "rich", Name: "Coin hoarder", Description: "Reach a balance of 1,000 coins", Unlocked: user.Coins >= 1000},
		{ID: "generous", Name: "Generous", Description: "Gift coins to another player", Unlocked: gifted},
		{ID: "lucky", Name: "Code redeemer", Description: "Redeem a code", Unlocked: redeemed},
		{ID: "secured", Name: "Locked down", Description: "Turn on two-factor authentication", Unlocked: user.TOTPEnabled},
	}
	writeJSON(w, http.StatusOK, list)
}
