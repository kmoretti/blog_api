# Moment Extension Card Editing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Allow editing, replacing, and clearing the single extension card attached to an existing moment.

**Architecture:** Reuse the existing `ExtensionEditor` in the edit dialog. Improve its controlled-value synchronization so existing JSON payloads refill the type-specific form, and preserve an explicit empty-string update so the backend clears the database value. Keep the existing single-card JSON storage format and five supported card types.

**Tech Stack:** Vue 3, TypeScript, Element Plus, Go/Gin API, pnpm/Vite.

---

### Task 1: Make the extension editor refill existing card fields

**Files:**
- Modify: `web/src/views/moments-extension/editor/ExtensionEditor.vue`

- [ ] Add a reset helper that clears all type-specific form fields.
- [ ] Add a helper that copies a parsed extension payload into the matching form object.
- [ ] Run the existing frontend production build after the change.

### Task 2: Connect the editor to the moment edit dialog

**Files:**
- Modify: `web/src/views/Moments.vue`

- [ ] Replace the edit-dialog renderer-only extension field with `ExtensionEditor v-model="editForm.extension"`.
- [ ] Pass the edit form value directly in the update payload, including `''` when the user removes the card.
- [ ] Reset the composer extension after successful creation if the current create flow does not already do so.
- [ ] Keep the list item extension synchronized after saving.

### Task 3: Ensure the backend accepts explicit extension clearing

**Files:**
- Modify: `src/model/request.go` only if request semantics need adjustment.
- Modify: `src/handler/action/moment.go` only if the existing pointer update path does not persist `''`.

- [ ] Verify the existing `*string` request field and update map preserve an empty string.
- [ ] Avoid changing the database schema or adding a migration.

### Task 4: Verify the feature

**Files:**
- Test: existing frontend build and Go test suite.

- [ ] Run `pnpm run build` from `web/`.
- [ ] Run `go test ./...` from the repository root.
- [ ] Run `git diff --check`.
- [ ] Review the final diff for single-card behavior, payload refill, replacement, and clearing.
