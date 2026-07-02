package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/auth"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/coinsvc"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/repo"
)

// maxCoinGrant caps any single coin grant (gift, redeem code) well below the
// int64 ceiling so balances can never be driven to overflow.
const maxCoinGrant = 1_000_000_000

type giftRequest struct {
	Username string `json:"username"`
	Amount   int64  `json:"amount"`
}

// GiftCoins transfers coins from the caller to another user (by username).
func (d Deps) GiftCoins(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	var req giftRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Amount <= 0 || req.Amount > maxCoinGrant {
		writeError(w, http.StatusBadRequest, "bad_request", "a recipient username and an amount between 1 and 1,000,000,000 are required")
		return
	}

	balance, err := d.CoinSvc.Gift(claims.UserID, req.Username, req.Amount)
	switch {
	case errors.Is(err, repo.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "no user with that username")
		return
	case errors.Is(err, coinsvc.ErrGiftToSelf):
		writeError(w, http.StatusBadRequest, "bad_request", "you can't gift coins to yourself")
		return
	case errors.Is(err, repo.ErrInsufficientBalance):
		writeError(w, http.StatusBadRequest, "insufficient_balance", "you don't have enough coins")
		return
	case err != nil:
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to send coins")
		return
	}

	d.audit(r, "coins.gift", req.Username, "")
	writeJSON(w, http.StatusOK, coinResultResponse{Credited: -req.Amount, Balance: balance})
}

type redeemRequest struct {
	Code string `json:"code"`
}

// RedeemCode applies a redeem code to the caller's balance.
func (d Deps) RedeemCode(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	var req redeemRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	req.Code = strings.TrimSpace(req.Code)
	if req.Code == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "a code is required")
		return
	}

	credited, balance, err := d.RedeemCodes.Redeem(req.Code, claims.UserID)
	switch {
	case errors.Is(err, repo.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "that code doesn't exist")
		return
	case errors.Is(err, repo.ErrCodeExhausted):
		writeError(w, http.StatusBadRequest, "code_exhausted", "that code has been fully redeemed")
		return
	case errors.Is(err, repo.ErrCodeAlreadyRedeemed):
		writeError(w, http.StatusBadRequest, "code_already_used", "you've already redeemed that code")
		return
	case err != nil:
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to redeem code")
		return
	}

	d.audit(r, "coins.redeem", req.Code, "")
	writeJSON(w, http.StatusOK, coinResultResponse{Credited: credited, Balance: balance})
}
