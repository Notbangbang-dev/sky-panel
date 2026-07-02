package repo

import (
	"database/sql"
	"errors"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
)

// ErrBalanceOverflow guards against a credit so large it would wrap int64.
var ErrBalanceOverflow = errors.New("coin balance would overflow")

var ErrInsufficientBalance = errors.New("insufficient coin balance")

type Ledger struct {
	db *sql.DB
}

func NewLedger(db *sql.DB) *Ledger {
	return &Ledger{db: db}
}

// AddEntry atomically applies amount (positive or negative) to userID's
// cached coin balance and records the ledger entry, returning the resulting
// balance. It fails with ErrInsufficientBalance rather than letting a
// balance go negative.
func (r *Ledger) AddEntry(userID string, amount int64, reason, metadata string) (newBalance int64, err error) {
	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var balance int64
	err = tx.QueryRow(`SELECT coins FROM users WHERE id = ?`, userID).Scan(&balance)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, err
	}

	if amount > 0 && balance > math.MaxInt64-amount {
		return 0, ErrBalanceOverflow
	}
	newBalance = balance + amount
	if newBalance < 0 {
		return 0, ErrInsufficientBalance
	}

	now := time.Now().UTC()
	if _, err := tx.Exec(`UPDATE users SET coins = ?, updated_at = ? WHERE id = ?`, newBalance, now, userID); err != nil {
		return 0, err
	}

	if _, err := tx.Exec(
		`INSERT INTO ledger_entries (id, user_id, amount, reason, metadata, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		uuid.NewString(), userID, amount, reason, metadata, now,
	); err != nil {
		return 0, err
	}

	return newBalance, tx.Commit()
}

// Transfer atomically moves amount coins from one user to another in a single
// transaction: it debits `fromUserID` (failing with ErrInsufficientBalance
// rather than going negative), credits `toUserID`, and records a ledger entry
// on each side. Returns the sender's resulting balance.
func (r *Ledger) Transfer(fromUserID, toUserID string, amount int64, reasonFrom, reasonTo, metadata string) (fromBalance int64, err error) {
	if amount <= 0 {
		return 0, errors.New("transfer amount must be positive")
	}
	if fromUserID == toUserID {
		return 0, errors.New("cannot transfer to self")
	}

	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	now := time.Now().UTC()

	// Debit the sender.
	var senderBalance int64
	err = tx.QueryRow(`SELECT coins FROM users WHERE id = ?`, fromUserID).Scan(&senderBalance)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, err
	}
	fromBalance = senderBalance - amount
	if fromBalance < 0 {
		return 0, ErrInsufficientBalance
	}

	// The recipient must exist (checked inside the tx so it can't vanish mid-flight).
	var recipientBalance int64
	err = tx.QueryRow(`SELECT coins FROM users WHERE id = ?`, toUserID).Scan(&recipientBalance)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, err
	}

	if recipientBalance > math.MaxInt64-amount {
		return 0, ErrBalanceOverflow
	}
	if _, err := tx.Exec(`UPDATE users SET coins = ?, updated_at = ? WHERE id = ?`, fromBalance, now, fromUserID); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(`UPDATE users SET coins = ?, updated_at = ? WHERE id = ?`, recipientBalance+amount, now, toUserID); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(
		`INSERT INTO ledger_entries (id, user_id, amount, reason, metadata, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		uuid.NewString(), fromUserID, -amount, reasonFrom, metadata, now,
	); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(
		`INSERT INTO ledger_entries (id, user_id, amount, reason, metadata, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		uuid.NewString(), toUserID, amount, reasonTo, metadata, now,
	); err != nil {
		return 0, err
	}

	return fromBalance, tx.Commit()
}

func (r *Ledger) ListByUser(userID string, limit int) ([]*models.LedgerEntry, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, amount, reason, metadata, created_at FROM ledger_entries
		 WHERE user_id = ? ORDER BY created_at DESC, rowid DESC LIMIT ?`, userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*models.LedgerEntry
	for rows.Next() {
		var e models.LedgerEntry
		if err := rows.Scan(&e.ID, &e.UserID, &e.Amount, &e.Reason, &e.Metadata, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &e)
	}
	return out, rows.Err()
}
