# HCI Asset – Runbook

One-page guide for checking health, reading logs, restarting services, and handling common failures.

---

## Health checks

| Check | How | Expected |
|-------|-----|----------|
| **API liveness** | `curl http://localhost:8080/health` | `200` body `ok` |
| **API readiness (DB)** | `curl http://localhost:8080/ready` | `200` body `ok` if DB is reachable; `503` body `db unreachable` if not |
| **Web UI** | `curl http://localhost:3000/health` | `200` body `ok` |

With Docker Compose, containers use these for `healthcheck`; `docker compose ps` shows health status.

---

## Logs

- **API**: Logs to stdout (JSON if `LOG_FORMAT=json`). Each request is logged with method, path, status, duration, and request ID (`X-Request-ID`).
- **Where to look**:
  - **Docker**: `docker compose logs api` or `docker compose logs -f api` for follow.
  - **Systemd / bare**: journalctl or the output redirection you configured.
- **Use request ID**: On errors, note the `X-Request-ID` response header (or the id in the log line) to correlate with log lines.

---

## Restarting services

- **Docker Compose**
  - Restart API only: `docker compose restart api`
  - Restart all: `docker compose restart`
  - Rebuild and start: `docker compose up -d --build`
- **Graceful shutdown**: API listens for SIGTERM/SIGINT, stops accepting new requests, waits up to 30s for in-flight requests, then exits. Send SIGTERM (e.g. `docker stop`) rather than SIGKILL when possible.

---

## "DB unreachable" (/ready returns 503)

1. **Check Postgres is running**: `docker compose ps` or `pg_isready -h <DB_HOST> -p <DB_PORT> -U <DB_USER>`.
2. **Check connectivity** from the API host/container: `docker compose exec api sh` then try connecting (e.g. install `nc` and test port, or use a one-off container that has `psql`).
3. **Check env**: API must have correct `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASS`, `DB_NAME`. In Docker Compose, `DB_HOST=postgres` (service name).
4. **Restart API** after DB is back: `docker compose restart api`. Optionally restart Postgres first if it was down: `docker compose restart postgres`.
5. **If DB was restarted**, ensure migrations are applied (API runs them on startup; if you skipped with `SKIP_MIGRATIONS=1`, run migrations or remove the env and restart API).

---

## "Scan stuck" (job stays running or never completes)

1. **Confirm status**: `GET /v1/scans/{id}` (or use Web UI **Active Scans** → View). Check `status` (e.g. `running`, `complete`, `canceled`, `error`) and `error` message if any.
2. **Cancel if needed**: `POST /v1/scans/{id}/cancel` (or **Cancel** in the scan detail UI). Only in-memory running jobs can be canceled; if the API restarted, the job may already be gone from memory (check DB for the job row).
3. **If nmap is slow or hanging**: Scans run in the background; large ranges (e.g. /16) can take a long time. Check API logs for the scan; if the process is stuck, restart the API (running scans will stop; completed ones are persisted).
4. **Clear old jobs**: To reset the active list, use **Clear all scans** in the Web UI or `DELETE /v1/scans` (admin). This deletes scan job rows; it does not stop in-memory running jobs (they will finish and then no longer appear in the list).

---

## Metrics (Prometheus)

- **Endpoint**: `GET http://localhost:8080/metrics` (no auth by default; restrict access in production if needed).
- **Main metrics**:
  - `http_request_duration_seconds` – request latency by method, path, status.
  - `http_requests_total` – request count by method, path, status.
  - `scan_jobs_running` – number of scans currently running (in-memory).
  - `scan_jobs_total` – total scan jobs finished, by status (completed, canceled, error).

Configure Prometheus to scrape the API (e.g. `scrape_configs` target `api:8080`, path `/metrics`).

---

## Escalation

- **Repeated DB unreachable**: Check Postgres logs, disk, and network. Consider DB backups and failover if applicable.
- **API crashes on startup**: Check logs for panic or fatal (e.g. missing `JWT_SECRET` in prod, migration errors). Fix config or DB and restart.
- **High memory/CPU**: Check `/metrics` and logs; consider resource limits (e.g. `deploy.resources.limits` in docker-compose) and tuning (e.g. `DB_MAX_OPEN_CONNS`).
