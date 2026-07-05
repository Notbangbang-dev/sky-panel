// Package quotasvc enforces per-user resource quotas: the total memory, CPU,
// and disk a user may allocate across all of their servers. A user's
// effective quota is the global default (admin-configurable via settings)
// plus any bonus they've bought from the store or been granted by an admin.
package quotasvc

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/repo"
)

// Global defaults applied to every user when the corresponding setting is
// unset. Chosen so a fresh account can comfortably run a server or two.
const (
	DefaultMemoryBytes int64 = 2048 * 1024 * 1024  // 2 GB
	DefaultCPUPercent  int   = 200                 // two cores
	DefaultDiskBytes   int64 = 10240 * 1024 * 1024 // 10 GB
	DefaultDatabases   int   = 0                   // databases are bought from the store

	SettingMemoryBytes       = "quota.default_memory_bytes"
	SettingCPUPercent        = "quota.default_cpu_percent"
	SettingDiskBytes         = "quota.default_disk_bytes"
	SettingDatabases         = "quota.default_databases"
	SettingAllowUnlimitedCPU = "quota.allow_unlimited_cpu"
)

// ErrUnlimitedCPU is returned when a non-admin requests unlimited CPU
// (cpu_limit 0) but an admin has disabled that, so CPU must stay quota-bound.
var ErrUnlimitedCPU = errors.New("unlimited CPU is disabled; set a CPU limit within your quota")

// Quota is a set of resource limits.
type Quota struct {
	MemoryBytes int64 `json:"memory_bytes"`
	CPUPercent  int   `json:"cpu_percent"`
	DiskBytes   int64 `json:"disk_bytes"`
	Databases   int   `json:"databases"`
}

// Usage is a user's current allocation across all their servers.
type Usage struct {
	Servers     int   `json:"servers"`
	MemoryBytes int64 `json:"memory_bytes"`
	CPUPercent  int   `json:"cpu_percent"`
	DiskBytes   int64 `json:"disk_bytes"`
	Databases   int   `json:"databases"`
}

// Error reports which quota dimension a request would exceed.
type Error struct {
	Dimension string // "memory" | "cpu" | "disk" | "databases"
	Limit     int64
	Have      int64
	Requested int64
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s quota exceeded: %d in use + %d requested exceeds limit of %d",
		e.Dimension, e.Have, e.Requested, e.Limit)
}

type Service struct {
	Servers  *repo.Servers
	Quotas   *repo.Quotas
	Settings *repo.Settings
	// Databases is optional; when set, database usage/limits are reported and
	// enforced. Nil in tests that don't exercise the databases dimension.
	Databases *repo.Databases
}

func NewService(servers *repo.Servers, quotas *repo.Quotas, settings *repo.Settings) *Service {
	return &Service{Servers: servers, Quotas: quotas, Settings: settings}
}

// Effective returns the user's quota: global defaults plus their bonus.
func (s *Service) Effective(userID string) (Quota, error) {
	base := Quota{
		MemoryBytes: s.settingInt64(SettingMemoryBytes, DefaultMemoryBytes),
		CPUPercent:  int(s.settingInt64(SettingCPUPercent, int64(DefaultCPUPercent))),
		DiskBytes:   s.settingInt64(SettingDiskBytes, DefaultDiskBytes),
	}
	bonus, err := s.Quotas.Get(userID)
	if err != nil {
		return Quota{}, err
	}
	return Quota{
		MemoryBytes: base.MemoryBytes + bonus.MemoryBytes,
		CPUPercent:  base.CPUPercent + bonus.CPUPercent,
		DiskBytes:   base.DiskBytes + bonus.DiskBytes,
		Databases:   int(s.settingInt64(SettingDatabases, int64(DefaultDatabases))) + bonus.Databases,
	}, nil
}

