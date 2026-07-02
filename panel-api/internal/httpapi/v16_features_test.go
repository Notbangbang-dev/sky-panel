package httpapi

import (
	"net/http"
	"testing"
)

// Coins can be gifted between users; the sender is debited and the recipient
// credited, and the obvious abuse cases are rejected.
func TestCoinGiftTransfersBetweenUsers(t *testing.T) {
	router, _ := newFullTestRouter(t)
	admin := registerAndGetAccessToken(t, router, "admin@example.com", "admin")
	giver := registerAndGetAccessToken(t, router, "giver@example.com", "giver")
	giverID := meID(t, router, giver)
	getter := registerAndGetAccessToken(t, router, "getter@example.com", "getter")

	// Fund the giver.
	authedJSON(t, router, http.MethodPost, "/api/v1/admin/users/"+giverID+"/coins/adjust", admin, adminAdjustCoinsRequest{Amount: 100})

	// Gift 30 to "getter".
	rec := authedJSON(t, router, http.MethodPost, "/api/v1/coins/gift", giver, giftRequest{Username: "getter", Amount: 30})
	if rec.Code != http.StatusOK {
		t.Fatalf("gift: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var res coinResultResponse
	decodeBody(t, rec, &res)
	if res.Balance != 70 {
		t.Errorf("expected giver balance 70 after gifting 30, got %d", res.Balance)
	}

	// Recipient was credited exactly the gifted amount.
	getterMe := authedRequest(t, router, http.MethodGet, "/api/v1/me", getter)
	var getterUser userResponse
	decodeBody(t, getterMe, &getterUser)
	if getterUser.Coins != 30 {
		t.Errorf("expected recipient balance 30, got %d", getterUser.Coins)
	}

	// Rejections: self-gift, over-balance, unknown user.
	if self := authedJSON(t, router, http.MethodPost, "/api/v1/coins/gift", giver, giftRequest{Username: "giver", Amount: 5}); self.Code != http.StatusBadRequest {
		t.Errorf("self-gift should be 400, got %d", self.Code)
	}
	if broke := authedJSON(t, router, http.MethodPost, "/api/v1/coins/gift", giver, giftRequest{Username: "getter", Amount: 9999}); broke.Code != http.StatusBadRequest {
		t.Errorf("over-balance gift should be 400, got %d", broke.Code)
	}
	if nobody := authedJSON(t, router, http.MethodPost, "/api/v1/coins/gift", giver, giftRequest{Username: "ghost", Amount: 1}); nobody.Code != http.StatusNotFound {
		t.Errorf("gift to unknown user should be 404, got %d", nobody.Code)
	}
}

// A redeem code credits once per user and respects max_uses across users.
func TestRedeemCodeOncePerUserAndMaxUses(t *testing.T) {
	router, _ := newFullTestRouter(t)
	admin := registerAndGetAccessToken(t, router, "admin@example.com", "admin")
	a := registerAndGetAccessToken(t, router, "a@example.com", "usera")
	b := registerAndGetAccessToken(t, router, "b@example.com", "userb")
	c := registerAndGetAccessToken(t, router, "c@example.com", "userc")

	create := authedJSON(t, router, http.MethodPost, "/api/v1/admin/redeem-codes", admin, createCodeRequest{Code: "WELCOME", Coins: 50, MaxUses: 2})
	if create.Code != http.StatusCreated {
		t.Fatalf("create code: expected 201, got %d: %s", create.Code, create.Body.String())
	}

	// a redeems once (ok), twice (already used).
	if r1 := authedJSON(t, router, http.MethodPost, "/api/v1/coins/redeem", a, redeemRequest{Code: "WELCOME"}); r1.Code != http.StatusOK {
		t.Fatalf("first redeem: expected 200, got %d: %s", r1.Code, r1.Body.String())
	}
	if r2 := authedJSON(t, router, http.MethodPost, "/api/v1/coins/redeem", a, redeemRequest{Code: "WELCOME"}); r2.Code != http.StatusBadRequest {
		t.Errorf("re-redeem by same user should be 400, got %d", r2.Code)
	}
	// b redeems (uses now 2 = max).
	if rb := authedJSON(t, router, http.MethodPost, "/api/v1/coins/redeem", b, redeemRequest{Code: "WELCOME"}); rb.Code != http.StatusOK {
		t.Fatalf("second user redeem: expected 200, got %d", rb.Code)
	}
	// c is over the max_uses cap.
	if rc := authedJSON(t, router, http.MethodPost, "/api/v1/coins/redeem", c, redeemRequest{Code: "WELCOME"}); rc.Code != http.StatusBadRequest {
		t.Errorf("redeem past max_uses should be 400, got %d", rc.Code)
	}
	// Unknown code.
	if bad := authedJSON(t, router, http.MethodPost, "/api/v1/coins/redeem", c, redeemRequest{Code: "NOPE"}); bad.Code != http.StatusNotFound {
		t.Errorf("unknown code should be 404, got %d", bad.Code)
	}
}

// Impersonation is admin-only and yields a working session for the target user.
func TestAdminImpersonateIsAdminOnlyAndScoped(t *testing.T) {
	router, _ := newFullTestRouter(t)
	admin := registerAndGetAccessToken(t, router, "admin@example.com", "admin")
	target := registerAndGetAccessToken(t, router, "target@example.com", "target")
	targetID := meID(t, router, target)

	// A non-admin cannot impersonate.
	if forbidden := authedRequest(t, router, http.MethodPost, "/api/v1/admin/users/"+targetID+"/impersonate", target); forbidden.Code != http.StatusForbidden {
		t.Errorf("non-admin impersonate should be 403, got %d", forbidden.Code)
	}

	// An admin gets a token pair for the target.
	rec := authedRequest(t, router, http.MethodPost, "/api/v1/admin/users/"+targetID+"/impersonate", admin)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin impersonate: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var tp tokenPairResponse
	decodeBody(t, rec, &tp)
	if tp.User.Username != "target" {
		t.Errorf("impersonation should return the target user, got %q", tp.User.Username)
	}
	me := authedRequest(t, router, http.MethodGet, "/api/v1/me", tp.AccessToken)
	var who userResponse
	decodeBody(t, me, &who)
	if who.Username != "target" {
		t.Errorf("impersonated token should resolve to target, got %q", who.Username)
	}
}
