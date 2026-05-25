// Package server composes the HTTP router, middleware chain, and *http.Server.
package server

import (
	"log/slog"
	"net/http"

	"github.com/GoWebApp/internal/config"
	"github.com/GoWebApp/internal/handler"
	"github.com/GoWebApp/internal/middleware"
	"github.com/gorilla/mux"
)

// New returns a fully wired *http.Server.
func New(cfg config.Config, log *slog.Logger) *http.Server {
	r := mux.NewRouter()

	// Liveness and readiness are unauthenticated and never rate-limited so
	// orchestrators can probe even when the app is under load.
	health := handler.NewHealth(version())
	r.HandleFunc("/healthz", health.Live).Methods(http.MethodGet)
	r.HandleFunc("/readyz", health.Ready).Methods(http.MethodGet)

	// Application routes
	api := r.PathPrefix("/api/v1").Subrouter()
	api.Use(middleware.RateLimit(cfg.RateLimitRPS, cfg.RateLimitBurst))
	api.HandleFunc("/hello", handler.Hello).Methods(http.MethodGet)
	api.Handle("/whoami", middleware.RequireAuth(http.HandlerFunc(handler.WhoAmI))).Methods(http.MethodGet)

	// Global middleware (applied outside-in: recovery -> logging -> handler)
	root := middleware.Recover(log)(middleware.RequestLogger(log)(r))

	return &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      root,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}
}

// version is overridden in tests; in production it's read from the build-time
// ldflag on the main package via runtime/debug.BuildInfo.
func version() string {
	return "dev"
}
