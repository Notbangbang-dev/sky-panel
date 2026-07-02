package httpapi

import (
	"net/http"
	"testing"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/quotasvc"
)

// meID fetches the authenticated user's id via /api/v1/me.
func meID(t *testing.T, r http.Handler, access string) string {
	t.Helper()
	rec := authedRequest(t, r, http.MethodGet, "/api/v1/me", access)
	if rec.Code != http.StatusOK {
		t.Fatalf("me: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var u userResponse
	decodeBody(t, rec, &u)
	return u.ID
}

func TestQuotaBlocksOversizedServer(t *testing.T) {
	router, _, _, ownerAccess, _ := setupServerWithFakeAgent(t)

	// The owner already has one 512 MB server. A second server that would push
	// total memory past the 2 GB default quota must be rejected.
	rec := authedJSON(t, router, http.MethodPost, "/api/v1/servers", ownerAccess, createServerRequest{
		NodeID: "does-not-matter", EggID: "does-not-matter", Name: "Too Big",
		MemoryBytes: 4096 * 1024 * 1024,
	})
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 quota_exceeded, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMyQuotaReportsUsageAndLimit(t *testing.T) {
	router, _, _, ownerAccess, _ := setupServerWithFakeAgent(t)

	rec := authedRequest(t, router, http.MethodGet, "/api/v1/me/quota", ownerAccess)
	if rec.Code != http.StatusOK {
		t.Fatalf("me/quota: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var q quotaResponse
	decodeBody(t, rec, &q)
	if q.Usage.Servers != 1 || q.Usage.MemoryBytes != 512*1024*1024 {
		t.Errorf("unexpected usage: %+v", q.Usage)
	}
	if q.Limit.MemoryBytes != quotasvc.DefaultMemoryBytes {
		t.Errorf("expected default memory limit %d, got %d", quotasvc.DefaultMemoryBytes, q.Limit.MemoryBytes)
	}
}

func TestStorePurchaseRequiresCoinsThenRaisesQuota(t *testing.T) {
	router, _ := newFullTestRouter(t)

	// First registered user is the admin; a second is a regular user.
	adminAccess := registerAndGetAccessToken(t, router, "admin@example.com", "admin")
	userAccess := registerAndGetAccessToken(t, router, "buyer@example.com", "buyer")
	userID := meID(t, router, userAccess)

	// Catalog is available.
	if rec := authedRequest(t, router, http.MethodGet, "/api/v1/store", userAccess); rec.Code != http.StatusOK {
		t.Fatalf("store list: expected 200, got %d", rec.Code)
	}

	// With no coins, a purchase is refused.
	broke := authedJSON(t, router, http.MethodPost, "/api/v1/store/purchase", userAccess, purchaseRequest{ItemID: "mem-1024"})
	if broke.Code != http.StatusConflict {
		t.Fatalf("expected 409 insufficient_balance, got %d: %s", broke.Code, broke.Body.String())
	}

	// Admin grants coins.
	adjust := authedJSON(t, router, http.MethodPost, "/api/v1/admin/users/"+userID+"/coins/adjust", adminAccess, adminAdjustCoinsRequest{Amount: 1000})
	if adjust.Code != http.StatusOK {
		t.Fatalf("coin adjust: expected 200, got %d: %s", adjust.Code, adjust.Body.String())
	}

	// Baseline memory limit before purchase.
	var before quotaResponse
	decodeBody(t, authedRequest(t, router, http.MethodGet, "/api/v1/me/quota", userAccess), &before)

	// Now the purchase succeeds and raises the memory quota by 1 GB.
	buy := authedJSON(t, router, http.MethodPost, "/api/v1/store/purchase", userAccess, purchaseRequest{ItemID: "mem-1024"})
	if buy.Code != http.StatusOK {
		t.Fatalf("purchase: expected 200, got %d: %s", buy.Code, buy.Body.String())
	}
	var pr purchaseResponse
	decodeBody(t, buy, &pr)
	if pr.Balance != 1000-450 {
		t.Errorf("expected balance %d after buying mem-1024, got %d", 1000-450, pr.Balance)
	}

	var after quotaResponse
	decodeBody(t, authedRequest(t, router, http.MethodGet, "/api/v1/me/quota", userAccess), &after)
	if after.Limit.MemoryBytes != before.Limit.MemoryBytes+1024*1024*1024 {
		t.Errorf("expected memory limit to rise by 1 GB, before=%d after=%d", before.Limit.MemoryBytes, after.Limit.MemoryBytes)
	}
}
