# hci-asset
Internal Asset &amp; Security Service (Tailnet + Proxmox)

=Project Goal=

Internal Go service that tracks assets running inside a Tailscale-connected Proxmox cluster.

-MVP Features

-Asset registration API

-Asset heartbeat

-Asset listing

-Token-based authentication

-PostgreSQL persistence

-Defensive Security tools built-in

-WebApp Front End
--------------------------------------------------------------------


==Build and Run with Docker==

This project includes a Dockerfile for building and running the HCI Asset API inside a container.

=Prerequisites=

-Docker installed

-PostgreSQL available (local or container)

-Git

=Build the Docker Image=

From the root of the repository:
"docker build -t hci-asset-api ."

This command compiles the Go application and creates a lightweight runtime image tagged as hci-asset-api.

=Run with Docker Compose (recommended)=

From the repo root, run the full stack (Postgres, API, Web UI):

  docker compose up -d

Then open the Web UI at http://localhost:3000 and the API at http://localhost:8080. The API runs database migrations on startup (see [Database setup](#database-setup-postgresql)).

=Run the Container=
"docker run -p 8080:8080 \
-e DB_HOST=host.docker.internal \
-e DB_PORT=5432 \
-e DB_NAME=hci_asset \
-e DB_USER=postgres \
-e DB_PASS=yourpassword \
-e JWT_SECRET=supersecretkey \
--name hci-asset-api \
hci-asset-api"

=Environmental Varibales=
| Variable   | Description                    |
| ---------- | ------------------------------ |
| DB_HOST    | PostgreSQL host                |
| DB_PORT    | PostgreSQL port                |
| DB_NAME    | Database name                  |
| DB_USER    | Database username              |
| DB_PASS    | Database password              |
| JWT_SECRET | Secret used to sign JWT tokens. When **ENV** is not `dev`, must be set to a secure value (default is refused). |
| ENV | `dev` (default) or `prod`. When `prod`, startup fails if **JWT_SECRET** is unset or equals the default. |
| JWT_EXPIRE_HOURS | JWT token lifetime in hours (default `24`). |
| NMAP_PATH  | Path to nmap executable (default: `nmap`). For Docker the image includes nmap. On Windows, set to e.g. `C:\Program Files (x86)\Nmap\nmap.exe` if scans fail. |
| CORS_ALLOWED_ORIGINS | Comma-separated list of origins allowed for CORS (e.g. `http://localhost:3000`, `https://app.example.com`). When unset, no CORS headers are sent (same-origin only). |

If PostgreSQL is running on your host machine, use:
"DB_HOST=host.docker.internal"

--------------------------------------------------------------------

## Database setup (PostgreSQL)

The API **runs migrations automatically** on startup. Schema is defined in `internal/db/migrations/` (golang-migrate format: `*_name.up.sql` / `*_name.down.sql`). On first connect, all pending migrations are applied; on later starts, already-applied migrations are skipped.

When using **docker-compose**, Postgres is created with database `assetdb`, user `assetuser`, password `assetpass`. Just start the stack; the API will create or update tables when it starts.

- **Skip migrations**: Set env `SKIP_MIGRATIONS=1` if you run migrations separately (e.g. from a job or CLI). The API will then assume the schema is already up to date.
- **Existing installs**: If you previously created tables manually (before this migration runner), back up your data before starting the API with migrations enabled, or set `SKIP_MIGRATIONS=1` until you are ready.
- **Auth**: Users have a `role` (`viewer` or `admin`). Only **viewer** can log in without a password; **admin** requires a password. The `users` table has `role` and optional `password_hash`.
do- **Existing users without a role / setting admin**: The Web UI **Users** page shows each user’s role and lets you change it (Edit → set Role to Admin). You can also set an existing user to admin via SQL (e.g. in the postgres container):  
  `UPDATE users SET role = 'admin' WHERE username = 'admin';`  
  **Admin users must have a password.** If that user has no password yet, set a bcrypt hash in the DB or create a new admin via Register with username, password, and role `admin`.

