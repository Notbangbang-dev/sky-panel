package httpapi

import (
	"errors"
	"net/http"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/auth"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/coinsvc"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/repo"
)

type coinResultResponse struct {
	Credited int64 `json:"credited"`
	Balance  int64 `json:"balance"`
}

func (d Deps) AFKHeartbeat(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	credited, balance, err := d.CoinSvc.Heartbeat(claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to record heartbeat")
		return
	}
	writeJSON(w, http.StatusOK, coinResultResponse{Credited: credited, Balance: balance})
}

func (d Deps) ClaimDailyReward(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	credited, balance, err := d.CoinSvc.ClaimDailyReward(claims.UserID)
	if errors.Is(err, coinsvc.ErrDailyRewardAlreadyClaimed) {
		writeError(w, http.StatusTooManyRequests, "already_claimed", "daily reward already claimed; try again later")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to claim daily reward")
		return
	}
	writeJSON(w, http.StatusOK, coinResultResponse{Credited: credited, Balance: balance})
}

type walletResponse struct {
	Balance int64            `json:"balance"`
	History []ledgerResponse `json:"history"`
}

type ledgerResponse struct {
	Amount    int64  `json:"amount"`
	Reason    string `json:"reason"`
	Metadata  string `json:"metadata,omitempty"`
	CreatedAt string `json:"created_at"`
}

func (d Deps) Wallet(w http.ResponseWriter, r *http.Request) {
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

	entries, err := d.CoinSvc.Ledger.ListByUser(claims.UserID, 50)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load ledger history")
		return
	}

	history := make([]ledgerResponse, 0, len(entries))
	for _, e := range entries {
		history = append(history, ledgerResponse{
			Amount: e.Amount, Reason: e.Reason, Metadata: e.Metadata, CreatedAt: e.CreatedAt.Format(rfc3339),
		})
	}

	writeJSON(w, http.StatusOK, walletResponse{Balance: user.Coins, History: history})
}

const rfc3339 = "2006-01-02T15:04:05Z07:00"

type adminAdjustCoinsRequest struct {
	Amount int64  `json:"amount"`
	Note   string `json:"note,omitempty"`
}

func (d Deps) AdminAdjustCoins(w http.ResponseWriter, r *http.Request) {
	var req adminAdjustCoinsRequest
	if err := decodeJSON(r, &req); err != nil || req.Amount == 0 {
		writeError(w, http.StatusBadRequest, "bad_request", "a non-zero amount is required")
		return
	}

	targetUserID := pathParam(r, "userID")
	balance, err := d.CoinSvc.AdminAdjust(targetUserID, req.Amount, req.Note)
	if errors.Is(err, repo.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "user not found")
		return
	}
	if errors.Is(err, repo.ErrInsufficientBalance) {
		writeError(w, http.StatusConflict, "insufficient_balance", "adjustment would make the balance negative")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to adjust balance")
		return
	}
	d.audit(r, "coins.adjust", targetUserID, req.Note)
	writeJSON(w, http.StatusOK, coinResultResponse{Credited: req.Amount, Balance: balance})
}
