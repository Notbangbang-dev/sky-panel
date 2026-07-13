package httpapi

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/auth"
)

// keyedLimiter holds an independent token-bucket rate limiter per key (a client
// IP or an authenticated user id), evicting buckets that have gone idle so the
// map can't grow without bound under a spray of distinct keys.
//
// When enabled is false every call is allowed — this lets the test suite build
// routers without tripping limits, while production wires the limiters live.
type keyedLimiter struct {
	enabled bool
	limit   rate.Limit
	burst   int
	idleTTL time.Duration

	mu      sync.Mutex
	buckets map[string]*bucketEntry
}

type bucketEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// newKeyedLimiter builds a limiter allowing roughly perMinute requests per key
// with the given burst allowance.
func newKeyedLimiter(enabled bool, perMinute float64, burst int) *keyedLimiter {
	return &keyedLimiter{
		enabled: enabled,
		limit:   rate.Limit(perMinute / 60.0),
		burst:   burst,
		idleTTL: 15 * time.Minute,
		buckets: make(map[string]*bucketEntry),
	}
}

func (kl *keyedLimiter) allow(key string) bool {
	if !kl.enabled {
		return true
	}

	kl.mu.Lock()
	defer kl.mu.Unlock()

	now := time.Now()
	e, ok := kl.buckets[key]
	if !ok {
		e = &bucketEntry{limiter: rate.NewLimiter(kl.limit, kl.burst)}
		kl.buckets[key] = e
	}
	e.lastSeen = now

	// Opportunistic eviction once the map has grown past a modest size, so a
	// long-running instance seeing many distinct IPs doesn't leak memory.
	if len(kl.buckets) > 2048 {
		for k, be := range kl.buckets {
			if now.Sub(be.lastSeen) > kl.idleTTL {
				delete(kl.buckets, k)
			}
		}
	}

	return e.limiter.Allow()
}

// clientIP extracts the caller's IP. chi's RealIP middleware has already
// resolved X-Forwarded-For / X-Real-IP into RemoteAddr where present; that form
// may be a bare IP (no port), so fall back to the raw value if it won't split.
func clientIP(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// userOrIPKey keys on the authenticated user id when present (so a limit
// follows the account regardless of source IP), falling back to the client IP
// for unauthenticated callers.
func userOrIPKey(r *http.Request) string {
	if claims, ok := auth.FromContext(r.Context()); ok && claims.UserID != "" {
		return "u:" + claims.UserID
	}
	return "ip:" + clientIP(r)
}

func rateLimitMiddleware(kl *keyedLimiter, keyFn func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !kl.allow(keyFn(r)) {
				w.Header().Set("Retry-After", "60")
				writeError(w, http.StatusTooManyRequests, "rate_limited", "too many requests — please slow down and try again shortly")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// rateLimiters bundles every named limiter used by the router. Limits are
// expressed as requests-per-minute with a burst allowance; they're generous
// enough for legitimate interactive use but choke scripted abuse (credential
// stuffing, redeem-code guessing, create/backup floods, coin farming).
type rateLimiters struct {
	authIP   *keyedLimiter // per-IP: login / register / refresh
	globalIP *keyedLimiter // per-IP: flood backstop across the whole API
	economy  *keyedLimiter // per-user: gift / daily reward
	redeem   *keyedLimiter // per-user: redeem code (brute-force guard)
	purchase *keyedLimiter // per-user: store purchases
	afk      *keyedLimiter // per-user: AFK heartbeat (frequent, looser)
	create   *keyedLimiter // per-user: server create / clone / reinstall
	backups  *keyedLimiter // per-user: backup create
	files    *keyedLimiter // per-user: file writes / mutations
	database *keyedLimiter // per-user: database create
}

func newRateLimiters(enabled bool) *rateLimiters {
	return &rateLimiters{
		authIP:   newKeyedLimiter(enabled, 40, 15),
		globalIP: newKeyedLimiter(enabled, 1200, 300),
		economy:  newKeyedLimiter(enabled, 30, 10),
		redeem:   newKeyedLimiter(enabled, 20, 8),
		purchase: newKeyedLimiter(enabled, 40, 12),
		afk:      newKeyedLimiter(enabled, 120, 30),
		create:   newKeyedLimiter(enabled, 20, 6),
		backups:  newKeyedLimiter(enabled, 15, 4),
		files:    newKeyedLimiter(enabled, 180, 60),
		database: newKeyedLimiter(enabled, 15, 5),
	}
}

// byIP / byUser return ready-to-use chi middlewares for the given limiter.
func (rl *rateLimiters) byIP(kl *keyedLimiter) func(http.Handler) http.Handler {
	return rateLimitMiddleware(kl, func(r *http.Request) string { return clientIP(r) })
}

func (rl *rateLimiters) byUser(kl *keyedLimiter) func(http.Handler) http.Handler {
	return rateLimitMiddleware(kl, userOrIPKey)
}
