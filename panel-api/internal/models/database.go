package models

import "time"

// Database is a MariaDB database provisioned for a user on the node that hosts
// one of their servers. The panel generates and stores the credentials so the
// owner can view them; the actual database lives on the node's MariaDB server.
type Database struct {
	ID        string    `json:"id"`
	OwnerID   string    `json:"owner_id"`
	ServerID  string    `json:"server_id"`
	NodeID    string    `json:"node_id"`
	Name      string    `json:"name"`
	Username  string    `json:"username"`
	Password  string    `json:"password"`
	Host      string    `json:"host"`
	Port      int       `json:"port"`
	CreatedAt time.Time `json:"created_at"`
}
