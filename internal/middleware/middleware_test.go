package middleware

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GoWebApp/internal/auth"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestVerifier(t *testing.T) *auth.Verifier {
	t.Helper()
	v, err := auth.New(auth.Config{
		Secret:   "test-secret-test-secret-test-secret",
		Issuer:   "gowebapp-test",
		Audience: "gowebapp",
		TTL:      time.Minute,
	})
	if err != nil {
		t.Fatalf("auth.New: %v", err)
	}
	return v
}

func TestRecover_TurnsPanicInto500(t *testing.T) {
	h := Recover(discardLogger())(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/x", nil))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rr.Code)
	}
}

func TestRequestLogger_PassesThrough(t *testing.T) {
	called := false
	h := RequestLogger(discardLogger())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("hi"))
	}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/x", nil))
	if !called {
		t.Fatal("inner handler not invoked")
	}
	if rr.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want 418", rr.Code)
	}
}

func TestRateLimit_BlocksAfterBurst(t *testing.T) {
	h := RateLimit(1, 2)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	statuses := make([]int, 0, 4)
	for i := 0; i < 4; i++ {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.RemoteAddr = "1.2.3.4:5678"
		h.ServeHTTP(rr, req)
		statuses = append(statuses, rr.Code)
	}
	if statuses[0] != http.StatusOK || statuses[1] != http.StatusOK {
		t.Fatalf("expected first two OK, got %v", statuses)
	}
	if statuses[2] != http.StatusTooManyRequests && statuses[3] != http.StatusTooManyRequests {
		t.Fatalf("expected at least one 429, got %v", statuses)
	}
}

func TestRequireJWT(t *testing.T) {
	v := newTestVerifier(t)
	validToken, err := v.Sign("alice")
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	var gotSubject string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSubject, _ = UserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	h := RequireJWT(v)(inner)

	cases := []struct {
		name       string
		authHeader string
		wantStatus int
		wantSub    string
	}{
		{"no header", "", http.StatusUnauthorized, ""},
		{"wrong scheme", "Basic " + validToken, http.StatusUnauthorized, ""},
		{"garbage token", "Bearer not-a-jwt", http.StatusUnauthorized, ""},
		{"valid", "Bearer " + validToken, http.StatusOK, "alice"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotSubject = ""
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			h.ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rr.Code, tc.wantStatus)
			}
			if gotSubject != tc.wantSub {
				t.Fatalf("subject = %q, want %q", gotSubject, tc.wantSub)
			}
		})
	}
}
