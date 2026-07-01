package httpapi

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/agenthub"
)

// helloIsAccepted dials the agent ws endpoint, sends hello with token, and
// reports whether the server kept the connection open (accepted) rather than
// closing it immediately (rejected — bad/expired token).
func helloIsAccepted(t *testing.T, ts *httptest.Server, token string) bool {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/agent/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial agent ws: %v", err)
	}
	defer conn.Close()

	env, err := agenthub.EncodeSigned([]byte(token), agenthub.TypeHello, agenthub.HelloPayload{NodeToken: token, AgentVersion: "test"})
	if err != nil {
		t.Fatalf("encode hello: %v", err)
	}
	if err := conn.WriteJSON(env); err != nil {
		t.Fatalf("write hello: %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	var in agenthub.Envelope
	err = conn.ReadJSON(&in)
	// A read timeout means the server kept the connection open with nothing
	// to send yet — i.e. hello was accepted. Any other error (EOF, close
	// frame) means the server closed it, i.e. hello was rejected.
	netErr, isTimeout := err.(net.Error)
	return isTimeout && netErr.Timeout()
}

func TestRotateNodeTokenIssuesFreshCredential(t *testing.T) {
	router, _ := newFullTestRouter(t)
	ts := httptest.NewServer(router)
	defer ts.Close()

	adminAccess := registerAndGetAccessToken(t, router, "admin@example.com", "admin")

	createRec := authedJSON(t, router, http.MethodPost, "/api/v1/admin/nodes", adminAccess, createNodeRequest{
		Name: "node-1", Address: "127.0.0.1",
	})
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create node: expected 201, got %d: %s", createRec.Code, createRec.Body.String())
	}
	var created createNodeResponse
	decodeBody(t, createRec, &created)

	rotateRec := authedRequest(t, router, http.MethodPost, "/api/v1/admin/nodes/"+created.ID+"/rotate-token", adminAccess)
	if rotateRec.Code != http.StatusOK {
		t.Fatalf("rotate token: expected 200, got %d: %s", rotateRec.Code, rotateRec.Body.String())
	}
	var rotated map[string]string
	decodeBody(t, rotateRec, &rotated)
	if rotated["node_token"] == "" || rotated["node_token"] == created.NodeToken {
		t.Fatalf("expected a fresh, different node_token, got %q (old was %q)", rotated["node_token"], created.NodeToken)
	}

	// The old token must no longer authenticate a fresh hello, while the new
	// one does.
	if helloIsAccepted(t, ts, created.NodeToken) {
		t.Errorf("expected the rotated-out token to be rejected")
	}
	if !helloIsAccepted(t, ts, rotated["node_token"]) {
		t.Errorf("expected the freshly rotated token to be accepted")
	}
}

func TestListNodesSlimReflectsConnectionState(t *testing.T) {
	router, _ := newFullTestRouter(t)
	ts := httptest.NewServer(router)
	defer ts.Close()

	adminAccess := registerAndGetAccessToken(t, router, "admin@example.com", "admin")
	userAccess := registerAndGetAccessToken(t, router, "user@example.com", "regularuser")

	createRec := authedJSON(t, router, http.MethodPost, "/api/v1/admin/nodes", adminAccess, createNodeRequest{
		Name: "node-1", Address: "10.0.0.5",
	})
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create node: expected 201, got %d: %s", createRec.Code, createRec.Body.String())
	}
	var created createNodeResponse
	decodeBody(t, createRec, &created)

	listRec := authedRequest(t, router, http.MethodGet, "/api/v1/nodes", userAccess)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list nodes: expected 200, got %d: %s", listRec.Code, listRec.Body.String())
	}
	var summaries []nodeSummary
	decodeBody(t, listRec, &summaries)
	if len(summaries) != 1 || summaries[0].ID != created.ID || summaries[0].Connected {
		t.Fatalf("expected one disconnected node before the agent dials in, got %+v", summaries)
	}
	if summaries[0].Name != "node-1" || summaries[0].Address != "10.0.0.5" {
		t.Errorf("unexpected node summary fields: %+v", summaries[0])
	}

	fakeAgent := connectFakeNodeAgent(t, ts, created.NodeToken)
	defer fakeAgent.Close()
	waitForNodeConnected(t, router, adminAccess, created.ID)

	listRec = authedRequest(t, router, http.MethodGet, "/api/v1/nodes", userAccess)
	decodeBody(t, listRec, &summaries)
	if len(summaries) != 1 || !summaries[0].Connected {
		t.Fatalf("expected the node to show connected once the agent dialed in, got %+v", summaries)
	}
}

func TestRotateNodeTokenUnknownNode(t *testing.T) {
	router, _ := newFullTestRouter(t)
	adminAccess := registerAndGetAccessToken(t, router, "admin@example.com", "admin")

	rec := authedRequest(t, router, http.MethodPost, "/api/v1/admin/nodes/does-not-exist/rotate-token", adminAccess)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for an unknown node, got %d", rec.Code)
	}
}
