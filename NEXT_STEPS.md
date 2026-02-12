# Next steps

- [x] **List recent scans on Scans page** – API `GET /scans` returns recent job IDs; Scans page shows a “Recent scans” table with links to each scan detail.
- [x] **Show heartbeat error on asset detail** – When `?heartbeat_error=1` is present, the asset detail page shows “Failed to record heartbeat. Try again.”
- [x] **Pagination for assets** – Assets list uses `limit=20` and `offset`; Previous/Next links preserve search; page in query.
- [x] **Docker Compose** – API uses config for DB DSN (works with `DB_HOST=postgres`); `web` service added (Dockerfile.web), depends on API, `HCI_ASSET_API_URL=http://api:8080`; `docker compose up` runs postgres, api, web.
- [x] **Tests** – Repo tests added: `internal/repo/asset_test.go` and `internal/repo/user_test.go` using go-sqlmock (Create, Get, Get not found, List, Delete, Heartbeat for assets; Create, GetByID, GetByUsername for users).
- [x] **CI** – GitHub Actions workflow `.github/workflows/ci.yml` runs on push/PR to main/master: checkout, setup Go (from go.mod), `go build ./...`, `go test -v ./...`.
