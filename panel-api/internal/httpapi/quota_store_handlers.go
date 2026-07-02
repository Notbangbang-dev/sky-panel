package httpapi

import (
	"errors"
	"net/http"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/auth"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/quotasvc"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/repo"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/storesvc"
)

// writeQuotaError renders a quotasvc.Error as a 409 with the offending
// dimension, or a generic 500 for any other error.
func (d Deps) writeQuotaError(w http.ResponseWriter, err error) {
	var qe *quotasvc.Error
	if errors.As(err, &qe) {
		writeError(w, http.StatusConflict, "quota_exceeded", qe.Error())
		return
	}
	writeError(w, http.StatusInternalServerError, "internal_error", "failed to check quota")
}

type quotaResponse struct {
	Usage quotasvc.Usage `json:"usage"`
	Limit quotasvc.Quota `json:"limit"`
}

// MyQuota reports the caller's current usage and effective limits, for the
// quota meters shown throughout the UI.
func (d Deps) MyQuota(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}
	d.writeQuotaFor(w, claims.UserID)
}

func (d Deps) writeQuotaFor(w http.ResponseWriter, userID string) {
	usage, err := d.QuotaSvc.Usage(userID, "")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to compute usage")
		return
	}
	limit, err := d.QuotaSvc.Effective(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load quota")
		return
	}
	writeJSON(w, http.StatusOK, quotaResponse{Usage: usage, Limit: limit})
}

// ListStore returns the store catalog.
func (d Deps) ListStore(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, storesvc.Catalog)
}

type purchaseRequest struct {
	ItemID string `json:"item_id"`
}

type purchaseResponse struct {
	ItemID  string `json:"item_id"`
	Balance int64  `json:"balance"`
}

// PurchaseStoreItem debits coins and grants the item's quota upgrade.
func (d Deps) PurchaseStoreItem(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	var req purchaseRequest
	if err := decodeJSON(r, &req); err != nil || req.ItemID == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "item_id is required")
		return
	}

	item, balance, err := d.StoreSvc.Purchase(claims.UserID, req.ItemID)
	if errors.Is(err, storesvc.ErrItemNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "no such store item")
		return
	}
	if errors.Is(err, repo.ErrInsufficientBalance) {
		writeError(w, http.StatusConflict, "insufficient_balance", "you don't have enough coins for this")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "purchase failed")
		return
	}
	d.audit(r, "store.purchase", claims.UserID, item.ID)
	writeJSON(w, http.StatusOK, purchaseResponse{ItemID: item.ID, Balance: balance})
}

type adminQuotaRequest struct {
	MemoryBytes int64 `json:"memory_bytes"`
	CPUPercent  int   `json:"cpu_percent"`
	DiskBytes   int64 `json:"disk_bytes"`
}

// AdminSetUserQuota sets a user's bonus quota (the amount added on top of the
// global defaults) to absolute values.
func (d Deps) AdminSetUserQuota(w http.ResponseWriter, r *http.Request) {
	userID := pathParam(r, "userID")
	if _, err := d.Users.GetByID(userID); errors.Is(err, repo.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "user not found")
		return
	}

	var req adminQuotaRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	if err := d.Quotas.Set(userID, req.MemoryBytes, req.CPUPercent, req.DiskBytes); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to set quota")
		return
	}
	d.audit(r, "user.quota.set", userID, "")
	d.writeQuotaFor(w, userID)
}
