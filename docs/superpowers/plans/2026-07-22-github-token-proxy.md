# GitHub Token Proxy Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move GitHub repository metadata requests behind the Go API so `GH_TOKEN` stays server-side while GitHub cards receive authenticated rate limits.

**Architecture:** Add a public, read-only repository metadata endpoint that accepts only an `owner/repository` path, builds the upstream GitHub API URL from parsed path segments, and forwards the server-only `GH_TOKEN` when configured. The frontend GitHub card calls this same-origin endpoint and retains its existing URL-based fallback when the proxy or GitHub API fails.

**Tech Stack:** Go, Gin, `net/http`, Vue 3, TypeScript, Viper/.env, pnpm/Vite.

---

### Task 1: Add the server-side GitHub token configuration

**Files:**
- Modify: `src/config/config.go` only if centralized config access is required.
- Modify: `.env.example`
- Modify: `README.md`

- [ ] Add `GH_TOKEN=` to `.env.example` with a note that it must never be exposed in frontend build variables.
- [ ] Read the token with `os.Getenv("GITHUB_TOKEN")` at request time so Compose `env_file` injection works without adding it to the public JSON config.
- [ ] Document the token scope and deployment restart requirement without including a real token.

### Task 2: Implement the public GitHub repository proxy

**Files:**
- Create: `src/handler/public/github.go`
- Create: `src/handler/public/github_test.go`
- Modify: `src/cmd/router/register.go`

- [ ] Add `GET /api/public/github/repository/:owner/:repo`.
- [ ] Reject owner/repo segments containing characters outside `[A-Za-z0-9_.-]` with HTTP 400.
- [ ] Strip an optional `.git` suffix from the repository segment before calling GitHub.
- [ ] Build `https://api.github.com/repos/{owner}/{repo}` using fixed path segments, never a user-provided full URL.
- [ ] Add `Accept: application/vnd.github+json`, `X-GitHub-Api-Version: 2022-11-28`, and `Authorization: Bearer <token>` only when `GH_TOKEN` is non-empty.
- [ ] Use an HTTP client timeout of 10 seconds.
- [ ] Return the upstream JSON body and status for successful GitHub responses, map upstream 404/403/429 to the same status without exposing the token, and map transport failures to HTTP 502.
- [ ] Add tests for path validation, `.git` normalization, missing token header behavior, token header behavior, upstream success passthrough, and upstream failure mapping.

### Task 3: Switch the frontend GitHub card to the same-origin proxy

**Files:**
- Modify: `web/src/views/moments-extension/cards/GithubCard.vue`

- [ ] Replace direct `https://api.github.com/repos/${path}` requests with `/api/public/github/repository/${owner}/${repo}`.
- [ ] Continue to parse only valid GitHub owner/repository URLs before requesting.
- [ ] Keep response fields compatible with the existing `GithubRepository` type.
- [ ] Preserve repository cache, request de-duplication, and URL fallback behavior.
- [ ] Ensure no `GH_TOKEN` or authorization header is present in frontend code.

### Task 4: Verify and document deployment

**Files:**
- Modify: `.env.example`
- Modify: `README.md`

- [ ] Document server configuration:

```dotenv
GH_TOKEN=github_pat_xxxxxxxxxxxxxxxxxxxx
```

- [ ] Document that the token should use the smallest possible read-only scope and that `.env` must remain outside Git.
- [ ] Document the restart/update commands:

```bash
docker compose up -d blog-api
docker compose restart blog-api
```

- [ ] Run `go test ./...`.
- [ ] Run `pnpm run build`.
- [ ] Run `git diff --check`.
