package agenthub

import "testing"

func line(id, msg string) EventPayload {
	return EventPayload{ServerID: id, Kind: EventConsoleLine, Message: msg}
}

func TestPlayerTrackerJoinLeave(t *testing.T) {
	tr := newPlayerTracker()
	tr.observe(line("s1", "[12:00:00] [Server thread/INFO]: Steve joined the game"))
	tr.observe(line("s1", "[12:00:01] [Server thread/INFO]: Alex joined the game"))
	got := tr.get("s1")
	if len(got.Players) != 2 || got.Players[0] != "Alex" || got.Players[1] != "Steve" {
		t.Fatalf("expected [Alex Steve], got %v", got.Players)
	}

	tr.observe(line("s1", "[12:01:00] [Server thread/INFO]: Steve left the game"))
	got = tr.get("s1")
	if len(got.Players) != 1 || got.Players[0] != "Alex" {
		t.Fatalf("expected [Alex] after Steve left, got %v", got.Players)
	}
}

func TestPlayerTrackerIgnoresChat(t *testing.T) {
	tr := newPlayerTracker()
	// A player typing a message that contains "joined the game" must not be
	// mistaken for a real join (chat lines start with "<name>").
	tr.observe(line("s1", "[12:00:00] [Server thread/INFO]: <Griefer> lol i just left the game"))
	tr.observe(line("s1", "[12:00:00] [Server thread/INFO]: <Griefer> Notch joined the game??"))
	if got := tr.get("s1"); len(got.Players) != 0 {
		t.Fatalf("chat lines must not add players, got %v", got.Players)
	}
}

func TestPlayerTrackerListSeedsRoster(t *testing.T) {
	tr := newPlayerTracker()
	tr.observe(line("s1", "[12:00:00] [Server thread/INFO]: Ghost joined the game"))
	// `list` output is authoritative — it replaces the roster.
	tr.observe(line("s1", "[12:00:05] [Server thread/INFO]: There are 2 of a max of 20 players online: Steve, Alex"))
	got := tr.get("s1")
	if got.Max != 20 {
		t.Errorf("expected max 20, got %d", got.Max)
	}
	if len(got.Players) != 2 || got.Players[0] != "Alex" || got.Players[1] != "Steve" {
		t.Fatalf("expected roster replaced with [Alex Steve], got %v", got.Players)
	}
}

func TestPlayerTrackerRejectsNestedPrefixSpoof(t *testing.T) {
	tr := newPlayerTracker()
	tr.observe(line("s1", "[12:00:00] [Server thread/INFO]: Steve joined the game"))
	// A chat/plugin line that embeds "]: " before crafted list text must not be
	// mistaken for real `list` output (which would wipe & replace the roster).
	tr.observe(line("s1", "[12:00:01] [Server thread/INFO]: [Rcon]: There are 9 of a max of 99 players online: pwned"))
	tr.observe(line("s1", "[12:00:02] [Server thread/INFO]: <Evil> ]: There are 0 of a max of 20 players online:"))
	got := tr.get("s1")
	if len(got.Players) != 1 || got.Players[0] != "Steve" {
		t.Fatalf("spoofed list lines must not touch the roster, got %v", got.Players)
	}
	if got.Max == 99 {
		t.Fatalf("spoofed list line must not set max, got %d", got.Max)
	}
}

func TestPlayerTrackerListRejectsBadNames(t *testing.T) {
	tr := newPlayerTracker()
	// Genuine list output whose names contain junk/oversized/HTML tokens — only
	// valid usernames should enter the roster.
	tr.observe(line("s1", "[12:00:00] [Server thread/INFO]: There are 3 of a max of 20 players online: Steve, <script>alert(1)</script>, waytoolongusername123"))
	got := tr.get("s1")
	if len(got.Players) != 1 || got.Players[0] != "Steve" {
		t.Fatalf("only valid usernames should be tracked, got %v", got.Players)
	}
}

func TestPlayerTrackerForget(t *testing.T) {
	tr := newPlayerTracker()
	tr.observe(line("s1", "[12:00:00] [Server thread/INFO]: Steve joined the game"))
	tr.forget("s1")
	if got := tr.get("s1"); len(got.Players) != 0 {
		t.Fatalf("expected empty roster after forget, got %v", got.Players)
	}
}

func TestPlayerTrackerVersionAndStop(t *testing.T) {
	tr := newPlayerTracker()
	tr.observe(line("s1", "[12:00:00] [Server thread/INFO]: Starting minecraft server version 1.21.1"))
	tr.observe(line("s1", "[12:00:01] [Server thread/INFO]: Steve joined the game"))
	if got := tr.get("s1"); got.Version != "1.21.1" || len(got.Players) != 1 {
		t.Fatalf("expected version 1.21.1 and 1 player, got version=%q players=%v", got.Version, got.Players)
	}
	// Stopping clears the roster.
	tr.observe(EventPayload{ServerID: "s1", Kind: EventStateChanged, Message: "offline"})
	if got := tr.get("s1"); len(got.Players) != 0 {
		t.Fatalf("expected empty roster after stop, got %v", got.Players)
	}
}