--------------------------------------------------------------------

## API

The API runs on port 8080 by default (configurable with `PORT`). All JSON request/response bodies use `Content-Type: application/json`.

### Health and readiness

| Method | Path    | Auth | Description |
|--------|---------|------|-------------|
| GET    | `/health` | No  | Liveness: returns `ok`. |
| GET    | `/ready`  | No  | Readiness: pings Postgres; returns 200 `ok` or 503 `db unreachable`. |

### Authentication and roles

Users have a **role**: `viewer` (view only) or `admin`. Obtain a JWT by registering and then logging in.

- **Viewer**: Can log in with **username only** (no password). Can only read: list/get assets, users, audit log, scans, schedules. Create/update/delete return 403 Forbidden.
- **Admin**: **Must have a password** (required at register and at login). Can do everything (create, update, delete assets, users, schedules; run/cancel scans; heartbeat).

1. **Register** (create a user):  
   `POST /auth/register`  
   Body: `{"username": "alice"}` (viewer, no password) or `{"username": "alice", "password": "secret", "role": "admin"}` (admin requires password).  
   Default role is `viewer`. Returns: `{"id": 1, "username": "alice", "role": "viewer"}` (or 200 with existing user if already registered).

2. **Login**:  
   `POST /auth/login`  
   Body: `{"username": "alice"}` for viewer (no password), or `{"username": "alice", "password": "secret"}` when the account has a password (required for admin).  
   Returns: `{"token": "<jwt>", "user": {"id": 1, "username": "alice", "role": "viewer"}}`.

3. **Use the token** on protected routes by sending the header:  
   `Authorization: Bearer <token>`

Auth endpoints are rate-limited per IP (10 requests/minute, burst 5); excess requests receive 429. Mutating operations (POST/PUT/DELETE on assets, users, schedules; scan start/cancel; heartbeat) require **admin**; viewers receive 403.

### Protected endpoints (require `Authorization: Bearer <token>`)

**Assets**

| Method | Path | Description |
|--------|------|-------------|
| GET    | `/assets` | List assets. Query: `limit`, `offset`, `search`. |
| GET    | `/assets/{id}` | Get one asset. |
| POST   | `/assets` | Create. Body: `{"name": "...", "description": "..."}`. |
| PUT    | `/assets/{id}` | Update. Body: `{"name": "...", "description": "..."}`. |
| POST   | `/assets/{id}/heartbeat` | Update `last_seen` (agent check-in). |
| DELETE | `/assets/{id}` | Delete asset. |

**Users**

| Method | Path | Description |
|--------|------|-------------|
| GET    | `/users` | List users. |
| GET    | `/users/{id}` | Get one user. |
| POST   | `/users` | Create. Body: `{"username": "..."}`. |
| PUT    | `/users/{id}` | Update. Body: `{"username": "..."}`. |
| DELETE | `/users/{id}` | Delete user. |

**Scans**

| Method | Path | Description |
|--------|------|-------------|
| GET    | `/scans` | List recent scan jobs. |
| POST   | `/scans` | Start scan. Body: `{"target": "192.168.1.0/24"}`. Returns `{"job_id": "1", "status": "running"}`. |
| GET    | `/scans/{id}` | Get scan status and discovered assets. |
| POST   | `/scans/{id}/cancel` | Cancel a running scan. |

Legacy paths `POST /scan`, `GET /scan/{id}`, `POST /scan/{id}/cancel` behave the same as the `/scans` variants.

**Scan schedules (recurring)**

| Method | Path | Description |
|--------|------|-------------|
| GET    | `/schedules` | List schedules. Query: `limit`, `offset`. |
| POST   | `/schedules` | Create. Body: `{"target": "192.168.1.0/24", "cron_expr": "0 * * * *", "enabled": true}` (5-field cron: min hour day month weekday). |
| GET    | `/schedules/{id}` | Get one schedule. |
| PUT    | `/schedules/{id}` | Update. Body: `{"target": "...", "cron_expr": "...", "enabled": true}`. |
| DELETE | `/schedules/{id}` | Delete schedule. |

