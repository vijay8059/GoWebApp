# Architecture

This document captures the non-obvious decisions behind the layout and
runtime. The README is the orientation; this is the rationale.

## Goals

1. **Reviewability.** A senior engineer should be able to read the whole
   repo in 30 minutes and have an accurate mental model.
2. **Operability.** The service has the seams an SRE needs: health
   probes, graceful shutdown, structured logs, bounded resource use.
3. **Substitutability.** Each layer can be replaced without rewriting the
   others: swap `mux` for `chi`, swap the in-memory limiter for Redis,
   swap MySQL for Postgres — the call sites don't move.

## Layered design

```
cmd/server   process lifecycle, signal handling, dependency wiring
   │
internal/server   router + middleware composition (no business logic)
   │
internal/handler  thin HTTP adapters; parse, delegate, encode
   │
   service / repo (to be added per feature)
   │
   infrastructure: database/sql pool, redis client, http clients
```

The middleware chain is deliberately ordered. Outer-to-inner:

1. `Recover` — must be first; a panic in any later middleware still
   produces a 500 with a logged stack instead of crashing the process.
2. `RequestLogger` — records every request including those rejected by
   later middleware (rate limit, auth), so operators can see the
   denials.
3. `RateLimit` — before auth, so we don't waste auth CPU on attackers
   hammering a stolen token.
4. `RequireAuth` — only on protected subrouters.

## Configuration

`internal/config` is the one place env vars are read. The rest of the
codebase consumes a typed `Config` struct, which means:

- A missing field is a compile error, not a runtime surprise.
- Configuration changes are visible in code review (someone has to add a
  field to the struct).
- Production-only validation (`SESSION_AUTH_KEY`, JSON logs) catches the
  "we shipped dev config to prod" class of incident at boot.

## Logging

`log/slog` is in the standard library since Go 1.21 — no third-party
dependency for structured logging. The default handler is JSON in
production; text in dev. The request logger writes one record per
request with `method`, `path`, `status`, `bytes`, `duration`, `remote`,
`user_agent`. These field names are stable and intentional: they match
the conventions most log aggregators auto-index.

## Graceful shutdown

`signal.NotifyContext` is the modern, race-free way to bind signals to a
context. On `SIGINT`/`SIGTERM` the context cancels, the main goroutine
calls `http.Server.Shutdown` with a deadlined context, and the server
stops accepting new connections while draining in-flight ones. If the
drain exceeds `SHUTDOWN_TIMEOUT` we exit anyway — better to be killed by
the orchestrator with a known signal than to hang forever.

## Rate limiting

`golang.org/x/time/rate` is the canonical token bucket. Per-IP buckets
live in a `map[string]*rate.Limiter` guarded by a mutex. This is fine for
a single replica; for N replicas the *effective* rate is `N × RPS`, and
the limiter should move to Redis (`INCR` with `EXPIRE` is the simplest
correct pattern).

## Testing strategy

- **Unit tests** for pure functions and middleware (`config`, `middleware`).
- **Handler tests** with `httptest.NewRecorder` — no network, fast, deterministic.
- **Server smoke test** instantiates the full router and asserts the
  contract of each route (status code per path). This is what catches
  routing-table regressions that pure handler tests miss.
- **Integration tests** (not in this repo yet) belong behind
  `//go:build integration` so `go test ./...` stays fast for everyone.

## What's intentionally *not* here

- A custom logger wrapper. `slog` is enough; wrappers rot.
- A DI container. `cmd/server/main.go` wires dependencies by hand. With
  ~5 dependencies, plain Go is clearer than any framework.
- A REST framework. `gorilla/mux` + `net/http` covers everything the
  service needs at this scale.
- Generated mocks. Interfaces are defined where they are *consumed*, so
  tests can write small hand-rolled fakes instead of generated mocks
  that drift.

## Where to grow

- **Service layer**: add `internal/<domain>` packages with their own
  interfaces and a thin `service.go` that handlers call into.
- **Persistence**: introduce `internal/store/<engine>` with a small
  interface owned by the service layer. The MySQL/Redis clients live
  here; nothing else imports the driver packages.
- **Observability**: add `/metrics` via `prometheus/client_golang` and
  wrap the middleware chain with `otelhttp` for tracing.
- **Auth**: replace the bearer token with JWT validation
  (`go-jose`/`golang-jwt`) and a `claims` value carried in context.
