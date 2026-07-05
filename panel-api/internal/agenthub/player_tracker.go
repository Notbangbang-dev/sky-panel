package agenthub

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// playerTracker derives a live player roster (and server version) per server by
// parsing the console lines the node already streams to the panel — join/leave
// messages, the `list` command's output, and the startup version line. It's the
// single source of truth for both the Players tab and the public status page,
// so neither has to parse the console itself.

// How long a server's roster is kept after its last console activity before it
// is reclaimed — long enough that an active server never loses its roster, short
// enough that stopped/deleted servers don't accumulate forever.
const rosterStaleAfter = 6 * time.Hour

var (
	// "<name> joined/left the game" — matched AFTER the log prefix is stripped
	// and anchored to end-of-line, so chat like "<Steve> I left the game early"
	// (which starts with "<") can't be mistaken for a real join/leave.
	joinLeaveRe = regexp.MustCompile(`^([A-Za-z0-9_]{1,16}) (joined|left) the game$`)
	// Output of the `list` command, e.g. "There are 2 of a max of 20 players online: Steve, Alex".
	listRe = regexp.MustCompile(`^There are (\d+) of a max of (\d+) players online:\s*(.*)$`)
	// "Starting minecraft server version 1.21.1"
	versionRe = regexp.MustCompile(`^Starting minecraft server version (\S+)`)
	// A genuine Minecraft/Java log prefix at the START of the line, e.g.
	// "[12:00:00] [Server thread/INFO]: " or "[12:00:00 INFO]: ". Anchoring on
	// this (instead of the LAST "]: " in the line) stops a chat message that
	// embeds "]: " from sliding the parse window past the real prefix and
	// spoofing a `list`/join line.
	logPrefixRe = regexp.MustCompile(`^\[[^\]]*\](?: \[[^\]]*\])?:\s`)
	// Valid Minecraft username, applied to names parsed from `list` output so a
	// crafted line can't inject arbitrary strings into the roster.
	validName = regexp.MustCompile(`^[A-Za-z0-9_]{1,16}$`)
)

type serverPlayers struct {
	players map[string]struct{}
	max     int
	version string
	updated time.Time
}

type playerTracker struct {
	mu      sync.Mutex
	servers map[string]*serverPlayers
}

func newPlayerTracker() *playerTracker {
	return &playerTracker{servers: make(map[string]*serverPlayers)}
}

func (t *playerTracker) entry(serverID string) *serverPlayers {
	sp, ok := t.servers[serverID]
	if !ok {
		sp = &serverPlayers{players: make(map[string]struct{}), updated: time.Now()}
		t.servers[serverID] = sp
	}
	return sp
}

// forget drops a server's roster immediately (e.g. on deletion).
func (t *playerTracker) forget(serverID string) {
	t.mu.Lock()
	delete(t.servers, serverID)
	t.mu.Unlock()
}

// sweep reclaims rosters for servers that have gone quiet past the TTL so the
// map can't grow without bound as servers are created and destroyed.
func (t *playerTracker) sweep() {
	t.mu.Lock()
	for id, sp := range t.servers {
		if time.Since(sp.updated) > rosterStaleAfter {
			delete(t.servers, id)
		}
	}
	t.mu.Unlock()
}

func (t *playerTracker) sweepLoop() {
	tk := time.NewTicker(30 * time.Minute)
	defer tk.Stop()
	for range tk.C {
		t.sweep()
	}
}

// observe feeds one event into the tracker.
func (t *playerTracker) observe(ev EventPayload) {
	if ev.ServerID == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	// A server that isn't running has no players — clear the roster so a stale
	// list can't linger after a stop/crash.
	if ev.Kind == EventStateChanged {
		if ev.Message != "running" {
			if sp, ok := t.servers[ev.ServerID]; ok {
				sp.players = make(map[string]struct{})
				sp.updated = time.Now()
			}
		}
		return
	}
	if ev.Kind != EventConsoleLine {
		return
	}

	line := stripLogPrefix(ev.Message)

	if m := listRe.FindStringSubmatch(line); m != nil {
		sp := t.entry(ev.ServerID)
		sp.max, _ = strconv.Atoi(m[2])
		sp.players = make(map[string]struct{})
		for _, name := range strings.Split(m[3], ",") {
			name = strings.TrimSpace(name)
			// Only accept real usernames — a crafted line can't inject
			// arbitrary strings (or oversized garbage) into the roster.
			if validName.MatchString(name) {
				sp.players[name] = struct{}{}
			}
		}
		sp.updated = time.Now()
		return
	}
	if m := joinLeaveRe.FindStringSubmatch(line); m != nil {
		sp := t.entry(ev.ServerID)
		if m[2] == "joined" {
			sp.players[m[1]] = struct{}{}
		} else {
			delete(sp.players, m[1])
		}
		sp.updated = time.Now()
		return
	}
	if m := versionRe.FindStringSubmatch(line); m != nil {
		sp := t.entry(ev.ServerID)
		sp.version = m[1]
		sp.updated = time.Now()
	}
}

// PlayerInfo is the roster snapshot returned to callers.
type PlayerInfo struct {
	Players []string `json:"players"`
	Max     int      `json:"max"`
	Version string   `json:"version"`
}

func (t *playerTracker) get(serverID string) PlayerInfo {
	t.mu.Lock()
	defer t.mu.Unlock()
	sp, ok := t.servers[serverID]
	if !ok {
		return PlayerInfo{Players: []string{}}
	}
	names := make([]string, 0, len(sp.players))
	for n := range sp.players {
		names = append(names, n)
	}
	sort.Strings(names)
	return PlayerInfo{Players: names, Max: sp.max, Version: sp.version}
}

// stripLogPrefix removes a Minecraft/Java log prefix ("[12:00:00 INFO]: ")
// leaving just the message body, so the parsers can anchor on it. It matches the
// prefix at the START of the line (not the last "]: " anywhere in it) so a chat
// message that itself embeds "]: " can't slide the parse window past the real
// prefix and spoof a `list`/join line.
func stripLogPrefix(msg string) string {
	if loc := logPrefixRe.FindStringIndex(msg); loc != nil {
		return strings.TrimSpace(msg[loc[1]:])
	}
	return strings.TrimSpace(msg)
}
