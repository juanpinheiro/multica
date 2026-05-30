# Issue 06: Active-workspace resolution wired into CLI/MCP

**Status:** `done`
**Model:** `opus`

## Parent

`.scratch/multi-repo-features/PRD.md`

## What to build

Wire the manifest machinery (Issue 05) into the CLI/MCP so the active workspace is resolved from where the user is, with zero ceremony. This is the DX core — getting the precedence wrong is directly user-visible.

Resolution runs on each CLI/MCP session and stops at the first match:

1. **Explicit override** — `--workspace <slug>` flag or `MULTICA_WORKSPACE` env.
2. **`.multica` walk-up** — `manifest.Find` from cwd → workspace slug + repo list (offline; common case).
3. **git remote → server lookup** — the cwd repo's `origin` matched against `repo.remote_url` on the server (covers detached worktrees outside the umbrella).
4. **Single workspace** — if exactly one workspace exists, use it.
5. **Last-used** — from `~/.multica/config.toml` when outside any repo/umbrella.
6. **None** — signal that onboarding is needed (the setup skill in Issue 08 handles the scan-and-suggest; this slice just returns the "no workspace resolved" state cleanly).

When the walk-up (2) finds a manifest, run the reconciler against the server and apply the plan: create the workspace if absent (rebuild from manifest), create missing repo rows, warn on `ReposMissingOnDisk`. Never delete. The manifest's pinned `workspace` slug makes re-reads idempotent.

The resolved workspace slug feeds the existing `X-Workspace-Slug` mechanism the API client already uses. Persist the resolved slug as last-used in `~/.multica/config.toml`.

## Acceptance criteria

- [ ] Running a CLI/MCP command inside an umbrella subdirectory resolves the workspace from the `.multica` above it, offline, with no flag.
- [ ] A detached worktree (no `.multica` above it) resolves via its git remote against the server.
- [ ] With exactly one workspace, resolution returns it without needing a manifest.
- [ ] Outside any repo/umbrella, resolution falls back to last-used; `--workspace`/`MULTICA_WORKSPACE` overrides everything.
- [ ] A manifest whose workspace slug is absent on the server triggers a rebuild (workspace + repos created via the reconciler plan); a missing-on-disk repo produces a warning, not a deletion.
- [ ] Re-running in the same directory is idempotent (binds to the same workspace, creates nothing new).
- [ ] Tests cover the precedence order and the reconcile-apply path with a fake API client; `make check` passes.

## Blocked by

- Issue 05 (resolver, scanner, reconciler modules).

## Comments

### Iteration 1 — implemented (Opus)

**Key decisions**

