# Issue 08: `.multica` umbrella namespacing

**Status:** `done`
**Model:** `sonnet`

## Parent

PRD 2 — Workspace In-Place Execution Mode (`.scratch/upstream-sync/PRD.md`).

## What to build

Make the umbrella's `.multica/` directory hold both the workspace manifest (`workspace.toml`) and the daemon's in-place runtime context under a distinct child path, so they coexist by design and never clobber. Ensure the in-place run writes its runtime context (the same context the daemon already writes for worktree runs) under a child of the umbrella's `.multica` that does not collide with `workspace.toml`, and confirm the reconciler and walk-up resolution read only the manifest and ignore the runtime-context sibling. Document the convention in the project guide so the manifest and runtime cache never drift into conflict.

## Acceptance criteria

- [x] In-place runtime context is written under a child path of the umbrella's `.multica` that does not overwrite or interfere with `workspace.toml`.
- [x] The reconciler / walk-up resolution read only the manifest and are unaffected by the presence of the runtime-context sibling.
- [x] The multi-repo manifest/reconciler tests still pass with an in-place runtime-context directory present.
- [x] The `.multica` namespacing convention is documented in the project guide (`CLAUDE.md`).

## Blocked by

- Issue 06 (Daemon in-place execution wiring) — the in-place run is what writes runtime context into the umbrella.

## Comments

### Key decisions

- All four acceptance criteria were satisfied by the developer of Issue 06, who implemented the full namespacing convention as part of that slice.
- **AC1:** `execenv.InPlaceFeatureDir = ".multica/feature"` (exported constant in `server/internal/daemon/execenv/context.go`) is the single source of truth for the runtime-context sub-path. `writeFeatureResources` writes `resources.json` there, which is structurally distinct from `workspace.toml` one level up.
- **AC2:** `manifest.Find()` stat()s `.multica/workspace.toml` by exact filename; it never scans the directory. The reconciler receives a pre-parsed `Manifest` struct and performs no disk I/O of its own.
- **AC3:** Two table-driven test cases in `manifest_test.go` — "in-place runtime context sibling does not interfere" and "in-place runtime context sibling, start inside sub-dir" — explicitly verify that a `.multica/feature/resources.json` sibling does not affect Find(). All 34 manifest + reconcile tests pass.
- **AC4:** `CLAUDE.md` already contains a `.multica Directory Layout` section (added by Issue 06) with the table of occupants, the `InPlaceFeatureDir` constant reference, and the resolver behavior rule.

### Files changed

No new changes required. All work was already in place:
- `server/internal/daemon/execenv/context.go` — `InPlaceFeatureDir` constant + `writeFeatureResources`
- `server/internal/workspace/manifest/manifest_test.go` — sibling-interference test cases
- `CLAUDE.md` — `.multica Directory Layout` section

### Blockers / notes

None. The `TestPrepareOpenclawConfigExpandsTilde` test failure in the execenv suite is a pre-existing Windows-environment issue (openclaw `~` expansion requires a native path that isn't present in CI's temp dirs), documented in Issue 06's notes — unrelated to this slice.
