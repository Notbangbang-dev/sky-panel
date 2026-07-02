package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/agenthub"
)

// setupServerWithFakeAgent creates a node + egg + running server, all owned
// by userAccess, with a fake node-agent connected so file/console/power
// dispatch has somewhere to land.
func setupServerWithFakeAgent(t *testing.T) (router http.Handler, srvURL string, adminAccess, ownerAccess string, server serverResponse) {
	t.Helper()
	r, _ := newFullTestRouter(t)
	// httptest.NewServer wraps r so the fake agent can dial a real ws URL.
	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)

	adminAccess = registerAndGetAccessToken(t, r, "admin@example.com", "admin")

	createNodeRec := authedJSON(t, r, http.MethodPost, "/api/v1/admin/nodes", adminAccess, createNodeRequest{
		Name: "node-1", Address: "127.0.0.1",
	})
	if createNodeRec.Code != http.StatusCreated {
		t.Fatalf("create node: expected 201, got %d: %s", createNodeRec.Code, createNodeRec.Body.String())
	}
	var node createNodeResponse
	decodeBody(t, createNodeRec, &node)

	fakeAgent := connectFakeNodeAgent(t, ts, node.NodeToken)
	t.Cleanup(func() { fakeAgent.Close() })

	createEggRec := authedJSON(t, r, http.MethodPost, "/api/v1/admin/eggs", adminAccess, createEggRequest{
		Name: "Minecraft", DockerImage: "itzg/minecraft-server", Startup: "java -jar server.jar",
	})
	if createEggRec.Code != http.StatusCreated {
		t.Fatalf("create egg: expected 201, got %d: %s", createEggRec.Code, createEggRec.Body.String())
	}
	var egg struct {
		ID string `json:"id"`
	}
	decodeBody(t, createEggRec, &egg)

	waitForNodeConnected(t, r, adminAccess, node.ID)

	// Node creation auto-seeds default port allocations; no manual seed needed.

	ownerAccess = registerAndGetAccessToken(t, r, "owner@example.com", "serverowner")
	createServerRec := authedJSON(t, r, http.MethodPost, "/api/v1/servers", ownerAccess, createServerRequest{
		NodeID: node.ID, EggID: egg.ID, Name: "My Server", MemoryBytes: 512 * 1024 * 1024,
	})
	if createServerRec.Code != http.StatusCreated {
		t.Fatalf("create server: expected 201, got %d: %s", createServerRec.Code, createServerRec.Body.String())
	}
	decodeBody(t, createServerRec, &server)

	return r, ts.URL, adminAccess, ownerAccess, server
}

func TestSubuserPermissionsGateServerEndpoints(t *testing.T) {
	router, _, _, ownerAccess, server := setupServerWithFakeAgent(t)

	helperAccess := registerAndGetAccessToken(t, router, "helper@example.com", "helperuser")

	// Before being added, the helper has no access at all.
	if rec := authedRequest(t, router, http.MethodGet, "/api/v1/servers/"+server.ID, helperAccess); rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 before being added as a subuser, got %d", rec.Code)
	}

	// Owner grants console-only access.
	addRec := authedJSON(t, router, http.MethodPost, "/api/v1/servers/"+server.ID+"/subusers", ownerAccess, addSubuserRequest{
		Username: "helperuser", Permissions: []string{"console"},
	})
	if addRec.Code != http.StatusCreated {
		t.Fatalf("add subuser: expected 201, got %d: %s", addRec.Code, addRec.Body.String())
	}

	listRec := authedRequest(t, router, http.MethodGet, "/api/v1/servers/"+server.ID+"/subusers", ownerAccess)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list subusers: expected 200, got %d", listRec.Code)
	}
	var subusers []subuserResponse
	decodeBody(t, listRec, &subusers)
	if len(subusers) != 1 || subusers[0].Permissions[0] != "console" {
		t.Fatalf("unexpected subuser list: %+v", subusers)
	}

	// Console access now works for the helper...
	consoleRec := authedJSON(t, router, http.MethodPost, "/api/v1/servers/"+server.ID+"/console", helperAccess, consoleInputRequest{Input: "say hi"})
	if consoleRec.Code != http.StatusNoContent {
		t.Fatalf("console input: expected 204, got %d: %s", consoleRec.Code, consoleRec.Body.String())
	}

	// ...but power actions still are not, since that permission wasn't granted.
	powerRec := authedJSON(t, router, http.MethodPost, "/api/v1/servers/"+server.ID+"/power", helperAccess, powerActionRequest{Action: "stop"})
	if powerRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for power action without the power permission, got %d", powerRec.Code)
	}

	// Owner revokes access entirely.
	removeRec := authedRequest(t, router, http.MethodDelete, "/api/v1/servers/"+server.ID+"/subusers/"+subusers[0].UserID, ownerAccess)
	if removeRec.Code != http.StatusNoContent {
		t.Fatalf("remove subuser: expected 204, got %d", removeRec.Code)
	}
	if rec := authedJSON(t, router, http.MethodPost, "/api/v1/servers/"+server.ID+"/console", helperAccess, consoleInputRequest{Input: "say hi"}); rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for console input after subuser removal, got %d", rec.Code)
	}
}