// Usage sums the resources allocated across all of a user's servers,
// optionally excluding one server (used when re-checking an update in place).
func (s *Service) Usage(userID, excludeServerID string) (Usage, error) {
	servers, err := s.Servers.ListByOwner(userID)
	if err != nil {
		return Usage{}, err
	}
	var u Usage
	for _, srv := range servers {
		if srv.ID == excludeServerID {
			continue
		}
		u.Servers++
		u.MemoryBytes += srv.MemoryBytes
		u.CPUPercent += srv.CPULimit
		u.DiskBytes += srv.DiskBytes
	}
	if s.Databases != nil {
		if n, err := s.Databases.CountByOwner(userID); err == nil {
			u.Databases = n
		}
	}
	return u, nil
}

// CheckDatabaseCreate verifies a user has room for one more database within
// their databases quota.
func (s *Service) CheckDatabaseCreate(userID string) error {
	limit, err := s.Effective(userID)
	if err != nil {
		return err
	}
	usage, err := s.Usage(userID, "")
	if err != nil {
		return err
	}
	if usage.Databases+1 > limit.Databases {
		return &Error{Dimension: "databases", Limit: int64(limit.Databases), Have: int64(usage.Databases), Requested: 1}
	}
	return nil
}

// CheckCreate verifies a new server's requested resources fit within the
// user's remaining quota. Returns an *Error naming the first dimension that
// would be exceeded.
func (s *Service) CheckCreate(userID string, memoryBytes int64, cpuPercent int, diskBytes int64) error {
	return s.check(userID, "", memoryBytes, cpuPercent, diskBytes)
}

// CheckUpdate is like CheckCreate but excludes the server being updated from
// the current usage total, so re-saving with the same limits always passes.
func (s *Service) CheckUpdate(userID, serverID string, memoryBytes int64, cpuPercent int, diskBytes int64) error {
	return s.check(userID, serverID, memoryBytes, cpuPercent, diskBytes)
}

// AllowUnlimitedCPU reports whether a server may be created with cpu_limit 0
// (unlimited on the node). Admins can turn this off so CPU is always bounded
// by the user's quota like memory and disk.
func (s *Service) AllowUnlimitedCPU() bool {
	return s.settingBool(SettingAllowUnlimitedCPU, true)
}

func (s *Service) check(userID, excludeServerID string, memoryBytes int64, cpuPercent int, diskBytes int64) error {
	// cpu_limit 0 means "unlimited" on the node, which would sidestep the CPU
	// quota entirely — refuse it when an admin has disabled unlimited CPU.
	if cpuPercent <= 0 && !s.AllowUnlimitedCPU() {
		return ErrUnlimitedCPU
	}

	limit, err := s.Effective(userID)
	if err != nil {
		return err
	}
	usage, err := s.Usage(userID, excludeServerID)
	if err != nil {
		return err
	}
	if usage.MemoryBytes+memoryBytes > limit.MemoryBytes {
		return &Error{Dimension: "memory", Limit: limit.MemoryBytes, Have: usage.MemoryBytes, Requested: memoryBytes}
	}
	if int64(usage.CPUPercent+cpuPercent) > int64(limit.CPUPercent) {
		return &Error{Dimension: "cpu", Limit: int64(limit.CPUPercent), Have: int64(usage.CPUPercent), Requested: int64(cpuPercent)}
	}
	if usage.DiskBytes+diskBytes > limit.DiskBytes {
		return &Error{Dimension: "disk", Limit: limit.DiskBytes, Have: usage.DiskBytes, Requested: diskBytes}
	}
	return nil
}

func (s *Service) settingInt64(key string, fallback int64) int64 {
	v, found, err := s.Settings.Get(key)
	if err != nil || !found {
		return fallback
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return fallback
	}
	return n
}

func (s *Service) settingBool(key string, fallback bool) bool {
	v, found, err := s.Settings.Get(key)
	if err != nil || !found {
		return fallback
	}
	return strings.EqualFold(strings.TrimSpace(v), "true")
}
