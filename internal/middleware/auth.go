package middleware

import (
	"context"
	"crypto/subtle"
	"net/http"
	"os"
	"strings"
)

type ctxKey int

const userCtxKey ctxKey = iota

// RequireAuth enforces a Bearer token on the request. For a portfolio demo
// the expected token comes from the API_TOKEN env var; in production this
// would validate a JWT or call an upstream auth service.
func RequireAuth(next http.Handler) http.Handler {
	expected := os.Getenv("API_TOKEN")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authz := r.Header.Get("Authorization")
		token := strings.TrimPrefix(authz, "Bearer ")
		if expected == "" || token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(expected)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), userCtxKey, "token-user")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserFromContext returns the authenticated user identifier, if any.
func UserFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(userCtxKey).(string)
	return v, ok
}