func TestSubuserManagementIsNotDelegable(t *testing.T) {
	router, _, _, ownerAccess, server := setupServerWithFakeAgent(t)

	helperAccess := registerAndGetAccessToken(t, router, "helper2@example.com", "helperuser2")
	addRec := authedJSON(t, router, http.MethodPost, "/api/v1/servers/"+server.ID+"/subusers", ownerAccess, addSubuserRequest{
		Username: "helperuser2", Permissions: []string{"console", "files", "power", "settings"},
	})
	if addRec.Code != http.StatusCreated {
		t.Fatalf("add subuser: expected 201, got %d: %s", addRec.Code, addRec.Body.String())
	}

	// Even with every permission, a subuser cannot manage other subusers.
	rec := authedJSON(t, router, http.MethodPost, "/api/v1/servers/"+server.ID+"/subusers", helperAccess, addSubuserRequest{
		Username: "helperuser2", Permissions: []string{"console"},
	})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for a subuser trying to manage subusers, got %d", rec.Code)
	}
}

func TestFileManagerEndpointsDispatchThroughAgent(t *testing.T) {
	router, _, _, ownerAccess, server := setupServerWithFakeAgent(t)

	listRec := authedRequest(t, router, http.MethodGet, "/api/v1/servers/"+server.ID+"/files?path=", ownerAccess)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list files: expected 200, got %d: %s", listRec.Code, listRec.Body.String())
	}
	var listResult agenthub.ListFilesResult
	decodeBody(t, listRec, &listResult)
	if len(listResult.Entries) != 1 || listResult.Entries[0].Name != "server.properties" {
		t.Fatalf("unexpected file listing: %+v", listResult)
	}

	readRec := authedRequest(t, router, http.MethodGet, "/api/v1/servers/"+server.ID+"/files/content?path=server.properties", ownerAccess)
	if readRec.Code != http.StatusOK {
		t.Fatalf("read file: expected 200, got %d: %s", readRec.Code, readRec.Body.String())
	}
	var readResult agenthub.ReadFileResult
	decodeBody(t, readRec, &readResult)
	if readResult.ContentBase64 == "" {
		t.Fatalf("expected non-empty content_base64")
	}

	writeRec := authedJSON(t, router, http.MethodPut, "/api/v1/servers/"+server.ID+"/files/content", ownerAccess, writeFileRequest{
		Path: "server.properties", ContentBase64: "bmV3LWNvbnRlbnRz",
	})
	if writeRec.Code != http.StatusNoContent {
		t.Fatalf("write file: expected 204, got %d: %s", writeRec.Code, writeRec.Body.String())
	}

	mkdirRec := authedJSON(t, router, http.MethodPost, "/api/v1/servers/"+server.ID+"/files/mkdir", ownerAccess, mkdirRequest{Path: "plugins"})
	if mkdirRec.Code != http.StatusNoContent {
		t.Fatalf("mkdir: expected 204, got %d: %s", mkdirRec.Code, mkdirRec.Body.String())
	}

	renameRec := authedJSON(t, router, http.MethodPost, "/api/v1/servers/"+server.ID+"/files/rename", ownerAccess, renameFileRequest{
		Path: "server.properties", NewPath: "server.properties.bak",
	})
	if renameRec.Code != http.StatusNoContent {
		t.Fatalf("rename: expected 204, got %d: %s", renameRec.Code, renameRec.Body.String())
	}

	deleteRec := authedRequest(t, router, http.MethodDelete, "/api/v1/servers/"+server.ID+"/files?path=server.properties.bak", ownerAccess)
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("delete: expected 204, got %d: %s", deleteRec.Code, deleteRec.Body.String())
	}
}

func TestFileManagerRequiresFilesPermission(t *testing.T) {
	router, _, _, ownerAccess, server := setupServerWithFakeAgent(t)

	helperAccess := registerAndGetAccessToken(t, router, "helper3@example.com", "helperuser3")
	addRec := authedJSON(t, router, http.MethodPost, "/api/v1/servers/"+server.ID+"/subusers", ownerAccess, addSubuserRequest{
		Username: "helperuser3", Permissions: []string{"console"},
	})
	if addRec.Code != http.StatusCreated {
		t.Fatalf("add subuser: expected 201, got %d: %s", addRec.Code, addRec.Body.String())
	}

	rec := authedRequest(t, router, http.MethodGet, "/api/v1/servers/"+server.ID+"/files?path=", helperAccess)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for files access without the files permission, got %d", rec.Code)
	}
}
