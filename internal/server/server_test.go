package server

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GoWebApp/internal/config"
)

func TestServer_RoutesWired(t *testing.T) {
	cfg := config.Config{
		HTTPAddr:       ":0",
		RateLimitRPS:   100,
		RateLimitBurst: 100,
	}
	srv := New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))

	cases := []struct {
		path string
		want int
	}{
		{"/healthz", http.StatusOK},
		{"/readyz", http.StatusOK},
		{"/api/v1/hello", http.StatusOK},
		{"/api/v1/whoami", http.StatusUnauthorized}, // no auth header
		{"/does/not/exist", http.StatusNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			srv.Handler.ServeHTTP(rr, req)
			if rr.Code != tc.want {
				t.Fatalf("%s: status = %d, want %d", tc.path, rr.Code, tc.want)
			}
		})
	}
}
