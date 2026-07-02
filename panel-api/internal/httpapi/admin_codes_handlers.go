package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/repo"
)

type createCodeRequest struct {
	Code    string `json:"code"`
	Coins   int64  `json:"coins"`
	MaxUses int    `json:"max_uses"`
}

// AdminCreateRedeemCode mints a redeem code.
func (d Deps) AdminCreateRedeemCode(w http.ResponseWriter, r *http.Request) {
	var req createCodeRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	req.Code = strings.TrimSpace(req.Code)
	if req.Code == "" || req.Coins <= 0 || req.Coins > maxCoinGrant || req.MaxUses < 0 {
		writeError(w, http.StatusBadRequest, "bad_request", "code, a coin amount between 1 and 1,000,000,000, and a non-negative max_uses are required")
		return
	}

	code, err := d.RedeemCodes.Create(req.Code, req.Coins, req.MaxUses)
	if errors.Is(err, repo.ErrDuplicate) {
		writeError(w, http.StatusConflict, "already_exists", "a code with that value already exists")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to create code")
		return
	}
	d.audit(r, "code.create", code.Code, "")
	writeJSON(w, http.StatusCreated, code)
}

// AdminListRedeemCodes lists all redeem codes with their usage counts.
func (d Deps) AdminListRedeemCodes(w http.ResponseWriter, r *http.Request) {
	codes, err := d.RedeemCodes.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list codes")
		return
	}
	if codes == nil {
		codes = []*models.RedeemCode{}
	}
	writeJSON(w, http.StatusOK, codes)
}

// AdminDeleteRedeemCode removes a redeem code (past redemptions are unaffected).
func (d Deps) AdminDeleteRedeemCode(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "codeID")
	if err := d.RedeemCodes.Delete(id); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "code not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete code")
		return
	}
	d.audit(r, "code.delete", id, "")
	w.WriteHeader(http.StatusNoContent)
}
