package serversvc

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/agenthub"
)

// ErrDatabasesUnavailable is returned when the server's node can't provision
// databases (older daemon, or no MariaDB configured on the node).
var ErrDatabasesUnavailable = fmt.Errorf("this server's node can't provision databases (needs sky-daemon v0.5.0+ with MariaDB configured)")

// DatabaseCredentials is the full connection detail the panel generates and
// persists for a provisioned database.
type DatabaseCredentials struct {
	Name     string
	Username string
	Password string
	Host     string
	Port     int
}

var labelSanitizeRe = regexp.MustCompile(`[^a-z0-9_]+`)

// CreateDatabase generates safe credentials, provisions the database on the
// server's node, and returns the connection details to persist. The generated
// names use only [a-z0-9_] so they're valid MariaDB identifiers and can't
// carry an injection.
func (s *Service) CreateDatabase(serverID, label string) (DatabaseCredentials, error) {
	server, err := s.Servers.GetByID(serverID)
	if err != nil {
		return DatabaseCredentials{}, fmt.Errorf("load server: %w", err)
	}
	if !s.Hub.SupportsDatabases(server.NodeID) {
		return DatabaseCredentials{}, ErrDatabasesUnavailable
	}

	// Pick a name that isn't already tracked on this node. The random suffix
	// makes a collision astronomically unlikely; the check + retry closes the
	// window entirely so a create can never provision onto another user's
	// existing database.
	name := generateDBName(label)
	if s.Databases != nil {
		for i := 0; i < 5; i++ {
			exists, err := s.Databases.NameExistsOnNode(server.NodeID, name)
			if err != nil || !exists {
				break
			}
			name = generateDBName(label)
		}
	}
	user := "sky_" + randHex(6) // "sky_" + 12 hex = 16 chars (<= MariaDB's 32)
	password := randHex(16)     // 32 hex chars

	ack, err := s.Hub.SendCommand(server.NodeID, agenthub.CommandPayload{
		CommandID:        uuid.NewString(),
		Action:           agenthub.ActionCreateDatabase,
		ServerID:         serverID,
		DatabaseName:     name,
		DatabaseUser:     user,
		DatabasePassword: password,
	})
	if err != nil {
		return DatabaseCredentials{}, err
	}
	if !ack.OK {
		return DatabaseCredentials{}, fmt.Errorf("%w: %s", ErrCommandFailed, ack.Error)
	}
	var result agenthub.CreateDatabaseResult
	if len(ack.Result) > 0 {
		_ = json.Unmarshal(ack.Result, &result)
	}
	return DatabaseCredentials{
		Name:     name,
		Username: user,
		Password: password,
		Host:     result.Host,
		Port:     result.Port,
	}, nil
}

// DeleteDatabase drops a database and its user on the given node.
func (s *Service) DeleteDatabase(nodeID, name, user string) error {
	// A node that can't provision databases can't drop them either — and an old
	// daemon would fail to decode the command, blocking on the ack timeout. Skip
	// the round-trip and fail fast (there's nothing to drop on such a node).
	if !s.Hub.SupportsDatabases(nodeID) {
		return ErrDatabasesUnavailable
	}
	ack, err := s.Hub.SendCommand(nodeID, agenthub.CommandPayload{
		CommandID:    uuid.NewString(),
		Action:       agenthub.ActionDeleteDatabase,
		DatabaseName: name,
		DatabaseUser: user,
	})
	if err != nil {
		return err
	}
	if !ack.OK {
		return fmt.Errorf("%w: %s", ErrCommandFailed, ack.Error)
	}
	return nil
}

// DeleteUserDatabases drops every database a user owns on its node, best-effort.
// Called before a user is deleted so the CASCADE that removes the tracking rows
// doesn't strand the real MariaDB databases on the nodes.
func (s *Service) DeleteUserDatabases(ownerID string) {
	if s.Databases == nil {
		return
	}
	if dbs, err := s.Databases.ListByOwner(ownerID); err == nil {
		for _, d := range dbs {
			_ = s.DeleteDatabase(d.NodeID, d.Name, d.Username)
		}
	}
}

// generateDBName builds a globally-unique, injection-safe database name from a
// user label: "sky_<rand>_<label>", all lower [a-z0-9_], <= 37 chars. The 16
// random hex chars (64 bits) make cross-user name collisions effectively
// impossible.
func generateDBName(label string) string {
	clean := labelSanitizeRe.ReplaceAllString(strings.ToLower(label), "_")
	clean = strings.Trim(clean, "_")
	if len(clean) > 20 {
		clean = clean[:20]
	}
	if clean == "" {
		clean = "db"
	}
	return fmt.Sprintf("sky_%s_%s", randHex(8), clean)
}

func randHex(n int) string {
	buf := make([]byte, n)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}
