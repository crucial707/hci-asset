# Next steps

## Completed

- [x] **List recent scans on Scans page** – API `GET /scans` returns recent job IDs; Scans page shows a “Recent scans” table with links to each scan detail.
- [x] **Show heartbeat error on asset detail** – When `?heartbeat_error=1` is present, the asset detail page shows “Failed to record heartbeat. Try again.”
- [x] **Pagination for assets** – Assets list uses `limit=20` and `offset`; Previous/Next links preserve search; page in query.
- [x] **Docker Compose** – API uses config for DB DSN (works with `DB_HOST=postgres`); `web` service added (Dockerfile.web), depends on API, `HCI_ASSET_API_URL=http://api:8080`; `docker compose up` runs postgres, api, web.
- [x] **Tests** – Repo tests: `internal/repo/asset_test.go`, `internal/repo/user_test.go` (go-sqlmock). Handler tests: `internal/handlers/asset_test.go` (List, Get, Create, errors).
- [x] **CI** – GitHub Actions workflow `.github/workflows/ci.yml` runs on push/PR to main/master: checkout, setup Go (from go.mod), `go build ./...`, `go test -v ./...`.

## Testing

- [ ] **More handler tests** – Auth (login), user CRUD, scan start/status handlers with httptest + mocked repos.
- [ ] **API integration test** – Small test that hits real HTTP routes (test DB or sqlmock-backed server) for a few critical flows.

## Security & operations

- [x] **Rate limiting** – Per-IP limit on auth: `internal/middleware/ratelimit.go` (10 req/min per IP, burst 5) applied to `POST /auth/login` and `POST /auth/register`; returns 429 JSON when exceeded.
- [x] **HTTPS / TLS** – API reads `TLS_CERT_FILE` / `TLS_KEY_FILE` from env (`internal/config.Config`) and, when both are set, serves HTTPS via `http.ListenAndServeTLS`; otherwise it falls back to plain HTTP.
- [ ] **Health check with DB** – `/health` (or `/ready`) pings Postgres so orchestrators can fail unhealthy instances.

## Features

- [ ] **Audit log** – Record who created/updated/deleted assets or users and when.
- [ ] **Asset tags or groups** – Filter and group assets in the UI.
- [ ] **Scan scheduling** – Recurring scans (e.g. cron-like) in addition to on-demand.

## Docs & DX

- [ ] **API section in README** – Document main endpoints and how auth works.
- [ ] **Local dev** – Script or short doc for `docker compose up` plus DB setup and running tests.
