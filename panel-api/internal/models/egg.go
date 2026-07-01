package models

import "time"

type EggVariable struct {
	Name         string `json:"name"`
	Env          string `json:"env"`
	Default      string `json:"default"`
	UserEditable bool   `json:"user_editable"`
}

type Egg struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Category    string        `json:"category"`
	Description string        `json:"description"`
	DockerImage string        `json:"docker_image"`
	Startup     string        `json:"startup"`
	StopCommand string        `json:"stop_command"`
	Variables   []EggVariable `json:"variables"`
	CreatedAt   time.Time     `json:"created_at"`
}
