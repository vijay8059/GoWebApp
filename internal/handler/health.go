// Package handler contains HTTP handlers. Handlers are deliberately thin —
// business logic lives in dedicated service packages so handlers stay easy
// to test and swap behind alternate transports (gRPC, queue consumers).
package handler

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"
)

type Health struct {
	version string
	ready   atomic.Bool
	started time.Time
}

func NewHealth(version string) *Health {
	h := &Health{version: version, started: time.Now()}
	// Mark ready once dependencies (DB/Redis) have been verified. For this
	// demo we mark ready immediately; in a real service this flips after
	// successful connection pool warm-up.
	h.ready.Store(true)
	return h
}

func (h *Health) SetReady(v bool) { h.ready.Store(v) }

// Live answers liveness probes: process is up and the runtime is healthy.
func (h *Health) Live(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"version": h.version,
		"uptime":  time.Since(h.started).String(),
	})
}

// Ready answers readiness probes: dependencies are reachable and the service
// is willing to accept traffic. Returning 503 lets orchestrators drain.
func (h *Health) Ready(w http.ResponseWriter, _ *http.Request) {
	if !h.ready.Load() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"status": "not_ready"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ready"})
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}
