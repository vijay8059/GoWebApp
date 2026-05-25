package server

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GoWebApp/internal/auth"
	"github.com/GoWebApp/internal/config"
)

func newTestConfig() config.Config {
	return config.Config{
		HTTPAddr:       ":0",
		RateLimitRPS:   100,
		RateLimitBurst: 100,
		JWTSecret:      "test-secret-test-secret-test-secret",
		JWTIssuer:      "gowebapp",
		JWTAudience:    "gowebapp",
		JWTTTL:         time.Minute,
	}
}

func discard() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestServer_PublicRoutes(t *testing.T) {
	srv, err := New(newTestConfig(), discard())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	cases := []struct {
		path string
		want int
	}{
		{"/healthz", http.StatusOK},
		{"/readyz", http.StatusOK},
		{"/api/v1/hello", http.StatusOK},
		{"/api/v1/whoami", http.StatusUnauthorized},
		{"/does/not/exist", http.StatusNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			rr := httptest.NewRecorder()
			srv.Handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if rr.Code != tc.want {
				t.Fatalf("%s: status = %d, want %d", tc.path, rr.Code, tc.want)
			}
		})
	}
}

func TestServer_WhoAmI_WithValidJWT(t *testing.T) {
	cfg := newTestConfig()
	srv, err := New(cfg, discard())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	v, _ := auth.New(auth.Config{
		Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, Audience: cfg.JWTAudience, TTL: cfg.JWTTTL,
	})
	tok, _ := v.Sign("alice")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/whoami", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", rr.Code, rr.Body.String())
	}
}

func TestServer_FailsClosedWithoutSecret(t *testing.T) {
	cfg := newTestConfig()
	cfg.JWTSecret = ""
	srv, err := New(cfg, discard())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Even with a token, the protected route should 401 because no verifier
	// is wired — fail closed.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/whoami", nil)
	req.Header.Set("Authorization", "Bearer anything")
	rr := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
}
