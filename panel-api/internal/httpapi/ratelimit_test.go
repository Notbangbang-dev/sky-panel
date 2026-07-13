package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestKeyedLimiterBurstThenBlock(t *testing.T) {
	// 60/min = 1/s refill, burst 2: first two immediate calls pass, third fails.
	kl := newKeyedLimiter(true, 60, 2)
	if !kl.allow("a") || !kl.allow("a") {
		t.Fatal("expected the first two calls within burst to be allowed")
	}
	if kl.allow("a") {
		t.Fatal("expected the third call to be rate-limited")
	}
	// A different key has its own bucket.
	if !kl.allow("b") {
		t.Fatal("expected a distinct key to be allowed independently")
	}
}

func TestKeyedLimiterDisabledAlwaysAllows(t *testing.T) {
	kl := newKeyedLimiter(false, 1, 1)
	for i := 0; i < 100; i++ {
		if !kl.allow("x") {
			t.Fatalf("disabled limiter blocked call %d", i)
		}
	}
}

func TestRateLimitMiddlewareReturns429(t *testing.T) {
	kl := newKeyedLimiter(true, 60, 1)
	mw := rateLimitMiddleware(kl, clientIP)
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	fire := func() int {
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.RemoteAddr = "203.0.113.5:1111"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}

	if code := fire(); code != http.StatusOK {
		t.Fatalf("first request: want 200, got %d", code)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.RemoteAddr = "203.0.113.5:2222"
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("second request from same IP: want 429, got %d", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Error("expected a Retry-After header on the 429 response")
	}
}

func TestClientIPStripsPort(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "198.51.100.7:54321"
	if got := clientIP(req); got != "198.51.100.7" {
		t.Fatalf("clientIP: want 198.51.100.7, got %q", got)
	}
	// Bare IP (as chi RealIP may leave it) must pass through unchanged.
	req.RemoteAddr = "198.51.100.7"
	if got := clientIP(req); got != "198.51.100.7" {
		t.Fatalf("clientIP bare: want 198.51.100.7, got %q", got)
	}
}
