# Docker and GHCR Deployment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a self-contained `linux/amd64` Docker image for the Blog API, deploy it with Docker Compose, and publish `ghcr.io/kmoretti/blog_api:latest` from GitHub Actions after pushes to `main`.

**Architecture:** Use a multi-stage Dockerfile rooted at `E:\kmoretti-github\blog_api\api`. The frontend is built during the image build and copied to an image-owned `/app/panel` directory, while `/app/data` is reserved for persistent SQLite, configuration, images, and local resources. Keep the existing Go working-directory assumptions by running the binary from `/app`.

**Tech Stack:** Go 1.25.4, CGO with SQLite, Vue 3, Vite, pnpm 10.30.3, Docker BuildKit/Buildx, Docker Compose, GitHub Container Registry, GitHub Actions.

---

## File Map

- Create: `E:\kmoretti-github\blog_api\api\Dockerfile` for the multi-stage frontend, backend, and runtime image.
- Create: `E:\kmoretti-github\blog_api\api\.dockerignore` for a minimal Docker build context.
- Create: `E:\kmoretti-github\blog_api\api\docker-compose.yml` for pulling and running `ghcr.io/kmoretti/blog_api:latest` with persistent data.
- Create: `E:\kmoretti-github\blog_api\api\.github\workflows\docker-publish.yml` for `main` to GHCR publishing.
- Modify: `E:\kmoretti-github\blog_api\api\src\cmd\router\static.go` so `/panel/` resolves the image-owned panel directory without exposing or depending on the persistent data mount.
- Modify: `E:\kmoretti-github\blog_api\api\src\cmd\router\router.go` only if a small unauthenticated health endpoint is needed for Compose health checks.
- Modify: `E:\kmoretti-github\blog_api\api\web\vite.config.ts` only if the production build needs a stable output path that can be copied into `/app/panel`; keep the `/panel/` base URL.
- Modify: `E:\kmoretti-github\blog_api\api\README.md` with the requested Docker build, GHCR pull, first-run initialization, update, backup, and troubleshooting instructions. This documentation file is explicitly requested by the deployment task.

## Task 1: Define the runtime path contract

**Files:**
- Modify: `E:\kmoretti-github\blog_api\api\src\cmd\router\static.go`
- Test: `E:\kmoretti-github\blog_api\api\src\cmd\router\static_test.go` if the existing router package has a compatible test pattern; otherwise verify through the container smoke test in Task 6.

- [ ] **Step 1: Inspect existing route registration and static path behavior**

Run from `E:\kmoretti-github\blog_api\api`:

```powershell
Get-Content src\cmd\router\router.go
Get-Content src\cmd\router\register.go
Get-Content src\cmd\router\static.go
```

Confirm where `staticFileHandler` is registered and whether the handler currently treats `/panel/` as a subdirectory of the configured resource base.

- [ ] **Step 2: Add one explicit image-owned panel root**

Use a stable runtime path such as `/app/panel`, resolved by the Go process from an environment variable with a default that preserves local behavior. The implementation must continue serving normal resource paths from `cfg.Data.Resource.Path`, but route `/panel/...` to the panel root before joining the request path to the data resource root. Reject traversal and hidden paths with the existing checks.

The intended behavior is:

```text
/panel/                 -> /app/panel/index.html
/panel/assets/app.js    -> /app/panel/assets/app.js
/images/foo.webp        -> configured data resource root
```

Do not add comments. Do not expose `/app/data` as a directory listing.

- [ ] **Step 3: Run focused Go tests or compile the router package**

```powershell
go test ./src/cmd/router
```

Expected: PASS, or successful package compilation if the package has no tests.

## Task 2: Create the multi-stage Dockerfile

**Files:**
- Create: `E:\kmoretti-github\blog_api\api\Dockerfile`

- [ ] **Step 1: Add the frontend builder stage**

