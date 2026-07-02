package models

import "time"

type ServerStatus string

const (
	StatusInstalling ServerStatus = "installing"
	StatusOffline    ServerStatus = "offline"
	StatusRunning    ServerStatus = "running"
	StatusStopping   ServerStatus = "stopping"
	StatusErrored    ServerStatus = "errored"
)

type Server struct {
	ID          string
	OwnerID     string
	NodeID      string
	EggID       string
	Name        string
	ContainerID string
	Status      ServerStatus
	MemoryBytes int64
	// CPULimit is the CPU cap as a percentage of one core (100 = one full
	// core, 200 = two cores). 0 means unlimited.
	CPULimit int
	// DiskBytes is the declared disk allocation, counted against the owner's
	// disk quota. 0 means unspecified.
	DiskBytes           int64
	Variables           map[string]string
	PrimaryPort         int
	BackupIntervalHours int
	LastBackupAt        *time.Time
	// Suspended servers are stopped and their owner can't start or control
	// them until an admin unsuspends. Admins are exempt from the block.
	Suspended bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Allocation struct {
	ID       string
	NodeID   string
	Port     int
	ServerID *string
}
