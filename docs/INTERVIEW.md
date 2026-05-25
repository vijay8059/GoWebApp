# Interview-ready talking points

Short, pre-rehearsed answers for the questions a reviewer is likely to
ask. Aim for 60–90 seconds per answer; offer to dive deeper.

---

## 1. Walk me through the repository architecture.

> "The entry point is `cmd/server/main.go`. It owns the process: load
> config, build the logger, construct the HTTP server, and block until
> a signal triggers graceful shutdown. Everything else lives in
> `internal/`, which the Go toolchain prevents from being imported
> outside this module.
>
> The HTTP path is `cmd/server → internal/server → internal/handler`,
> with `internal/middleware` as the cross-cutting layer. I separated
> *router composition* from *handlers* so handlers stay testable without
> a real server, and so swapping the router doesn't ripple into the
> business code.
>
> The middleware chain is ordered deliberately: `Recover` is outermost
> so a panic anywhere becomes a 500 with a logged stack; `RequestLogger`
> sees every request including denials; `RateLimit` runs before auth so
> we don't waste auth CPU on attackers."

## 2. How do you test this?

> "Three layers, each fast enough to run on every save:
>
> - **Unit** — pure functions in `config`, and middleware tested
>   in isolation (panic → 500, rate limit triggers at the right point,
>   bearer auth rejects missing/wrong tokens).
> - **Handler** — `httptest.NewRecorder`, table-driven, JSON-decoded
>   assertions. No network, no sleeps, no flakes.
> - **Server smoke** — instantiate the whole router and assert the
>   status code per path. This catches routing-table regressions that
>   pure handler tests can't see.
>
> Everything runs with `-race` in CI. Integration tests against Redis
> and MySQL would go behind a `//go:build integration` tag so they
> don't slow the default `go test ./...` loop."

## 3. Describe the deployment workflow.

> "On push to a feature branch, CI runs lint, unit tests with the race
> detector, and a Linux/amd64 build. On push to `main` it additionally
> builds the production Docker image with Buildx and GHA cache.
>
> The image itself is a multi-stage build: `golang:1.22-alpine` for the
> build, `gcr.io/distroless/static-debian12:nonroot` for runtime. No
> shell in the runtime layer, runs as a non-root user, single static
> binary as the entrypoint. From here it's a `kubectl rollout` or an
> ECS service update — config flows in via env vars, secrets via the
> platform's secret manager, and the orchestrator probes `/readyz` to
> sequence the rollout."

## 4. How do you approach debugging?

> "I work from the outside in. First question: is the symptom in the
> logs? Every request emits one structured record with status,
> duration, and path — that usually narrows it to a route. Panics
> include the stack via the `Recover` middleware.
>
> If logs aren't enough, I reach for `pprof` (HTTP profile endpoints
> are cheap to add behind a flag), the race detector during tests, and
> `dlv` for step-through when the failure reproduces locally.
>
> For distributed cases, the next layer is OpenTelemetry spans through
> the middleware chain — once a request has a trace ID, correlating
> across services becomes a join, not detective work."

## 5. Walk me through the Docker setup.

> "Multi-stage. The build stage uses cache mounts for the module cache
> and the build cache, so an incremental build is seconds, not minutes.
> It produces a static binary with `CGO_ENABLED=0` and `-ldflags='-s -w
> -X main.version=…'` — stripped, with the version baked in.
>
> The runtime stage is `distroless/static:nonroot`. No shell, no apt,
> no setuid binaries, runs as UID 65532. The attack surface is the
> kernel and our binary — that's it. If the binary needs to be debugged
> in production, we ship a `:debug` tag of distroless that includes
> busybox, but the default image has nothing extra."

## 6. Walk me through the CI/CD pipeline.

> "Four jobs: lint, test, build, docker. Lint and test run in parallel;
> build waits on both; docker waits on build and only runs on push to
> `main`. `concurrency.cancel-in-progress` kills superseded runs so a
> force-push doesn't queue.
>
> The Go module cache and golangci-lint cache are both keyed by
> `go.sum`, which makes warm runs about 30 seconds. The Docker job uses
> Buildx with GHA cache, so unchanged layers come from cache across
> runs."

## 7. How would this repo demonstrate distributed-systems thinking?

> "A few seams are wired but not extended in this repo:
>
> - **Idempotent health probes** — separate liveness vs. readiness;
>   readiness flipping to 503 lets the orchestrator drain a replica
>   without dropping in-flight traffic.
> - **Bounded resource use** — connection pool limits, request
>   timeouts, rate limits — all configurable.
> - **Graceful shutdown** — `SIGTERM` triggers `Shutdown` with a
>   deadline, draining in-flight requests; this is what makes blue/green
>   and rolling deploys actually graceful.
> - **Stateless replicas** — the only in-process state is the rate
>   limiter, and the README calls out the Redis-backed version that's
>   required when N > 1.
>
> The natural next step is OpenTelemetry: once every span carries
> trace/span IDs, debugging stops being per-service and becomes
> per-request across the fleet."

## 8. How would you evaluate someone else's repository for this role?

> "I'd skim in roughly this order, because each layer informs the next:
>
> 1. `README` — does it tell me what the repo is and how to run it in
>    under five minutes?
> 2. Layout — is there a clear entrypoint? Is `internal/` used or is
>    everything public? Are layers separated?
> 3. `main` — does it do *only* lifecycle, or does business logic
>    leak in?
> 4. Configuration — env-driven or hardcoded? Validated?
> 5. Error handling — wrapped with context? Panics recovered?
> 6. Tests — fast, deterministic, exercising the seams that matter?
> 7. Dockerfile — multi-stage? Non-root? Pinned base?
> 8. CI — does it actually gate merges or is it decorative?
>
> The fastest tell is `main.go`: a 200-line `main` with database
> connections and route handlers in it tells me the rest of the repo
> will fight me. A short `main` that wires things up tells me the
> author has thought about lifecycle."
