// Package agenthub is panel-api's side of the persistent outbound WebSocket
// connection each node-agent opens inward. The message schema here mirrors
// node-agent's internal/protocol package by convention (they're separate Go
// modules communicating over plain JSON, not a shared package).
package agenthub

import "encoding/json"

type Envelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

const (
	TypeHello     = "hello"
	TypeHeartbeat = "heartbeat"
	TypeEvent     = "event"
	TypeAck       = "ack"
	TypeCommand   = "command"
)

type HelloPayload struct {
	NodeToken    string `json:"node_token"`
	AgentVersion string `json:"agent_version"`
}

type ContainerHeartbeat struct {
	ServerID string  `json:"server_id"`
	Running  bool    `json:"running"`
	CPU      float64 `json:"cpu_percent"`
	MemUsed  uint64  `json:"mem_used_bytes"`
	MemLimit uint64  `json:"mem_limit_bytes"`
	NetRx    uint64  `json:"net_rx_bytes"`
	NetTx    uint64  `json:"net_tx_bytes"`
}

type HeartbeatPayload struct {
	Containers []ContainerHeartbeat `json:"containers"`
}

const (
	EventConsoleLine  = "console_line"
	EventStateChanged = "state_changed"
	EventBackupDone   = "backup_done"
	EventBackupFailed = "backup_failed"
)

type EventPayload struct {
	ServerID string `json:"server_id"`
	Kind     string `json:"kind"`
	Message  string `json:"message"`
}

const (
	ActionCreate  = "create"
	ActionStart   = "start"
	ActionStop    = "stop"
	ActionKill    = "kill"
	ActionRemove  = "remove"
	ActionConsole = "console_input"
)

type PortBinding struct {
	ContainerPort string `json:"container_port"`
	HostPort      string `json:"host_port"`
}

type ContainerSpec struct {
	Name         string            `json:"name"`
	Image        string            `json:"image"`
	Cmd          []string          `json:"cmd"`
	Env          []string          `json:"env"`
	WorkingDir   string            `json:"working_dir"`
	Binds        []string          `json:"binds"`
	PortBindings []PortBinding     `json:"port_bindings"`
	MemoryBytes  int64             `json:"memory_bytes"`
	NanoCPUs     int64             `json:"nano_cpus"`
	Labels       map[string]string `json:"labels"`
}

type CommandPayload struct {
	CommandID   string         `json:"command_id"`
	Action      string         `json:"action"`
	ServerID    string         `json:"server_id"`
	ContainerID string         `json:"container_id,omitempty"`
	Spec        *ContainerSpec `json:"spec,omitempty"`
	Input       string         `json:"input,omitempty"`
}

type AckPayload struct {
	CommandID string `json:"command_id"`
	OK        bool   `json:"ok"`
	Error     string `json:"error,omitempty"`
}

func Encode(msgType string, payload any) (Envelope, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return Envelope{}, err
	}
	return Envelope{Type: msgType, Payload: raw}, nil
}
