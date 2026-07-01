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
	Variables   map[string]string
	PrimaryPort int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Allocation struct {
	ID       string
	NodeID   string
	Port     int
	ServerID *string
}
