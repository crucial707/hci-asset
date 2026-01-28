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

