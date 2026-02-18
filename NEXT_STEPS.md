# Next steps

## Completed

- [x] **List recent scans on Scans page** – API `GET /scans` returns recent job IDs; Scans page shows a “Recent scans” table with links to each scan detail.
- [x] **Show heartbeat error on asset detail** – When `?heartbeat_error=1` is present, the asset detail page shows “Failed to record heartbeat. Try again.”
- [x] **Pagination for assets** – Assets list uses `limit=20` and `offset`; Previous/Next links preserve search; page in query.
- [x] **Docker Compose** – API uses config for DB DSN (works with `DB_HOST=postgres`); `web` service added (Dockerfile.web), depends on API, `HCI_ASSET_API_URL=http://api:8080`; `docker compose up` runs postgres, api, web.
- [x] **Tests** – Repo tests: `internal/repo/asset_test.go`, `internal/repo/user_test.go` (go-sqlmock). Handler tests: `internal/handlers/asset_test.go` (List, Get, Create, errors).
- [x] **CI** – GitHub Actions workflow `.github/workflows/ci.yml` runs on push/PR to main/master: checkout, setup Go (from go.mod), `go build ./...`, `go test -v ./...`.

## Testing

- [x] **More handler tests** – `internal/handlers/auth_test.go` (Login, Login invalid/bad JSON, Register, Register bad JSON); `internal/handlers/user_test.go` (CreateUser, ListUsers, GetUser, GetUser not found/invalid ID, UpdateUser, DeleteUser); `internal/handlers/scan_test.go` (StartScan, ListScans, GetScanStatus, CancelScan; all with httptest + sqlmock where needed).
- [x] **API integration test** – `cmd/api/api_test.go`: full router built with `newRouter(db, cfg)` and sqlmock DB; `TestAPI_LoginThenListAssets` (POST /auth/login → GET /assets with Bearer token); `TestAPI_Health` for /health.

## Security & operations

- [x] **Rate limiting** – Per-IP limit on auth: `internal/middleware/ratelimit.go` (10 req/min per IP, burst 5) applied to `POST /auth/login` and `POST /auth/register`; returns 429 JSON when exceeded.
- [x] **HTTPS / TLS** – API reads `TLS_CERT_FILE` / `TLS_KEY_FILE` from env (`internal/config.Config`) and, when both are set, serves HTTPS via `http.ListenAndServeTLS`; otherwise it falls back to plain HTTP.
- [x] **Health check with DB** – `/ready` pings Postgres via `db.Ping()`; returns 200 "ok" when reachable, 503 "db unreachable" when not. `/health` remains a simple liveness check.

## Features

- [x] **Audit log** – Record who created/updated/deleted assets or users and when (table, repo, handler, GET /audit).
- [x] **Asset tags or groups** – Filter and group assets in the UI (tags column, API create/update/list by tag, Web UI form and list filter).
- [x] **Scan scheduling** – Recurring scans (cron-like): scan_schedules table, GET/POST/PUT/DELETE /schedules, background scheduler runs enabled schedules.

## Docs & DX

- [x] **API section in README** – New "## API" section: health/ready, auth (register, login, Bearer token), tables for assets, users, scans endpoints and error format.
- [x] **Local dev** – README "Local dev (quick start)" under Development: docker compose up, create DB tables (with link to Database setup), run tests, optional local API/Web against Postgres.

--------------------------------------------------------------------

## Suggested next steps

Priorities below are optional; pick by impact and effort.

### Web UI for new features

- [x] **Schedules page** – Web UI to list, create, edit, and delete scan schedules (GET/POST/PUT/DELETE `/schedules`). Nav link, schedules list, form (target, cron_expr, enabled), delete confirm.
- [x] **Audit log page** – Web UI to view the audit log (GET `/audit` with limit/offset). Nav link, table: user_id, action, resource type/id, details, when; Previous/Next pagination.

### Reliability and operations

- [x] **Persist scan jobs** – Scan jobs stored in `scan_jobs` table (id, target, status, started_at, completed_at, error, assets JSONB). List/Get read from DB; running jobs kept in memory for cancel; runScan persists on completion/cancel/error. History survives API restarts.
- [x] **Formal migrations** – golang-migrate in `internal/db/migrate.go`; migrations embedded from `internal/db/migrations/`; API runs `db.Run(dsn)` on startup (set `SKIP_MIGRATIONS=1` to skip). All schema lives in migrations; ad-hoc ensures removed from main.

### Testing and quality

- [x] **Schedule handler tests** – `internal/handlers/schedule_test.go`: ListSchedules, ListSchedules with query params, GetSchedule, GetSchedule NotFound/InvalidID, CreateSchedule, CreateSchedule BadRequest, UpdateSchedule, UpdateSchedule InvalidID, DeleteSchedule, DeleteSchedule InvalidID (httptest + sqlmock).
- [x] **Schedule repo tests** – `internal/repo/schedule_test.go`: List, List empty, ListEnabled, GetByID, GetByID NotFound, Create, Update, Delete (sqlmock).

### Product and security

