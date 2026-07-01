package models

import "time"

type EggVariable struct {
	Name         string `json:"name"`
	Env          string `json:"env"`
	Default      string `json:"default"`
	UserEditable bool   `json:"user_editable"`
}

type Egg struct {
	ID          string
	Name        string
	Category    string
	Description string
	DockerImage string
	Startup     string
	StopCommand string
	Variables   []EggVariable
	CreatedAt   time.Time
}
