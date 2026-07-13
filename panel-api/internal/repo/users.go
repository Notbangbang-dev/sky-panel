package repo

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrDuplicate = errors.New("already exists")
)

type Users struct {
	db *sql.DB
}

func NewUsers(db *sql.DB) *Users {
	return &Users{db: db}
}

// Count returns the total number of users. Used to decide whether a newly
// registered user should be bootstrapped as the first admin.
func (r *Users) Count() (int, error) {
	var n int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

// CountAdmins returns how many users currently hold the admin role. Used to
// refuse demoting or deleting the last remaining admin, which would otherwise
// lock the whole instance out of the admin console.
func (r *Users) CountAdmins() (int, error) {
	var n int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM users WHERE role = ?`, string(models.RoleAdmin)).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

func (r *Users) Create(u *models.User) error {
	_, err := r.db.Exec(
		`INSERT INTO users (id, email, username, password_hash, role, coins, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		u.ID, u.Email, u.Username, u.PasswordHash, string(u.Role), u.Coins, u.CreatedAt, u.UpdatedAt,
	)
	if isUniqueViolation(err) {
		return ErrDuplicate
	}
	return err
}

func (r *Users) GetByID(id string) (*models.User, error) {
	return r.scanOne(`SELECT id, email, username, password_hash, role, totp_secret, totp_enabled, coins, created_at, updated_at
		FROM users WHERE id = ?`, id)
}

func (r *Users) GetByEmail(email string) (*models.User, error) {
	return r.scanOne(`SELECT id, email, username, password_hash, role, totp_secret, totp_enabled, coins, created_at, updated_at
		FROM users WHERE email = ?`, email)
}

func (r *Users) GetByUsername(username string) (*models.User, error) {
	return r.scanOne(`SELECT id, email, username, password_hash, role, totp_secret, totp_enabled, coins, created_at, updated_at
		FROM users WHERE username = ?`, username)
}

func (r *Users) SetTOTP(userID, secret string, enabled bool) error {
	res, err := r.db.Exec(
		`UPDATE users SET totp_secret = ?, totp_enabled = ?, updated_at = ? WHERE id = ?`,
		secret, enabled, time.Now().UTC(), userID,
	)
	return checkRowsAffected(res, err)
}

func (r *Users) SetPasswordHash(userID, hash string) error {
	res, err := r.db.Exec(
		`UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`,
		hash, time.Now().UTC(), userID,
	)
	return checkRowsAffected(res, err)
}

func (r *Users) SetRole(userID string, role models.Role) error {
	res, err := r.db.Exec(`UPDATE users SET role = ?, updated_at = ? WHERE id = ?`, string(role), time.Now().UTC(), userID)
	return checkRowsAffected(res, err)
}

func (r *Users) Delete(userID string) error {
	res, err := r.db.Exec(`DELETE FROM users WHERE id = ?`, userID)
	return checkRowsAffected(res, err)
}

func (r *Users) List() ([]*models.User, error) {
	rows, err := r.db.Query(
		`SELECT id, email, username, password_hash, role, totp_secret, totp_enabled, coins, created_at, updated_at
		 FROM users ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*models.User
	for rows.Next() {
		u, err := scanUserRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// TopByCoins returns the highest-balance users (username + coins), for the
// leaderboard.
func (r *Users) TopByCoins(limit int) ([]*models.User, error) {
	rows, err := r.db.Query(
		`SELECT id, username, coins FROM users ORDER BY coins DESC, created_at LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Coins); err != nil {
			return nil, err
		}
		out = append(out, &u)
	}
	return out, rows.Err()
}

func (r *Users) scanOne(query string, args ...any) (*models.User, error) {
	u, err := scanUserRow(r.db.QueryRow(query, args...))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

func scanUserRow(row rowScanner) (*models.User, error) {
	var u models.User
	var role string
	var totpSecret sql.NullString
	var totpEnabled bool

	if err := row.Scan(&u.ID, &u.Email, &u.Username, &u.PasswordHash, &role, &totpSecret, &totpEnabled, &u.Coins, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil, err
	}

	u.Role = models.Role(role)
	u.TOTPSecret = totpSecret.String
	u.TOTPEnabled = totpEnabled
	return &u, nil
}

func checkRowsAffected(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint")
}
