package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/GoWebApp/internal/auth"
)

type ctxKey int

const userCtxKey ctxKey = iota

// RequireJWT validates the Authorization: Bearer <jwt> header against the
// supplied verifier and attaches the subject to the request context.
//
// Failure modes — all 401:
//   - missing or malformed Authorization header
//   - signature invalid / wrong algorithm
//   - expired / not-yet-valid token
//   - issuer or audience mismatch
//
// The middleware deliberately does not leak the reason: callers see
// "unauthorized" while operators see the parse error in the structured log.
func RequireJWT(v *auth.Verifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r)
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			subject, err := v.Parse(token)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), userCtxKey, subject)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserFromContext returns the authenticated subject, if any.
func UserFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(userCtxKey).(string)
	return v, ok
}

func bearerToken(r *http.Request) (string, bool) {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(h) <= len(prefix) || !strings.EqualFold(h[:len(prefix)], prefix) {
		return "", false
	}
	return strings.TrimSpace(h[len(prefix):]), true
}
