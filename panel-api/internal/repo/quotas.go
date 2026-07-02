package repo

import "database/sql"

// Quotas stores each user's bonus quota — the amount added on top of the
// global defaults, accumulated from store purchases and admin grants. A user
// with no row simply has zero bonus.
type Quotas struct {
	db *sql.DB
}

func NewQuotas(db *sql.DB) *Quotas {
	return &Quotas{db: db}
}

// Bonus holds a user's additional quota over the global defaults.
type Bonus struct {
	MemoryBytes int64
	CPUPercent  int
	DiskBytes   int64
}

// Get returns the user's bonus quota, or an all-zero Bonus if they have no row.
func (r *Quotas) Get(userID string) (Bonus, error) {
	var b Bonus
	err := r.db.QueryRow(
		`SELECT bonus_memory_bytes, bonus_cpu_percent, bonus_disk_bytes FROM user_quotas WHERE user_id = ?`,
		userID,
	).Scan(&b.MemoryBytes, &b.CPUPercent, &b.DiskBytes)
	if err == sql.ErrNoRows {
		return Bonus{}, nil
	}
	return b, err
}

// Add increments a user's bonus quota (used by store purchases and admin
// grants that stack). Creates the row if it doesn't exist.
func (r *Quotas) Add(userID string, memoryBytes int64, cpuPercent int, diskBytes int64) error {
	_, err := r.db.Exec(
		`INSERT INTO user_quotas (user_id, bonus_memory_bytes, bonus_cpu_percent, bonus_disk_bytes)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(user_id) DO UPDATE SET
		   bonus_memory_bytes = bonus_memory_bytes + excluded.bonus_memory_bytes,
		   bonus_cpu_percent  = bonus_cpu_percent  + excluded.bonus_cpu_percent,
		   bonus_disk_bytes   = bonus_disk_bytes   + excluded.bonus_disk_bytes`,
		userID, memoryBytes, cpuPercent, diskBytes,
	)
	return err
}

// Set replaces a user's bonus quota with absolute values (used by the admin
// console to grant or reset a user's extra quota).
func (r *Quotas) Set(userID string, memoryBytes int64, cpuPercent int, diskBytes int64) error {
	_, err := r.db.Exec(
		`INSERT INTO user_quotas (user_id, bonus_memory_bytes, bonus_cpu_percent, bonus_disk_bytes)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(user_id) DO UPDATE SET
		   bonus_memory_bytes = excluded.bonus_memory_bytes,
		   bonus_cpu_percent  = excluded.bonus_cpu_percent,
		   bonus_disk_bytes   = excluded.bonus_disk_bytes`,
		userID, memoryBytes, cpuPercent, diskBytes,
	)
	return err
}
