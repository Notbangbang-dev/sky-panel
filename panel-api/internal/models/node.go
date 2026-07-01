package models

import "time"

type Node struct {
	ID           string
	Name         string
	TokenHash    string
	Address      string
	DockerSocket string
	CreatedAt    time.Time
}
