package models

import "time"

type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

type User struct {
	ID           string
	Email        string
	Username     string
	PasswordHash string
	Role         Role
	TOTPSecret   string
	TOTPEnabled  bool
	Coins        int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
