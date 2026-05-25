// Package server composes the HTTP router, middleware chain, and *http.Server.
package server

import (
	"log/slog"
	"net/http"

	"github.com/GoWebApp/internal/auth"
	"github.com/GoWebApp/internal/config"
	"github.com/GoWebApp/internal/handler"
	"github.com/GoWebApp/internal/middleware"
	"github.com/gorilla/mux"
)

// New returns a fully wired *http.Server.
//
// If cfg.JWTSecret is set, /api/v1/whoami requires a valid JWT. When it's
// unset (typical for first-boot dev), the route still mounts but always
// 401s — this is intentional: a misconfigured prod must fail closed.
func New(cfg config.Config, log *slog.Logger) (*http.Server, error) {
	r := mux.NewRouter()

	health := handler.NewHealth(version())
	r.HandleFunc("/healthz", health.Live).Methods(http.MethodGet)
	r.HandleFunc("/readyz", health.Ready).Methods(http.MethodGet)

	api := r.PathPrefix("/api/v1").Subrouter()
	api.Use(middleware.RateLimit(cfg.RateLimitRPS, cfg.RateLimitBurst))
	api.HandleFunc("/hello", handler.Hello).Methods(http.MethodGet)

	protected := api.PathPrefix("").Subrouter()
	if cfg.JWTSecret != "" {
		verifier, err := auth.New(auth.Config{
			Secret:   cfg.JWTSecret,
			Issuer:   cfg.JWTIssuer,
			Audience: cfg.JWTAudience,
			TTL:      cfg.JWTTTL,
		})
		if err != nil {
			return nil, err
		}
		protected.Use(middleware.RequireJWT(verifier))
	} else {
		// Fail-closed when no secret is configured: every protected request
		// returns 401 instead of silently letting traffic through.
		protected.Use(denyAll)
	}
	protected.HandleFunc("/whoami", handler.WhoAmI).Methods(http.MethodGet)

	root := middleware.Recover(log)(middleware.RequestLogger(log)(r))

	return &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      root,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}, nil
}

func denyAll(http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
}

func version() string { return "dev" }
