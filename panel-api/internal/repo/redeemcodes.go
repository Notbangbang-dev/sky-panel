package repo

import (
	"database/sql"
	"errors"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
)

var (
	// ErrCodeExhausted is returned when a code has hit its max_uses.
	ErrCodeExhausted = errors.New("redeem code has no uses left")
	// ErrCodeAlreadyRedeemed is returned when the user already redeemed it.
	ErrCodeAlreadyRedeemed = errors.New("redeem code already used by this user")
)

type RedeemCodes struct {
	db *sql.DB
}

func NewRedeemCodes(db *sql.DB) *RedeemCodes {
	return &RedeemCodes{db: db}
}

// Create mints a new code. Returns ErrDuplicate if the code already exists.
func (r *RedeemCodes) Create(code string, coins int64, maxUses int) (*models.RedeemCode, error) {
	rc := &models.RedeemCode{
		ID:        uuid.NewString(),
		Code:      code,
		Coins:     coins,
		MaxUses:   maxUses,
		CreatedAt: time.Now().UTC(),
	}
	_, err := r.db.Exec(
		`INSERT INTO redeem_codes (id, code, coins, max_uses, uses, created_at) VALUES (?, ?, ?, ?, 0, ?)`,
		rc.ID, rc.Code, rc.Coins, rc.MaxUses, rc.CreatedAt,
	)
	if isUniqueViolation(err) {
		return nil, ErrDuplicate
	}
	if err != nil {
		return nil, err
	}
	return rc, nil
}

func (r *RedeemCodes) List() ([]*models.RedeemCode, error) {
	rows, err := r.db.Query(`SELECT id, code, coins, max_uses, uses, created_at FROM redeem_codes ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*models.RedeemCode
	for rows.Next() {
		var c models.RedeemCode
		if err := rows.Scan(&c.ID, &c.Code, &c.Coins, &c.MaxUses, &c.Uses, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

// HasRedeemed reports whether a user has redeemed any code (for achievements).
func (r *RedeemCodes) HasRedeemed(userID string) (bool, error) {
	var one int
	err := r.db.QueryRow(`SELECT 1 FROM redeem_code_redemptions WHERE user_id = ? LIMIT 1`, userID).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *RedeemCodes) Delete(id string) error {
	res, err := r.db.Exec(`DELETE FROM redeem_codes WHERE id = ?`, id)
	return checkRowsAffected(res, err)
}

// Redeem atomically applies a code for a user: it checks the code exists, has
// uses left, and hasn't already been redeemed by this user; then records the
// redemption, bumps the use count, and credits the user's coins — all in one
// transaction so concurrent redemptions can't double-spend or exceed max_uses.
// Returns the coins granted and the user's new balance.
func (r *RedeemCodes) Redeem(code, userID string) (coins, newBalance int64, err error) {
	tx, err := r.db.Begin()
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()

	var codeID string
	var maxUses, uses int
	err = tx.QueryRow(`SELECT id, coins, max_uses, uses FROM redeem_codes WHERE code = ?`, code).
		Scan(&codeID, &coins, &maxUses, &uses)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, ErrNotFound
	}
	if err != nil {
		return 0, 0, err
	}
	if maxUses > 0 && uses >= maxUses {
		return 0, 0, ErrCodeExhausted
	}

	now := time.Now().UTC()
	// The (code_id, user_id) primary key makes a second redemption by the same
	// user fail here — turn that into a clean error.
	if _, err := tx.Exec(
		`INSERT INTO redeem_code_redemptions (code_id, user_id, created_at) VALUES (?, ?, ?)`,
		codeID, userID, now,
	); err != nil {
		if isUniqueViolation(err) {
			return 0, 0, ErrCodeAlreadyRedeemed
		}
		return 0, 0, err
	}

	// Authoritative, race-safe cap: bump the use count only while it's still
	// under max_uses. If two redemptions by different users race at the
	// boundary, this conditional UPDATE lets exactly one through — the other
	// affects 0 rows and is rolled back. (The read above is just a fast path.)
	res, err := tx.Exec(`UPDATE redeem_codes SET uses = uses + 1 WHERE id = ? AND (max_uses = 0 OR uses < max_uses)`, codeID)
	if err != nil {
		return 0, 0, err
	}
	if n, err := res.RowsAffected(); err != nil {
		return 0, 0, err
	} else if n == 0 {
		return 0, 0, ErrCodeExhausted
	}

	var balance int64
	if err := tx.QueryRow(`SELECT coins FROM users WHERE id = ?`, userID).Scan(&balance); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, 0, ErrNotFound
		}
		return 0, 0, err
	}
	if coins > 0 && balance > math.MaxInt64-coins {
		return 0, 0, errors.New("redeem would overflow balance")
	}
	newBalance = balance + coins
	if _, err := tx.Exec(`UPDATE users SET coins = ?, updated_at = ? WHERE id = ?`, newBalance, now, userID); err != nil {
		return 0, 0, err
	}
	if _, err := tx.Exec(
		`INSERT INTO ledger_entries (id, user_id, amount, reason, metadata, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		uuid.NewString(), userID, coins, models.ReasonRedeemCode, code, now,
	); err != nil {
		return 0, 0, err
	}

	return coins, newBalance, tx.Commit()
}
