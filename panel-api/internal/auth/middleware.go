package auth

import (
	"context"
	"net/http"
	"strings"
)

type ctxKey int

const claimsCtxKey ctxKey = iota

// ClaimsResolver resolves a raw non-JWT bearer token (e.g. a personal API key)
// to Claims. It returns ok=false when the token isn't a valid key.
type ClaimsResolver func(rawToken string) (*Claims, bool)

// RequireAuth parses the Bearer access token, validates it, and stores the
// resulting Claims on the request context for downstream handlers. If the
// token isn't a valid JWT and a resolver is supplied, it's given a chance to
// resolve the token as a personal API key.
func RequireAuth(mgr *Manager, resolve ClaimsResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := bearerToken(r)
			if raw == "" {
				http.Error(w, "missing bearer token", http.StatusUnauthorized)
				return
			}

			claims, err := mgr.ParseAccessToken(raw)
			if err != nil {
				if resolve != nil {
					if c, ok := resolve(raw); ok {
						ctx := context.WithValue(r.Context(), claimsCtxKey, c)
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}
				}
				http.Error(w, "invalid or expired token", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), claimsCtxKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole must be chained after RequireAuth. It rejects requests whose
// authenticated user does not hold one of the allowed roles.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := FromContext(r.Context())
			if !ok || !allowed[claims.Role] {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func FromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsCtxKey).(*Claims)
	return claims, ok
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if strings.HasPrefix(h, prefix) {
		return strings.TrimPrefix(h, prefix)
	}
	// Fallback for WebSocket upgrades, which can't set custom headers from
	// browser JS: allow the access token as a query parameter.
	return r.URL.Query().Get("access_token")
}