- **New pure-ish package `internal/workspace/resolve`** holds the precedence
  engine + reconcile-apply. It depends only on three injected interfaces
  (`Server`, `FS`, `Git`), so the whole thing is unit-tested over an in-memory
  fake server + fake fs/git — no OS, no network. This is the testable heart of
  the slice (the acceptance "tests cover the precedence order and the
  reconcile-apply path with a fake API client").
- **Precedence implemented exactly as specced**: override → manifest walk-up →
  git-remote → single-workspace → last-used → none. `Resolve` returns a
  `Source` enum so the caller can tell *why* a workspace was chosen (and detect
  `SourceNone` to hand off to onboarding — Issue 08).
- **Manifest branch is offline-first.** It always returns the manifest's slug,
  even when the server is unreachable; reconcile (create workspace if absent,
  create missing repos, warn on missing-on-disk) runs best-effort and any
  server error degrades to a `Warnf` line rather than failing resolution. This
  satisfies both "resolves offline from `.multica`" and "absent slug triggers a
  rebuild" without two code paths.
- **Reconcile reuses Issue 05's pure `reconcile.Reconcile`.** `resolve` only
  gathers `WorkspaceState` (server existence + repo names + on-disk presence)
  and *applies* the returned `Plan`. Idempotency falls out for free: an in-sync
  manifest yields an empty plan, so a re-run creates nothing and binds to the
  same workspace (covered by `TestResolve_ManifestRebuildIsIdempotent`).
- **Wired into the single chokepoint `newAPIClient`** (used by every CLI
  command *and* the MCP server via `runMCP`), so CLI and MCP share one
  resolution path. Resolution runs only when there is **no explicit
  flag/env override** (`--workspace-id`/`MULTICA_WORKSPACE_ID`) and **not inside
  an agent task** (the daemon stays authoritative there — preserves the #1235
  contamination guard). Gating on the *explicit* override (not on the value
  `resolveWorkspaceID` folds in from config) is what lets the manifest win over
  a remembered config default, matching the spec precedence (manifest #2 >
  last-used #5). `resolveWorkspaceID` itself is left untouched so its existing
  direct-call tests are unaffected.
- **Resolved identity is sent as `X-Workspace-Slug`** (already supported by the
  server middleware) in addition to `X-Workspace-ID`. Added a `WorkspaceSlug`
  field to `cli.APIClient`. In the normal (online) case the manifest reconcile
  also yields the UUID, so `WorkspaceID` is populated too and ID-dependent
  commands keep working; slug-only happens only when fully offline (server
  unreachable anyway).
- **New `--workspace` / `MULTICA_WORKSPACE` override** added as the documented
  escape hatch (slug-based). The legacy `--workspace-id` / `MULTICA_WORKSPACE_ID`
  (UUID) remains the top-priority explicit override.
- **Last-used persisted** to `~/.multica/config.json` (the fork uses JSON, not
  the PRD's aspirational `config.toml`; matched the actual codebase) only when
  the resolved ID is known and changed — no config churn per command.
- **Git-remote lookup** has no dedicated server endpoint, so the CLI adapter
  iterates workspaces × repos and matches on a normalized remote
  (`normalizeRemote` drops scheme/user@/`.git`/trailing slash and canonicalizes
  scp-style `git@host:owner/repo`). The resolver itself just calls
  `Server.FindWorkspaceByRemote`, keeping the messy iteration in the thin
  (non-unit-tested) adapter.

**Files changed**

- New `server/internal/workspace/resolve/resolve.go` — `Resolve`, precedence,
  `applyManifest`/`serverState`/`diskPresence`, `Server`/`FS`/`Git` interfaces,
  `Source`/`Result`/`Inputs`/`Workspace`/`RepoInput` types.
- New `server/internal/workspace/resolve/resolve_test.go` — 10 table/case tests
  covering all six precedence rules + reconcile-apply (rebuild, idempotency,
  missing-on-disk warning, offline degradation) against the fake server.
- New `server/cmd/multica/workspace_resolve.go` — `applyResolvedWorkspace`
  wiring, `persistLastUsed`, the `serverAdapter` (REST impl of `resolve.Server`),
  `osFS`/`osGit`, and `normalizeRemote`.
- `server/internal/cli/client.go` — `WorkspaceSlug` field + `X-Workspace-Slug`
  header.
- `server/cmd/multica/cmd_agent.go` — `newAPIClient` invokes resolution when no
  explicit override and not in agent context.
- `server/cmd/multica/main.go` — `--workspace` persistent flag.

**Verification**

- `go build ./...`: clean. `go vet` on touched packages: clean. `gofmt`: clean.
- `go test ./internal/cli/... ./internal/workspace/... ./cmd/multica/...`: 303
  pass (incl. the 10 new resolve tests). No existing CLI/workspace test broke —
  the cmd tests that set `MULTICA_WORKSPACE_ID` short-circuit before resolution,
  and the two config-writing tests exercise `resolveWorkspaceID`/`runWorkspaceSwitch`
  directly (their own clients), not `newAPIClient`.
- `pnpm typecheck` (4/4) and `pnpm test` (669) green — TS untouched.
- DB-gated handler tests still skip locally (Docker down, same as Issues 01–04);
  this slice adds no handler/SQL changes, so nothing new there to verify against
  a live DB.

**Notes for the next iteration**

- Issue 08's setup skill consumes the `SourceNone` state from `Resolve` to kick
  off scan-and-suggest, and can reuse `serverAdapter.CreateWorkspace`/`CreateRepo`
  for registration after writing the manifest.
- The git-remote lookup is O(workspaces × repos) per session; if that becomes a
  cost, add a server-side `GET /api/repos?remote=` lookup and collapse
  `FindWorkspaceByRemote` to one call.
- `~/.multica/config.json` is the real config (not `config.toml`); if a future
  issue wants the PRD's TOML, it's a separate migration.
