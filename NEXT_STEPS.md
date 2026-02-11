# Next steps

- [x] **List recent scans on Scans page** – API `GET /scans` returns recent job IDs; Scans page shows a “Recent scans” table with links to each scan detail.
- [x] **Show heartbeat error on asset detail** – When `?heartbeat_error=1` is present, the asset detail page shows “Failed to record heartbeat. Try again.”
- [ ] **Pagination for assets** – Use API `limit`/`offset` and add Next/Previous (or page numbers) on the assets list.
- [ ] **Docker Compose** – Add or extend `docker-compose.yml` so Postgres, API, and optionally the Web UI run together with one command.
- [ ] **Tests** – Unit tests for handlers/repos (e.g. asset/user CRUD, scan job lifecycle) and/or a small API integration test.
- [ ] **CI** – GitHub Actions (or similar) workflow that runs `go build ./...` and tests on push/PR.