- [x] **Password auth (optional)** – Optional password on register/login; bcrypt hashing; users without a password use username-only login. Web login form includes password field; API and README updated.
- [x] **RBAC or roles** – Roles `viewer` (view only) and `admin`. Only viewer can log in without a password; admin requires password at register and login. Mutating API routes require admin (403 for viewer). JWT includes role; RequireAdmin middleware protects create/update/delete.

--------------------------------------------------------------------

## Professional grade (suggested)

Priorities: High → API reliability, security, config & deploy; Medium → observability, API design, Web UI; Lower → tracing, E2E, runbooks.

### API: Reliability & operations

- [x] **Panic recovery** – Wrap API router in recovery middleware; on panic log stack and return 500 JSON (Web already has Recoverer; API does not).
- [x] **Request ID** – Generate or propagate `X-Request-ID` on every request; log it and optionally return in response headers for tracing/support.
- [x] **Structured logging** – Replace ad-hoc `log.Printf` with structured logging (e.g. slog/zerolog): request ID, method, path, status, duration, error; JSON in production for aggregation.
- [x] **DB connection pool** – Set `SetMaxOpenConns` / `SetMaxIdleConns` in `cmd/api/main.go` so the API doesn’t over-consume connections under load.
- [x] **Context in handlers** – Use `r.Context()` in handlers and pass into repo calls so queries can be cancelled on client disconnect and respect timeouts.
- [x] **Graceful shutdown** – Listen for SIGTERM/SIGINT; stop accepting new requests, finish in-flight with timeout, then close DB and exit (Kubernetes/Docker).

### API: Security & HTTP

- [x] **CORS** – Configurable CORS middleware (e.g. AllowOrigins from env) when Web UI or clients are on a different origin.
- [x] **Security headers** – Middleware: `X-Content-Type-Options: nosniff`, `X-Frame-Options`, `Content-Security-Policy` (minimal); optionally `Strict-Transport-Security` when HTTPS.
- [x] **No default JWT in prod** – Fail startup if `JWT_SECRET` is unset or equals a well-known dev default when not in dev mode.
- [x] **JWT expiry config** – Make token lifetime configurable (e.g. `JWT_EXPIRE_HOURS`); consider refresh tokens or sliding expiry.
- [x] **Validation and errors** – Validate request bodies (length, required fields, types); consistent JSON errors with field-level details; no internal details in 500 responses.

### API: Design & consistency

- [x] **API versioning** – Prefix routes with `/v1/` (e.g. `/v1/assets`, `/v1/auth/login`) for future breaking changes and clear contracts.
- [x] **OpenAPI / Swagger** – Document API with OpenAPI 3; serve `/openapi.json` or Swagger UI for discoverability and codegen.
- [x] **Pagination response shape** – Standardize list responses: `{"items": [...], "total": N, "limit": 20, "offset": 0}` for “Page X of Y”.

### Observability

- [x] **Prometheus metrics** – Expose `request_duration_seconds`, `request_total` by method/path/status, `scan_jobs_*`; `/metrics` endpoint (optional auth).
- [ ] **Tracing (optional)** – Distributed tracing (e.g. OpenTelemetry) for scan lifecycle and API calls across components.

### Web UI: UX & polish

- [x] **Layout and branding** – Clear app identity (logo, app name, nav hierarchy); improve section headings and spacing.
- [x] **Loading and errors** – Loading states for list/detail and after submit; global error banner or toast; per-form validation messages.
- [x] **Accessibility** – Focus management after login; semantic HTML and aria-*; color contrast and visible focus rings.
- [x] **Responsive** – Tables and forms usable on small screens (scroll, stacked layout, or card layout).
- [x] **Session display** – Show “Logged in as &lt;username&gt;” and role in nav; clear “Session expired” on 401 with next preserved.

### Deployment & config

- [x] **Secrets** – Don’t pass secrets via plain env in compose; use secrets files or vault; document overriding JWT_SECRET and DB_* in prod.
- [x] **Docker healthchecks** – Add `healthcheck` for API (GET /health) and optionally Web; use depends_on with conditions when supported.
- [x] **Resource limits** – Add deploy.resources.limits (CPU/memory) in compose so a runaway process doesn’t take down the host.
- [x] **Environment mode** – Support `dev` / `prod` (or similar) to disable default JWT, stricter CORS, and adjust log level / error verbosity.

### Testing & quality

- [ ] **Integration tests** – Suite against real Postgres (testcontainers or CI DB): start API, auth + CRUD + scan flows, assert responses and DB state.
- [ ] **E2E (optional)** – Critical journeys (e.g. login → list assets → heartbeat) via browser or API E2E to guard regressions.
- [ ] **Dependency checks** – Run `go mod tidy` and vulnerability check (e.g. govulncheck, Dependabot) in CI; document how to address findings.

### Documentation

- [x] **Production checklist in README** – TLS, non-default JWT, DB backups, CORS, logging, where to find metrics/health.
- [x] **Runbook** – One-page: how to check health, read logs, restart services; what to do on “DB unreachable” or “scan stuck”.
