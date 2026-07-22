# GitHub Repository Card Ech0 Parity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the project's GitHub extension card match Ech0-5.4.6 in iconography, layout, repository metadata, and graceful fallback behavior.

**Architecture:** Keep the current extension-card shell and `github` payload contract. Add a local GitHub SVG icon, use the existing backend GitHub repository endpoint if available, and update the card to load/cache repository metadata with a safe fallback to the parsed owner/repository URL. Keep all external links guarded by the existing URL payload and avoid introducing a new card data schema.

**Tech Stack:** Vue 3, TypeScript, Element Plus, scoped CSS, existing Go API endpoint, pnpm/Vite.

---

### Task 1: Reuse or expose the existing GitHub repository API contract

**Files:**
- Modify: `web/src/api/` only if an existing GitHub repository API wrapper is missing.
- Modify: `web/src/views/moments-extension/cards/GithubCard.vue`

- [ ] Search the current project for an existing GitHub repository request and response type before adding code.
- [ ] Use the existing request wrapper if present; otherwise add a narrowly scoped wrapper for the repository metadata endpoint with fields `name`, `description`, `stargazers_count`, `forks_count`, and `owner.avatar_url`.
- [ ] Keep failed requests non-blocking so the card still renders from the repository URL.

### Task 2: Add the Ech0 GitHub icon and metadata icons

**Files:**
- Create: `web/src/components/icons/github.vue`
- Create: `web/src/components/icons/star.vue` if an equivalent local icon does not exist.
- Create: `web/src/components/icons/fork.vue` if an equivalent local icon does not exist.
- Modify: `web/src/views/moments-extension/cards/GithubCard.vue`

- [ ] Port the GitHub path from Ech0's `github-icon` symbol into a Vue SVG component using `currentColor` so light and dark themes inherit the card color.
- [ ] Reuse existing local star/fork icons when available; otherwise add minimal inline SVG components matching the Ech0 visual weight.
- [ ] Replace Element Plus `Connection` usage in both the card header and avatar fallback.

### Task 3: Match the Ech0 card layout and states

**Files:**
- Modify: `web/src/views/moments-extension/cards/GithubCard.vue`

- [ ] Render the card with the existing `ExtensionCardShell`.
- [ ] Render a circular owner avatar when API metadata contains `owner.avatar_url`; otherwise render the local GitHub icon in the avatar area.
- [ ] Render repository name, description fallback `${owner}/${repo}`, star count, and fork count when metadata is available.
- [ ] Preserve accessible external-link behavior, focus-visible styling, ellipsis/line clamping, and mobile-safe flex sizing.
- [ ] Keep the existing project theme variables instead of copying Ech0-specific variable names that do not exist here.

### Task 4: Verify the implementation

**Files:**
- Test: existing frontend build and repository tests.

- [ ] Run `pnpm run build` from `web/`.
- [ ] Run `go test ./...` from the repository root if the API wrapper touches backend contracts.
- [ ] Run `git diff --check`.
- [ ] Review the final diff for icon consistency, metadata fallback, and absence of secrets or unrelated changes.
