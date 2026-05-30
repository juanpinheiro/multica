# Issue 04: Execution-mode resolution

**Status:** `done`
**Model:** `sonnet`

## Parent

PRD 2 — Workspace In-Place Execution Mode (`.scratch/upstream-sync/PRD.md`).

## What to build

A pure decision module that, given the workspace's execution mode, the resolved repo, and the issue, returns how and where a task should run — extending (not replacing) the multi-repo branch resolver. For a `worktree` workspace it yields the existing behavior: an isolated worktree on the derived `feature/<slug>` branch, parallel-eligible. For an `in_place` workspace it yields the umbrella directory as the working directory with the same `feature/<slug>` branch to be prepared in the real repo, serial-eligible. The branch name is identical in both modes; only the location and the parallel/serial eligibility differ. No I/O — it operates over data so it can be unit-tested in isolation.

## Acceptance criteria

- [ ] Worktree workspace → isolated-worktree result with derived `feature/<slug>` branch, marked parallel-eligible.
- [ ] In-place workspace → umbrella-directory result with the same derived `feature/<slug>` branch, marked serial-eligible.
- [ ] Branch name is independent of mode and matches the multi-repo derivation.
- [ ] Pure: no DB/filesystem/network; deterministic over inputs.
- [ ] Table-driven tests extending the existing branch-resolver tests, covering both modes and the parallel/serial flag.

## Blocked by

- Issue 01 (Workspace execution-mode attribute) — needs the mode to resolve against.

## Comments

### Key decisions

- **Placed in the `feature` package** as a second function alongside `Resolve`, making the extension natural: `Plan` calls `Resolve` internally for branch derivation, then adds mode-aware location and parallelism. No new package needed.
- **`Location` is a typed string constant** (`LocationWorktree` / `LocationUmbrella`), consistent with the `execmode` package's string-constant pattern. The string values match the mode vocabulary (`"worktree"` / `"umbrella"`), making log output self-describing.
- **Unknown mode falls back to worktree** — consistent with `execmode.Normalize`'s behavior. `Plan` does not re-normalize (the caller should have normalized already); it simply treats anything that isn't `execmode.InPlace` as worktree.
- **`UmbrellaDir` is empty for worktree results** — callers don't need to branch on `Location` to detect whether an umbrella path was set; empty string is the zero value and unambiguous.
- **Pure function** — no I/O of any kind; all inputs are values. The umbrella directory is passed in, not looked up.

### Files changed

- `server/internal/feature/target.go` (new) — `Location` type + constants, `RunTarget` struct, `Plan` function.
- `server/internal/feature/target_test.go` (new) — 9 table-driven cases covering both modes, metadata overrides, nil feature, unknown mode fallback, plus two cross-cutting tests: `TestPlan_BranchIndependentOfMode` and `TestPlan_BranchIndependentOfRepo`.

### Verification

- `go test ./internal/feature/` — 23 passed (includes all prior branch resolver tests + 11 new).
- `go test ./internal/workspace/... ./internal/feature/...` — 95 passed across 7 packages.
- `go vet ./...` + `go build ./...` clean.
