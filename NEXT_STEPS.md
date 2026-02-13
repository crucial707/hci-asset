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
- [ ] **RBAC or roles** – Restrict who can manage users, delete assets, or manage schedules (e.g. admin vs viewer).
