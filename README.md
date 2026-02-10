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

## CLI Usage

The repository also includes a Go-based CLI for interacting with the API.

- **Binary name**: `hci-asset`
- **Commands**:
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

