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
| JWT_SECRET | Secret used to sign JWT tokens |

If PostgreSQL is running on your host machine, use:
"DB_HOST=host.docker.internal"

--------------------------------------------------------------------

## Database setup (PostgreSQL)

The API does **not** run migrations automatically. When using **docker-compose**, Postgres is created with database `assetdb`, user `assetuser`, password `assetpass`. The `assets` and `users` tables must exist.

- **Create or fix the users table** in the Postgres container (e.g. after `docker compose up -d postgres` or if you run Postgres as container `asset-postgres`). The API uses **username-only** auth (no passwords). If your table has a NOT NULL `password_hash` column, either drop it or run the fix below.

  **Option A – Fresh: create table (no password_hash):**
  ```powershell
  docker exec -i asset-postgres psql -U assetuser -d assetdb -c "CREATE TABLE IF NOT EXISTS users (id SERIAL PRIMARY KEY, username VARCHAR(255) NOT NULL UNIQUE, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW());"
  ```

  **Option B – Existing table has password_hash:** align schema with the app (removes the column so INSERT only needs username):
  ```powershell
  docker exec -i asset-postgres psql -U assetuser -d assetdb -c "ALTER TABLE users DROP COLUMN IF EXISTS password_hash;"
  ```

- **Assets table**: If you use the migrations in `internal/db/migrations/`, run the assets migration first (e.g. the `*_create_assets_table.up.sql` file) in the same way, or ensure the table exists. The API will fail at startup with a clear message if the `users` table is missing.

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
  - `hci-asset assets list` – list assets in a go-pretty table (or JSON with `--json`)
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

A small web dashboard runs as a separate binary and talks to the API.

- **Build and run** (API must be running on port 8080):

  ```bash
  go run ./cmd/web
  ```

  Then open http://localhost:3000

- **Pages**: Login (username only; register via CLI first), Dashboard (asset count + recent assets), Assets list, Asset detail.

- **Config**: `HCI_WEB_PORT` (default 3000), `HCI_ASSET_API_URL` (default http://localhost:8080). Create a user first with `go run ./cmd/cli login --username you --register` (or `.\hci-asset.exe login --username you --register`); then log in to the UI with that username. The UI stores a JWT in a cookie.

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