Enabled schedules are run by a background scheduler; each run starts an on-demand scan for that schedule’s target.

Errors return JSON: `{"error": "message"}` with an appropriate HTTP status (400, 401, 404, 429, 500).

--------------------------------------------------------------------

## CLI Usage

The repository also includes a Go-based CLI for interacting with the API.

- **Run without installing** (from repo root):

  ```powershell
  go run ./cmd/cli login --username ab --register
  go run ./cmd/cli assets list
  ```

  On Windows you can build an executable and run it in this directory:

  ```powershell
  go build -o hci-asset.exe ./cmd/cli
  .\hci-asset.exe login --username ab --register
  .\hci-asset.exe assets list
  ```

  To use `hci-asset` from anywhere, add the folder containing `hci-asset.exe` to your PATH.

- **Commands** (shown as `hci-asset`; use `go run ./cmd/cli` or `.\hci-asset.exe` if not on PATH):
  - `hci-asset assets list` – list assets in a go-pretty table (or JSON with `--json`); includes **last seen** (heartbeat)
  - `hci-asset assets heartbeat [id]` – record a heartbeat for an asset (updates `last_seen`)
  - `hci-asset users list` – list users in a go-pretty table (or JSON with `--json`)
  - `hci-asset scan start [target]` – start a network scan
  - `hci-asset scan status [jobID]` – check scan status and discovered assets (table output)
  - `hci-asset scan cancel [jobID]` – cancel a running scan and show any discovered assets

### CLI Configuration

- **API URL**: The CLI talks to the API via a base URL.
  - Default: `http://localhost:8080`
  - Override with environment variable:

    ```bash
    export HCI_ASSET_API_URL="http://localhost:8080"
    ```

    (Point this at your dev/stage/prod API as needed.)

### Example CLI commands

```bash
# List assets in a table
hci-asset assets list

# List assets as raw JSON
hci-asset assets list --json

# List users in a table
hci-asset users list

# Start a scan
hci-asset scan start 192.168.1.0/24

# Check scan status
hci-asset scan status 1
```

--------------------------------------------------------------------

## Web UI

A web dashboard runs as a separate binary and talks to the API.

- **Build and run** (API must be running on port 8080):

  ```bash
  go run ./cmd/web
  ```

  Then open http://localhost:3000

- **Pages**:
  - **Login** – Username only (create a user first via CLI: `hci-asset login --username you --register`).
  - **Dashboard** – Asset count and recent assets with links to detail.
  - **Assets** – List with search (by name or description), “+ New asset”, and per-row View. From asset detail: Edit, Delete, **Record heartbeat** (updates last seen). Create and edit use a simple name + description form.
  - **Users** – List with “+ Add user”, and per-row Edit and Delete. Add and edit use a username-only form.
  - **Scans** – Start a network scan (target e.g. `192.168.1.0/24`). Scan detail shows target, status, **elapsed/duration timer**, cancel button, and discovered assets with links to asset pages. Page auto-refreshes every few seconds while a scan is running.

