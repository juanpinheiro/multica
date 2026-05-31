# Issue 01: Workspace execution-mode attribute

**Status:** `done`
**Model:** `opus`

## Parent

PRD 2 — Workspace In-Place Execution Mode (`.scratch/upstream-sync/PRD.md`).

## What to build

Introduce a per-workspace **execution mode** with two values, `worktree` (default) and `in_place`, declared only in the workspace manifest and projected onto the server's workspace record. This slice is the end-to-end plumbing for the field — no execution behavior changes yet (the daemon still runs worktree mode regardless), but the mode is parsed, stored, reconciled, and returned by the API.

End-to-end path: the manifest gains a top-level `mode` key; the manifest parser reads it (defaulting to `worktree` when absent); the workspace record carries the mode (folded into the consolidated init schema); the reconciler sets the server's mode from the manifest on session start; the workspace API response includes it. At the moment a workspace is created or switched to `in_place`, the setup/reconcile flow surfaces a one-line note that in-place execution is serial.

```toml
# .multica/workspace.toml
workspace = "meu-produto"
mode = "in_place"   # default "worktree"
```

## Acceptance criteria

- [x] Manifest parsing reads `mode`; absent or unknown value defaults to `worktree` (unknown logged, not fatal).
- [x] Workspace schema (consolidated init) holds the mode; default `worktree`.
- [x] Reconciler projects the manifest `mode` onto the server workspace on session start; a wiped DB rebuilds the mode from the manifest.
- [x] Workspace API response includes the mode.
- [x] Creating or switching a workspace to `in_place` surfaces a note that execution is serial.
- [x] Tests cover: default when absent, explicit `in_place`, unknown value falls back to `worktree`, reconciler sets the mode.

## Blocked by

None — can start immediately.

## Comments

### Key decisions

- **New `execmode` package as the single normalization point.** Created `server/internal/workspace/execmode` holding the `Worktree`/`InPlace` constants and `Normalize(raw) (mode, known)`. Every layer (manifest→reconcile, create handler, update handler) routes through it, so the "absent/unknown → worktree" rule lives in exactly one place. `known` is false only for a non-empty unrecognized value, which is what drives the "unknown logged, not fatal" warning. This also seeds the shared vocabulary Issue 04 (execution-mode resolution) will extend.
- **Manifest keeps the raw mode string; normalization + the unknown-value warning happen at the resolve layer**, where a `warnf` is available. Parsing stays pure (`Parse(data) (Manifest, error)` unchanged signature).
- **Reconciler drives both create and switch.** `reconcile.Plan` gained `WorkspaceMode` (normalized desired) + `UpdateMode` (true when the workspace exists and the server mode differs). `WorkspaceState` gained `WorkspaceMode` (server's current). A wiped DB rebuilds via create-with-mode; a manifest change to an existing workspace projects via `UpdateWorkspaceMode`. This satisfies the explicit "creating **or switching**" requirement.
- **Serial note** is emitted via the resolver's `warnf` channel (the only output the resolver has; CLI wires it to stderr) when the resolved mode is `in_place` and the workspace was created or switched. It is deliberately silent when already in-place (not a change) and for worktree.
- **Schema:** added `mode text DEFAULT 'worktree'::text NOT NULL` inline to the consolidated `001_init.up.sql` workspace table (no standalone migration, per the fork's one-init decision). No CHECK constraint — the app normalizes on every write path, and a CHECK would reject future modes.
- **TS scope intentionally deferred.** The Go API now returns `mode`; extra response fields are ignored by the TS client. Adding `mode` to the `Workspace` TS type + UI belongs to Issue 07 (UI execution-mode indicator), which is "mode must be exposed by the API" + the read-only surface. Kept this slice backend-only to avoid touching ~5 unrelated TS fixtures.

### Files changed

- `server/internal/workspace/execmode/execmode.go` (+ `_test.go`) — new package: mode constants + `Normalize`.
- `server/internal/workspace/manifest/manifest.go` (+ `_test.go`) — `Manifest.Mode toml:"mode"` (raw).
- `server/internal/workspace/reconcile/reconcile.go` (+ `_test.go`) — `WorkspaceState.WorkspaceMode`; `Plan.WorkspaceMode`/`Plan.UpdateMode`; `Reconcile` computes them.
- `server/internal/workspace/resolve/resolve.go` (+ `_test.go`) — `Workspace.Mode`; `Server.CreateWorkspace(...,mode)` + new `Server.UpdateWorkspaceMode`; unknown-mode warning; create-with-mode / switch / serial note in `applyManifest`; `serverState` reads server mode.
- `server/migrations/001_init.up.sql` — `workspace.mode` column.
- `server/pkg/db/queries/workspace.sql` — `mode` in ListWorkspaces / CreateWorkspace / UpdateWorkspace (narg). Regenerated via `sqlc generate` (`models.go`, `workspace.sql.go`).
- `server/internal/handler/workspace.go` (+ `_test.go`) — `WorkspaceResponse.Mode`; create/update requests accept + normalize `mode`.
- `server/cmd/multica/workspace_resolve.go` + `cmd_workspace.go` — `serverAdapter` sends/reads `mode`, implements `UpdateWorkspaceMode` (PATCH); `workspaceSummary.Mode`.

### Verification

- `go build ./...`, `go vet` clean. Targeted Go suites: 1110 passed across workspace + handler + cmd/multica packages (handler tests require the running test DB).
- `pnpm typecheck` and `pnpm test` (672 view tests + others) green — no TS changes.
- Note: the long-lived local Postgres predated the schema change, so the `mode` column was synced into it with `ALTER TABLE workspace ADD COLUMN IF NOT EXISTS mode ...` (matches the migration). A fresh/CI DB gets it straight from `001_init.up.sql`.

### Notes for next iteration

- Issue 04 (execution-mode resolution) should build on the `execmode` package and extend the multi-repo branch resolver, returning the worktree-vs-umbrella location + parallel/serial eligibility.
- Issue 07 (UI) should add `mode` (and a `"worktree" | "in_place"` union) to the TS `Workspace` type and surface it read-only, with an enum-drift fallback per the API-compatibility rules.
