# Docker and GHCR Deployment Design

## Scope

Package the Blog API project as a Docker image that runs on Linux x86_64 servers, provide a Docker Compose deployment, and publish the image to GitHub Container Registry with the `latest` tag after pushes to `main`.

## Current Runtime

The application is built from the `api` repository root. The Go process reads `.env` and JSON configuration relative to its working directory, uses SQLite, serves the Vue panel under `/panel/`, and serves files from the configured resource directory. The frontend build currently produces the panel under `data/panel` for local execution.

## Architecture

Use a multi-stage Docker build:

1. Build the Vue panel with the repository's pnpm version.
2. Build the Go backend with CGO enabled so `go-sqlite3` is linked correctly.
3. Produce a runtime image containing the Go binary, migrations, and the compiled panel in an image-owned directory.

The runtime working directory is `/app`. The panel is stored separately from `/app/data` so the persistent data mount cannot hide or replace the panel. The backend static-file configuration or serving logic will be adjusted as part of implementation to resolve the panel from this image-owned location while keeping user data under `/app/data`.

## Persistent Data

Docker Compose mounts the deployment directory's `./data` to `/app/data`. This preserves:

- `data/config/system_config.json`
- `data/config/friend_list.json`
- `data/database.db`
- `data/image/`
- configured local resource files

The image does not rely on persistent frontend files. A new image can therefore update the panel without deleting the database or uploaded files.

The `.env` file is mounted read-only at `/app/.env`. Secrets remain outside the image and are not committed or logged.

## Compose Runtime

The Compose service will:

- pull `ghcr.io/kmoretti/blog_api:latest`
- expose host port `10024` to container port `10024`
- mount `./data:/app/data`
- mount `./.env:/app/.env:ro`
- restart automatically unless explicitly stopped
- include a health check against an existing lightweight HTTP endpoint, if one is available; otherwise use a TCP check or add the smallest compatible health endpoint without changing application behavior

The container will run as `linux/amd64` and bind to `0.0.0.0:10024` by default.

## GitHub Actions

Add a workflow under `.github/workflows/` that runs on pushes to `main`.

The workflow will:

- check out the repository
- authenticate to GHCR using the built-in `GITHUB_TOKEN`
- grant only `contents: read` and `packages: write`
- configure Docker Buildx
- build and push only `linux/amd64`
- publish `ghcr.io/kmoretti/blog_api:latest`

The workflow will not publish commit hashes as the primary deployment tag. The image reference will be centralized in Compose so deployment always uses `latest`.

## Configuration and First Run

The deployment instructions will require the operator to:

1. Create a deployment directory.
2. Download `docker-compose.yml` and `.env.example` or create equivalent files.
3. Create `.env` with production credentials and secrets.
4. Create `data/config/system_config.json` from the example configuration.
5. Optionally create `data/config/friend_list.json`.
6. Ensure the host user can write to `data`.
7. Pull and start the service with Docker Compose.

The implementation must preserve existing local behavior for API routes, panel routes, SQLite migrations, scheduled jobs, local image/resource storage, optional OSS integrations, Telegram/Discord integrations, email verification, Turnstile, and PWA assets, subject to their required configuration and external network access.

## Failure Handling

The image build must fail on frontend type/build errors or Go compilation errors. Compose startup must fail clearly when required configuration files or environment values are missing. The application must not include secrets in the image layers.

Persistent data must remain outside the image so pulling a new `latest` image does not overwrite it. The deployment procedure will include a database backup recommendation before image updates.

## Verification

Verify the implementation by:

- building the frontend with the repository's package manager
- compiling the Go backend with CGO enabled
- building the Docker image for `linux/amd64`
- starting the service with Compose using a temporary data directory
- checking the API response and `/panel/` response
- checking that panel assets are available while `/app/data` is mounted
- checking that SQLite and configuration files remain persistent across container recreation
- validating the workflow YAML and image naming