- **Config**: `HCI_WEB_PORT` (default 3000), `HCI_ASSET_API_URL` (default http://localhost:8080). The UI stores a JWT in a cookie after login.

- **Branding**: The UI is branded **Humboldt Cyber Intelligence** with a logo in the header and footer. To use your own logo image, add `logo.png` (or `logo.svg`) to `cmd/web/static/` and set the `src` in `templates/layout.html` and `templates/login.html` to `/static/logo.png` (or `.svg`). By default a placeholder is served from `static/logo.svg`.

--------------------------------------------------------------------

## Development

### Running tests

From the repo root:

- **All tests** (repo, handlers, CLI, etc.):
  ```bash
  go test ./...
  ```

- **Repo tests only** (database layer with go-sqlmock):
  ```bash
  go test ./internal/repo/...
  ```

- **Handler tests only** (API handlers with mocked repos):
  ```bash
  go test ./internal/handlers/...
  ```

**CI**: A GitHub Actions workflow (`.github/workflows/ci.yml`) runs on push and pull requests to `main`/`master`: it checks out the repo, sets up Go from `go.mod`, runs `go build ./...` and `go test -v ./...`.

### Local dev (quick start)

1. **Start the stack** (from repo root):
   ```bash
   docker compose up -d
   ```
   This starts Postgres, the API (port 8080), and the Web UI (port 3000). After changing web or API code, rebuild and recreate: `docker compose up -d --build` (or `docker compose build web` then `docker compose up -d web` for web-only changes).

2. **Database**: The API runs migrations on startup (see [Database setup (PostgreSQL)](#database-setup-postgresql)). No manual table creation needed when using Docker Compose.

3. **Run tests** (no Docker required; uses mocks):
   ```bash
   go test ./...
   ```
   For only repo or handler tests, see the commands above under *Running tests*.

4. **Optional – run API or Web locally** (with Go, against the same Postgres):
   - API: set `DB_HOST=localhost` (and DB_PORT/DB_NAME/DB_USER/DB_PASS to match your Postgres), then `go run ./cmd/api`.
   - Web: `go run ./cmd/web` (defaults to API at `http://localhost:8080`).

--------------------------------------------------------------------

## Production and deployment

### Environment mode

- **ENV**: Set to `dev` (default) for local/Docker Compose development, or `prod` for production.
- When **ENV=prod**, the API **refuses to start** if `JWT_SECRET` is unset or equals the default (`supersecretkey`). Set a strong secret (e.g. 32+ random bytes) in production.
- In prod you may also set `LOG_FORMAT=json` and restrict `CORS_ALLOWED_ORIGINS` to your real front-end origin(s).

### Secrets

- **Do not** commit real secrets to the repo. For production:
  - Use an **env file** that is not in version control: e.g. `env_file: .env.production` in `docker-compose` and add `.env.production` to `.gitignore`, or use a secrets manager.
  - Override at least: **JWT_SECRET** (required in prod), **DB_PASS** (and optionally DB_USER/DB_NAME if different from dev).
- Example override for the API service in Compose:
  ```yaml
  environment:
    ENV: prod
    JWT_SECRET: ${JWT_SECRET}   # set in shell or env file
    DB_PASS: ${DB_PASS}
  ```
  Or use `env_file: .env.production` and set variables there.

### Docker Compose: healthchecks and resources

- **Healthchecks** are defined for all three services:
  - **postgres**: `pg_isready -U assetuser` (interval 5s).
  - **api**: `GET http://localhost:8080/health` via curl (interval 10s).
  - **web**: `GET http://localhost:3000/health` via curl (interval 10s).
- **api** and **web** start only after their dependencies report healthy (`depends_on` with `condition: service_healthy`), so the API waits for Postgres and the Web UI waits for the API.
- **Resource limits** are set so a single service cannot exhaust the host:
  - postgres: 512M memory, 0.5 CPU
  - api: 512M memory, 0.5 CPU
  - web: 256M memory, 0.25 CPU  
  Adjust in `docker-compose.yml` under `deploy.resources.limits` if needed.

### Production checklist (summary)

- Set **ENV=prod** and a strong **JWT_SECRET**.
- Use **TLS** in production: set `TLS_CERT_FILE` and `TLS_KEY_FILE` for the API (or put the API behind a TLS-terminating proxy).
- Keep **DB backups**; document restore procedure.
- Restrict **CORS_ALLOWED_ORIGINS** to your real UI origin(s).
- Use **LOG_FORMAT=json** for production logging.
- Liveness: **GET /health**. Readiness (DB): **GET /ready**.

--------------------------------------------------------------------

=Verify the API is Running=
"curl http://localhost:8080/health"

Expected Response:
ok

=Stop the Container=
"docker stop hci-asset-api
docker rm hci-asset-api"

=Rebuild After Code Changes=
"docker build -t hci-asset-api .
docker run -p 8080:8080 --env-file .env hci-asset-api"

(Optional) You may create a .env file to store environment variables.

