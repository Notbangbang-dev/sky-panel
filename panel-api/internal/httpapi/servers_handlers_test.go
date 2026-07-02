package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/agenthub"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/auth"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/coinsvc"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/quotasvc"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/repo"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/serversvc"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/store"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/storesvc"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/wshub"
)

// newFullTestRouter wires every domain (not just auth) so servers/eggs/nodes
// endpoints can be exercised, including real dispatch through agenthub to a
// simulated node-agent connected over its own WebSocket. It also returns the
// Allocations repo directly, since there's no HTTP endpoint for seeding port
// allocations yet (that's operator-side provisioning, out of scope here).
func newFullTestRouter(t *testing.T) (http.Handler, *repo.Allocations) {
	t.Helper()

	name := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	dsn := "file:" + name + "?mode=memory&cache=shared"

	db, err := store.Open(dsn)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	users := repo.NewUsers(db)
	nodes := repo.NewNodes(db)
	eggs := repo.NewEggs(db)
	servers := repo.NewServers(db)
	allocations := repo.NewAllocations(db)
	subusers := repo.NewSubusers(db)
	ledger := repo.NewLedger(db)
	afk := repo.NewAFKState(db)
	dailyRewards := repo.NewDailyRewards(db)
	quotas := repo.NewQuotas(db)
	settings := repo.NewSettings(db)
	hub := wshub.NewHub()

	registry := agenthub.NewRegistry()
	agentHandler := agenthub.NewHandler(registry, nodes, hub)

	deps := Deps{
		Users:         users,
		RefreshTokens: repo.NewRefreshTokens(db),
		JWT:           auth.NewManager("test-secret", 15*time.Minute),
		Hub:           hub,
		Nodes:         nodes,
		Eggs:          eggs,
		Servers:       servers,
		Allocations:   allocations,
		Subusers:      subusers,
		Quotas:        quotas,
		ServerSvc:     serversvc.NewService(servers, eggs, nodes, allocations, registry),
		AgentHub:      agentHandler,
		CoinSvc:       coinsvc.NewService(users, ledger, afk, dailyRewards),
		QuotaSvc:      quotasvc.NewService(servers, quotas, settings),
		StoreSvc:      storesvc.NewService(ledger, quotas),
		Settings:      settings,
		Audit:         repo.NewAudit(db),
		RefreshTTL:    30 * 24 * time.Hour,
	}

	return NewRouter(deps), allocations
}

// connectFakeNodeAgent dials srv's /agent/ws endpoint pretending to be a
// node-agent, sends hello, and auto-acks every command it receives with OK.
func connectFakeNodeAgent(t *testing.T, srv *httptest.Server, nodeToken string) *websocket.Conn {
	t.Helper()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/agent/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial agent ws: %v", err)
	}

	env, err := agenthub.EncodeSigned([]byte(nodeToken), agenthub.TypeHello, agenthub.HelloPayload{NodeToken: nodeToken, AgentVersion: "test"})
	if err != nil {
		t.Fatalf("encode hello: %v", err)
	}
	if err := conn.WriteJSON(env); err != nil {
		t.Fatalf("write hello: %v", err)
	}

	go func() {
		for {
			var in agenthub.Envelope
			if err := conn.ReadJSON(&in); err != nil {
				return
			}
			if in.Type != agenthub.TypeCommand {
				continue
			}
			var cmd agenthub.CommandPayload
			if json.Unmarshal(in.Payload, &cmd) != nil {
				continue
			}
			ack := agenthub.AckPayload{CommandID: cmd.CommandID, OK: true}
			switch cmd.Action {
			case agenthub.ActionListFiles:
				result, _ := json.Marshal(agenthub.ListFilesResult{Entries: []agenthub.FileEntry{
					{Name: "server.properties", IsDir: false, SizeBytes: 42},
				}})
				ack.Result = result
			case agenthub.ActionReadFile:
				result, _ := json.Marshal(agenthub.ReadFileResult{ContentBase64: "ZmFrZS1jb250ZW50cw=="})
				ack.Result = result
			case agenthub.ActionBackup:
				result, _ := json.Marshal(agenthub.BackupResult{Filename: "20260701-120000.tar.zst", SizeBytes: 1024})
				ack.Result = result
			case agenthub.ActionListBackups:
				result, _ := json.Marshal(agenthub.ListBackupsResult{Backups: []agenthub.BackupEntry{
					{Filename: "backup-1782000000.tar.zst", SizeBytes: 1024, CreatedAt: 1782000000},
				}})
				ack.Result = result
			}
			ackEnv, err := agenthub.EncodeSigned([]byte(nodeToken), agenthub.TypeAck, ack)
			if err != nil {
				continue
			}
			if conn.WriteJSON(ackEnv) != nil {
				return
			}
		}
	}()

	return conn
}

