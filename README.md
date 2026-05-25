# GoWebApp

A small, production-shaped Go HTTP service. The repository is intentionally
sized to be reviewable end-to-end in a single sitting while exercising the
patterns a senior reviewer expects to see: clean layout, dependency
inversion at the edges, structured logging, graceful shutdown, middleware
chain, tests with the race detector, multi-stage Docker, and a CI gate.

---

## Table of contents

- [Architecture](#architecture)
- [Layout](#layout)
- [Quick start](#quick-start)
- [Configuration](#configuration)
- [HTTP API](#http-api)
- [Testing](#testing)
- [Docker & docker-compose](#docker--docker-compose)
- [CI/CD](#cicd)
- [Operational concerns](#operational-concerns)
- [Further reading](#further-reading)

---

## Architecture

```
                +----------------------+
   client  -->  |  Recover             |   panics -> 500, stack to log
                |    RequestLogger     |   structured access log
                |      mux.Router      |
                |        RateLimit     |   per-IP token bucket
                |          RequireAuth |   (selected routes only)
                |            Handler   |   thin: parse, call service, encode
                +----------------------+
                          |
                          v
                  service / repository
                          |
                          v
                   Redis  +  MySQL
```

- **`cmd/server`** owns process lifecycle: load config, build the logger,
  wire dependencies, install signal handlers, and block on
  `http.Server.ListenAndServe`. Shutdown drains in-flight requests within
  `SHUTDOWN_TIMEOUT` before exiting.
- **`internal/server`** composes the router and middleware. The middleware
  order is deliberate: `Recover` is outermost so even a panic inside
  `RequestLogger` becomes a 500 with a logged stack trace.
- **`internal/handler`** holds thin HTTP handlers. They never touch
  infrastructure directly — they call into service packages that can be
  swapped in tests.
- **`internal/middleware`** is the cross-cutting layer: request logging,
  panic recovery, rate limiting, and bearer-token auth.
- **`internal/config`** is the single source of truth for env vars. The
  rest of the codebase never calls `os.Getenv`, which makes configuration
  drift visible in code review.

## Layout

```
.
├── cmd/server/              # process entrypoint (main package)
├── internal/
│   ├── config/              # typed, validated configuration
│   ├── logger/              # slog setup (json/text, level)
│   ├── server/              # router + middleware composition
│   ├── handler/             # HTTP handlers (health, app)
│   └── middleware/          # logging, recover, ratelimit, auth
├── docs/                    # ARCHITECTURE.md, INTERVIEW.md
├── .github/workflows/ci.yml # lint / test / build / docker
├── Dockerfile               # multi-stage, distroless, non-root
├── docker-compose.yml       # app + redis + mysql
├── Makefile                 # developer entrypoints
├── .env.example             # copy to .env
└── go.mod
```

`internal/` is enforced by the Go toolchain — packages there cannot be
imported from outside this module, which keeps the public surface honest.

## Quick start

```bash
git clone https://github.com/vijay8059/GoWebApp.git
cd GoWebApp
cp .env.example .env

# Native
make tidy
make run                         # starts on :8080

# Containerised
make up                          # docker compose up app + redis + mysql
curl -s localhost:8080/healthz
curl -s localhost:8080/api/v1/hello?name=vj
```

Requires Go 1.22+ and Docker 24+.

## Configuration

All runtime configuration is environment-driven (12-factor). See
[`.env.example`](.env.example) for the full catalog. Highlights:

| Variable           | Default            | Purpose                              |
|--------------------|--------------------|--------------------------------------|
| `APP_ENV`          | `development`      | gates production validations         |
| `HTTP_ADDR`        | `:8080`            | listen address                       |
| `LOG_LEVEL`        | `info`             | `debug` / `info` / `warn` / `error`  |
| `LOG_FORMAT`       | `json`             | `json` is required in production     |
| `RATE_LIMIT_RPS`   | `50`               | per-IP token refill rate             |
| `RATE_LIMIT_BURST` | `100`              | per-IP burst size                    |
| `SHUTDOWN_TIMEOUT` | `15s`              | grace period for in-flight requests  |
| `API_TOKEN`        | _empty_            | bearer token for `RequireAuth`       |

In `APP_ENV=production`, `config.Load` refuses to start without a
`SESSION_AUTH_KEY` and rejects `LOG_FORMAT=text` — the kind of guardrail
that pays for itself the first time someone copies the dev `.env` into
prod.

## HTTP API

| Method | Path                | Auth | Description               |
|--------|---------------------|------|---------------------------|
| GET    | `/healthz`          | no   | liveness; always 200      |
| GET    | `/readyz`           | no   | readiness; 503 when draining |
| GET    | `/api/v1/hello`     | no   | demo handler              |
| GET    | `/api/v1/whoami`    | yes  | requires `Authorization: Bearer …` |

```bash
curl -i localhost:8080/api/v1/whoami                          # 401
curl -i -H 'Authorization: Bearer dev-token' \
        localhost:8080/api/v1/whoami                          # 200
```

## Testing

```bash
make test     # race detector + coverage
make cover    # open HTML coverage report
```

The suite covers handlers (table-driven, `httptest`), middleware
(panic-into-500, rate limit thresholds, bearer auth), config validation,
and a server-level smoke test that exercises the full middleware chain
against the real router.

Integration tests against Redis/MySQL would live behind a `//go:build
integration` tag and be wired into a separate CI job that provisions the
dependencies via the same `docker-compose.yml` used locally.

## Docker & docker-compose

The Dockerfile is multi-stage:

1. `golang:1.22-alpine` builds a static, stripped binary with cached
   modules and build cache mounts.
2. `gcr.io/distroless/static-debian12:nonroot` runs the binary as
   `nonroot:nonroot`. No shell, no package manager, no setuid binaries.

```bash
make docker                     # build local image
docker run --rm -p 8080:8080 gowebapp:dev
```

`docker-compose.yml` brings up the app together with Redis and MySQL,
wired through healthchecks so the app waits for its dependencies before
starting.

## CI/CD

[`.github/workflows/ci.yml`](.github/workflows/ci.yml) runs four jobs on
every push and PR to `main`:

1. **lint** — `go vet` + `golangci-lint` (latest).
2. **test** — race detector, coverage upload as an artifact.
3. **build** — produces a Linux/amd64 binary, uploaded as an artifact.
4. **docker** — builds the production image via Buildx with GHA cache;
   only runs on push-to-main, mirroring what a release pipeline would do.

`concurrency.cancel-in-progress: true` ensures a force-push doesn't burn
CI minutes on superseded commits.

## Operational concerns

- **Graceful shutdown**: `signal.NotifyContext` catches `SIGINT`/`SIGTERM`,
  then `http.Server.Shutdown` drains in-flight requests within
  `SHUTDOWN_TIMEOUT`. Long-running handlers get a deadlined context so
  they can abort cooperatively.
- **Structured logging**: `log/slog` JSON output, one record per request,
  plus per-event records for panics. Fields are stable so they can be
  indexed.
- **Connection pooling**: `DB_MAX_OPEN_CONNS` / `DB_MAX_IDLE_CONNS` /
  `DB_CONN_MAX_LIFE` are exposed via config and applied to `database/sql`.
  Defaults assume a moderately sized service; tune based on the database's
  `max_connections` and the replica count.
- **Rate limiting**: in-memory token bucket per client IP. For a
  multi-replica deployment, swap the limiter store for Redis
  (`INCR` + `EXPIRE` or a Lua script) so the limit is enforced globally.
- **Observability hooks**: the structured logs are the floor; the next
  layer would be `/metrics` (Prometheus) and OpenTelemetry traces around
  the middleware chain.

## Further reading

- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — design decisions and
  trade-offs at more depth.
- [`docs/INTERVIEW.md`](docs/INTERVIEW.md) — interview-ready talking
  points: testing strategy, deployment workflow, debugging approach,
  Docker setup, CI/CD pipeline.
