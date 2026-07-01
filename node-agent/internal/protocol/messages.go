// Package protocol defines the JSON message contract exchanged over the
// persistent WebSocket connection node-agent opens outward to panel-api.
// panel-api (a separate Go module) mirrors this schema by convention rather
// than a shared Go package — the wire format is plain JSON either way.
package protocol

import "encoding/json"

// Envelope wraps every message in both directions. Payload is left as raw
// JSON so a receiver only decodes the shape it expects for that Type.
type Envelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// Message types sent by node-agent to panel-api.
const (
	TypeHello     = "hello"     // sent once, immediately after connecting
	TypeHeartbeat = "heartbeat" // sent periodically with live stats
	TypeEvent     = "event"     // console output line, state change, backup done, ...
	TypeAck       = "ack"       // acknowledges a command by ID
)

// Message types sent by panel-api to node-agent.
const (
	TypeCommand = "command"
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

// EventKind values.
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

// Command actions requested by panel-api.
const (
	ActionCreate  = "create"
	ActionStart   = "start"
	ActionStop    = "stop"
	ActionKill    = "kill"
	ActionRemove  = "remove"
	ActionAttach  = "attach"
	ActionConsole = "console_input"
)

type PortBinding struct {
	ContainerPort string `json:"container_port"` // e.g. "25565/tcp"
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
	Input       string         `json:"input,omitempty"` // for console_input
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