func TestServerLifecycleEndToEndOverHTTP(t *testing.T) {
	router, allocations := newFullTestRouter(t)
	srv := httptest.NewServer(router)
	defer srv.Close()

	// Admin bootstrap (first registered user becomes admin).
	adminAccess := registerAndGetAccessToken(t, router, "admin@example.com", "admin")

	createNodeRec := authedJSON(t, router, http.MethodPost, "/api/v1/admin/nodes", adminAccess, createNodeRequest{
		Name: "node-1", Address: "127.0.0.1",
	})
	if createNodeRec.Code != http.StatusCreated {
		t.Fatalf("create node: expected 201, got %d: %s", createNodeRec.Code, createNodeRec.Body.String())
	}
	var node createNodeResponse
	decodeBody(t, createNodeRec, &node)

	// Connect the fake node-agent using the freshly issued token.
	fakeAgent := connectFakeNodeAgent(t, srv, node.NodeToken)
	defer fakeAgent.Close()

	createEggRec := authedJSON(t, router, http.MethodPost, "/api/v1/admin/eggs", adminAccess, createEggRequest{
		Name:        "Minecraft",
		DockerImage: "itzg/minecraft-server",
		Startup:     "java -jar server.jar --port {{SERVER_PORT}}",
	})
	if createEggRec.Code != http.StatusCreated {
		t.Fatalf("create egg: expected 201, got %d: %s", createEggRec.Code, createEggRec.Body.String())
	}
	var egg struct {
		ID string `json:"id"`
	}
	decodeBody(t, createEggRec, &egg)

	waitForNodeConnected(t, router, adminAccess, node.ID)

	// A free port allocation must exist before a server can be created.
	if err := allocations.Create("alloc-1", node.ID, 25565); err != nil {
		t.Fatalf("seed allocation: %v", err)
	}

	// Regular user registers and creates a server on the node.
	userAccess := registerAndGetAccessToken(t, router, "user@example.com", "regularuser")

	createServerRec := authedJSON(t, router, http.MethodPost, "/api/v1/servers", userAccess, createServerRequest{
		NodeID: node.ID, EggID: egg.ID, Name: "My Server", MemoryBytes: 512 * 1024 * 1024,
	})
	if createServerRec.Code != http.StatusCreated {
		t.Fatalf("create server: expected 201, got %d: %s", createServerRec.Code, createServerRec.Body.String())
	}

	var server serverResponse
	decodeBody(t, createServerRec, &server)
	if server.Status != "running" {
		t.Errorf("expected server status running after create, got %q", server.Status)
	}

	// Another user must not be able to see or control it.
	otherAccess := registerAndGetAccessToken(t, router, "other@example.com", "otheruser")
	getRec := authedRequest(t, router, http.MethodGet, "/api/v1/servers/"+server.ID, otherAccess)
	if getRec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for a non-owner accessing the server, got %d", getRec.Code)
	}

	// The owner can stop it.
	stopRec := authedJSON(t, router, http.MethodPost, "/api/v1/servers/"+server.ID+"/power", userAccess, powerActionRequest{Action: "stop"})
	if stopRec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 from power stop, got %d: %s", stopRec.Code, stopRec.Body.String())
	}
}

func authedJSON(t *testing.T, r http.Handler, method, path, accessToken string, body any) *httptest.ResponseRecorder {
	t.Helper()
	req := jsonRequest(t, method, path, body)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func authedRequest(t *testing.T, r http.Handler, method, path, accessToken string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

// waitForNodeConnected polls until the admin nodes list is non-empty, purely
// to give the background fake-agent goroutine time to complete its hello
// handshake before the test proceeds.
func waitForNodeConnected(t *testing.T, r http.Handler, adminAccess, nodeID string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		rec := authedRequest(t, r, http.MethodGet, "/api/v1/admin/nodes", adminAccess)
		if rec.Code == http.StatusOK && strings.Contains(rec.Body.String(), nodeID) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}
