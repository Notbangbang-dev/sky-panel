// Package agenthub is panel-api's side of the persistent outbound WebSocket
// connection each sky-daemon opens inward. The message schema here mirrors
// sky-daemon's `protocol` crate by convention (separate repos communicating
// over plain signed JSON, not a shared package).
package agenthub

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// Envelope is every message on the wire, in both directions. Payload stays
// as raw, unparsed JSON so the signature always covers the exact bytes
// that were transmitted — re-serializing a parsed value could reorder
// fields or change whitespace and silently break verification.
type Envelope struct {
	Type      string          `json:"type"`
	Timestamp int64           `json:"timestamp"`
	Nonce     string          `json:"nonce"`
	Payload   json.RawMessage `json:"payload"`
	Sig       string          `json:"sig"`
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
	ActionCreate     = "create"
	ActionStart      = "start"
	ActionStop       = "stop"
	ActionKill       = "kill"
	ActionRemove     = "remove"
	ActionConsole    = "console_input"
	ActionListFiles  = "list_files"
	ActionReadFile   = "read_file"
	ActionWriteFile  = "write_file"
	ActionRenameFile = "rename_file"
	ActionDeleteFile = "delete_file"
	ActionMkdir      = "mkdir"

	ActionBackup        = "backup"
	ActionListBackups   = "list_backups"
	ActionRestoreBackup = "restore_backup"
	ActionDeleteBackup  = "delete_backup"
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
	CommandID      string         `json:"command_id"`
	Action         string         `json:"action"`
	ServerID       string         `json:"server_id"`
	ContainerID    string         `json:"container_id,omitempty"`
	Spec           *ContainerSpec `json:"spec,omitempty"`
	Input          string         `json:"input,omitempty"`
	Path           string         `json:"path,omitempty"`
	NewPath        string         `json:"new_path,omitempty"`
	ContentBase64  string         `json:"content_base64,omitempty"`
}

type AckPayload struct {
	CommandID string          `json:"command_id"`
	OK        bool            `json:"ok"`
	Error     string          `json:"error,omitempty"`
	Result    json.RawMessage `json:"result,omitempty"`
}

type FileEntry struct {
	Name      string `json:"name"`
	IsDir     bool   `json:"is_dir"`
	SizeBytes uint64 `json:"size_bytes"`
}

type ListFilesResult struct {
	Entries []FileEntry `json:"entries"`
}

type ReadFileResult struct {
	ContentBase64 string `json:"content_base64"`
	SizeBytes     uint64 `json:"size_bytes"`
}

type BackupEntry struct {
	Filename  string `json:"filename"`
	SizeBytes uint64 `json:"size_bytes"`
	// CreatedAt is unix seconds (the daemon has no date library; the web
	// formats it). Derived from the archive file's modified time.
	CreatedAt int64 `json:"created_at"`
}

type BackupResult struct {
	Filename  string `json:"filename"`
	SizeBytes uint64 `json:"size_bytes"`
}

type ListBackupsResult struct {
	Backups []BackupEntry `json:"backups"`
}

// EncodeSigned builds and signs a new envelope carrying payload, using
// secret as the HMAC key (the node's raw token).
func EncodeSigned(secret []byte, kind string, payload any) (Envelope, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return Envelope{}, err
	}

	nonce, err := randomNonce()
	if err != nil {
		return Envelope{}, err
	}

	timestamp := time.Now().Unix()
	sig := Sign(secret, kind, timestamp, nonce, payloadBytes)

	return Envelope{Type: kind, Timestamp: timestamp, Nonce: nonce, Payload: json.RawMessage(payloadBytes), Sig: sig}, nil
}

// Verify checks this envelope's signature and timestamp freshness against
// secret. It does not check nonce uniqueness — that's stateful and belongs
// to the caller (see nonceCache), not this otherwise-pure type.
func (e Envelope) Verify(secret []byte) bool {
	if !e.TimestampIsFresh() {
		return false
	}
	return Verify(secret, e.Type, e.Timestamp, e.Nonce, e.Payload, e.Sig)
}

func (e Envelope) TimestampIsFresh() bool {
	delta := time.Now().Unix() - e.Timestamp
	if delta < 0 {
		delta = -delta
	}
	return delta <= MaxClockSkewSecs
}

func randomNonce() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