Use a Node image that supports the repository's pnpm lockfile and enable Corepack. Copy `web/package.json`, `web/pnpm-lock.yaml`, and `web/pnpm-workspace.yaml` before installing dependencies for layer reuse. Run `pnpm install --frozen-lockfile` and `pnpm run build`. Copy the resulting `web/dist` directory to the final image as `/app/panel`.

- [ ] **Step 2: Add the CGO-enabled Go builder stage**

Use a Go 1.25.4 Linux builder with a C compiler available. Copy `go.mod` and `go.sum`, run `go mod download`, then copy the backend source and migrations. Build `main.go` with `CGO_ENABLED=1` and `GOOS=linux`, using a release-oriented build command that produces `/out/blog-api`.

- [ ] **Step 3: Add the runtime stage**

Use a small Linux runtime image that contains the dynamically linked libraries required by the CGO SQLite binary. Set `WORKDIR /app`, copy `/out/blog-api` to `/app/blog-api`, copy migrations to `/app/migrations`, and copy the frontend to `/app/panel`. Set `PORT=10024`, `LISTEN_ADDRESS=0.0.0.0`, and the panel root environment variable only if the Go implementation defines one. Expose port `10024` and run `/app/blog-api`.

The runtime image must not copy `.env`, `data/`, `database.db`, or any secret-bearing file.

- [ ] **Step 4: Build the image for the target architecture**

```powershell
docker build --platform linux/amd64 -t blog-api:local .
```

Expected: the image builds successfully and contains `/app/blog-api`, `/app/panel`, and `/app/migrations`.

## Task 3: Create the Docker build context exclusions

**Files:**
- Create: `E:\kmoretti-github\blog_api\api\.dockerignore`

- [ ] **Step 1: Exclude local and secret material**

Include exclusions for `.git`, `.env`, `data`, `web/node_modules`, `web/dist`, frontend caches, Go build artifacts, editor directories, local logs, and database sidecar files such as `*.db-shm` and `*.db-wal`.

- [ ] **Step 2: Confirm secrets are not sent to the builder**

```powershell
docker build --no-cache --platform linux/amd64 -t blog-api:context-check .
```

Expected: the build succeeds without copying `.env` or `data` into any explicit Dockerfile stage.

## Task 4: Add Docker Compose deployment

**Files:**
- Create: `E:\kmoretti-github\blog_api\api\docker-compose.yml`

- [ ] **Step 1: Define the service and image**

Use `ghcr.io/kmoretti/blog_api:latest` as the image, set `platform: linux/amd64`, map `10024:10024`, set `restart: unless-stopped`, and mount:

```yaml
- ./data:/app/data
- ./.env:/app/.env:ro
```

Do not mount a host `panel` directory because the panel belongs to the image and must update with the image.

- [ ] **Step 2: Add a health check using a non-authenticated endpoint**

Prefer an existing public status endpoint if it does not require JWT. If every status route requires authentication, use a TCP health check supported by the selected runtime image instead of adding an unrelated API endpoint. The health check must not expose secrets or require credentials.

- [ ] **Step 3: Validate the Compose file**

```powershell
docker compose config
```

Expected: normalized Compose YAML with no interpolation or syntax errors.

## Task 5: Add the GHCR publishing workflow

**Files:**
- Create: `E:\kmoretti-github\blog_api\api\.github\workflows\docker-publish.yml`

- [ ] **Step 1: Define the workflow trigger and permissions**

Trigger on pushes to `main`. Set job permissions to `contents: read` and `packages: write`. Do not use personal access tokens or hard-code credentials.

- [ ] **Step 2: Add checkout, Buildx, and GHCR login**

Use the official checkout, setup-buildx, and login actions. Login to `ghcr.io` with `${{ github.actor }}` and `${{ secrets.GITHUB_TOKEN }}`.

- [ ] **Step 3: Build and push only amd64 latest**

Configure the metadata and build-push steps to build `linux/amd64`, use the repository-derived image name `ghcr.io/kmoretti/blog_api`, and publish the `latest` tag. The workflow must push the image and must not use commit hashes as the deployment tag.

- [ ] **Step 4: Validate the YAML shape locally**

