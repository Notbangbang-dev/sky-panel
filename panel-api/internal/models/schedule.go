package models

import "time"

// Schedule actions.
const (
	ScheduleStart   = "start"
	ScheduleStop    = "stop"
	ScheduleRestart = "restart"
	ScheduleKill    = "kill"
	ScheduleBackup  = "backup"
	ScheduleCommand = "command"
)

// Schedule is a per-server automation that runs an action on a fixed interval.
type Schedule struct {
	ID              string
	ServerID        string
	Name            string
	Action          string
	Payload         string // console line, for Action == command
	IntervalMinutes int
	Enabled         bool
	LastRunAt       *time.Time
	CreatedAt       time.Time
}