Inspect the workflow file and, if available, run a YAML parser or the repository's existing workflow validation command. Confirm the image path matches the Compose file exactly.

## Task 6: Run local container smoke tests

**Files:**
- No source changes unless a test exposes a real defect.

- [ ] **Step 1: Prepare an isolated temporary data directory**

From `E:\kmoretti-github\blog_api\api`, create a temporary directory outside the tracked project data and copy only the example configuration files into its `config` directory. Set test credentials through a temporary `.env` file; do not reuse production secrets.

- [ ] **Step 2: Start the locally built image with the same mounts as Compose**

Use a temporary container or a temporary Compose override to mount the isolated data directory at `/app/data` and the temporary environment file at `/app/.env`.

- [ ] **Step 3: Verify backend and panel routes**

Check:

```powershell
curl.exe -i http://localhost:10024/panel/
curl.exe -i http://localhost:10024/api/public/moments/
```

Expected: `/panel/` returns the compiled panel HTML and the public API returns an HTTP response without a container crash.

- [ ] **Step 4: Verify persistence across recreation**

Confirm that the SQLite database and JSON configuration remain in the host data directory after stopping and removing the container, then start a new container with the same mounts and confirm the panel still loads.

- [ ] **Step 5: Stop and remove only test resources**

Remove the temporary container, temporary Compose project, and temporary data directory. Do not delete or alter the existing project `data` directory.

## Task 7: Run repository verification

**Files:**
- No source changes unless verification identifies an implementation defect.

- [ ] **Step 1: Run backend tests and build**

```powershell
go test ./...
go build ./...
```

Expected: both commands exit successfully.

- [ ] **Step 2: Run frontend typecheck and production build**

```powershell
pnpm install --frozen-lockfile
pnpm run build
```

Expected: the Vue typecheck and Vite build complete successfully.

- [ ] **Step 3: Rebuild the target image**

```powershell
docker build --platform linux/amd64 -t blog-api:local .
```

Expected: successful target image build.

- [ ] **Step 4: Validate Compose and workflow references**

```powershell
docker compose config
Select-String -Path .github\workflows\docker-publish.yml -Pattern 'ghcr.io/kmoretti/blog_api','linux/amd64','latest'
```

Expected: Compose validates and the workflow contains the exact registry image, target platform, and `latest` tag.

## Task 8: Update deployment documentation

**Files:**
- Modify: `E:\kmoretti-github\blog_api\api\README.md`

- [ ] **Step 1: Document first deployment**

Add commands that show how to prepare `.env`, create `data/config/system_config.json`, authenticate to GHCR if the package is private, and start the service:

```bash
mkdir -p data/config data/image
touch .env
cp system_config.example.json data/config/system_config.json
docker compose pull
docker compose up -d
```

Use Linux-compatible commands in the deployment section and explain the Windows equivalent only where needed.

- [ ] **Step 2: Document updates and rollback preparation**

Document:

```bash
docker compose exec blog-api sh -c 'test -f /app/data/database.db'
cp data/database.db data/database.db.backup
docker compose pull
docker compose up -d
```

Do not claim zero-downtime updates. Explain that `latest` is intentionally mutable and that a database backup should be made before updates.

- [ ] **Step 3: Document access and troubleshooting**

Include the panel URL, API port, common permission errors on `data`, GHCR authentication errors, and the requirement for HTTPS for PWA installation outside localhost.

## Self-Review Checklist

- [ ] The plan keeps `/app/panel` outside the `/app/data` mount.
- [ ] The Go static handler path contract is implemented before Docker Compose verification.
- [ ] The Dockerfile uses CGO and a runtime compatible with the linked SQLite binary.
- [ ] No `.env`, database, or local data is copied into the image.
- [ ] The workflow publishes only `linux/amd64` with `latest` and has minimal permissions.
- [ ] The Compose image name and workflow image name are identical.
- [ ] Verification covers backend, frontend, image, routes, persistence, and Compose syntax.
- [ ] README deployment commands match the actual files and mount paths.
