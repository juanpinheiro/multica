# Issue 04: Multi-issue workers push directly to the shared feature branch

**Status:** `done`
**Model:** `opus`

## Parent

`.scratch/initiative-runner-autonomy/PRD.md` — Module #2.

## What to build

The intended model is one PR per Initiative on the shared `feature/<branch_slug>` branch (ADR-0005, issue 13). Single-issue Initiatives already produce a clean PR. Multi-issue Initiatives don't: the execution plane creates per-task `agent/*` worktree branches and the workers self-merge `agent → feature` PRs, cluttering the single PR with merge commits and intermediate branches.

Formalize the decided model in the execution plane: for an issue that resolves (via `feature.Resolve`) to a shared feature branch, the worker's worktree tracks and pushes to that shared branch directly, instead of creating a per-task `agent/*` branch and self-merging. The serial-within gate already guarantees only one task holds the shared branch at a time, so there is no concurrent-writer problem within a single Initiative.

`feature.Resolve` remains the single source of truth for the branch name and its contract is unchanged. The change is in the worktree branch selection / push path in the daemon (repocache worktree). The single-issue path is unchanged. Because all issues push to the same branch, per-Milestone commits stay navigable in the single PR with no extra server logic (issue 13's "structural, not code" decision).

## Acceptance criteria

- [ ] For a feature-scoped issue under a multi-issue Initiative, the worker tracks/pushes the resolved `feature/<slug>` branch directly and does NOT create a per-task `agent/*` branch.
- [ ] No `agent → feature` self-merge PR is opened for multi-issue Initiative tasks.
- [ ] The single-issue path is unchanged and still yields one clean PR.
- [ ] The resulting PR contains the per-Milestone commits in order on the shared branch.
- [ ] Tests cover the worktree branch selection on the shared-branch multi-issue path (tracks/pushes the shared branch, no `agent/*` branch created). Prior art: `branch_test.go`, `branch_parity_test.go`, `target_test.go`, repocache worktree tests.
- [ ] `make check` passes (known Windows `repocache` git-clone flakes excepted).

## Blocked by

None - can start immediately.

## Comments

### Key decisions

- **The worker's worktree is minted in exactly one place:** `repocache.CreateWorktree`
  (`server/internal/daemon/repocache/cache.go`), reached when the agent runs
  `multica repo checkout <url> --ref <branch>` against the daemon's
  `/repo/checkout` endpoint. `execenv.Prepare` starts each task with an empty
  workdir; repos are checked out on demand. So the entire change lives in
  `repocache` — no CLI, HTTP, env-var, or `feature.Resolve` changes. `feature.Resolve`'s
  contract is untouched, as the issue required.
- **Signal = the `feature/` prefix on the resolved ref**, via a new
  `isSharedFeatureBranch(ref)` predicate. `feature.Resolve` mints `feature/<slug>`
  for every feature-scoped (shared) issue and `issue/<identifier>` otherwise, and
  issue 01's `ValidateBranchSlug` rejects any slug containing `feature/` — so the
  prefix is the authoritative on-the-wire materialization of `shared=true`. This is
  the same convention `resolveBaseRef` already keys its first-Run fallback on
  (right above the changed code), so no new plumbing was introduced to thread a
  redundant `IsSharedBranch` flag through the CLI/HTTP boundary.
- **Shared branch → checked out directly, never renamed.** For a shared ref the
  worktree is created on a local branch named exactly `feature/<slug>` via
  `git worktree add --force -B <branch> <path> <baseRef>` (`runSharedWorktreeAdd`):
  `-B` resets the branch to the resolved base (origin/feature/<slug> when it exists,
  else the remote default on the first Run via the existing `resolveBaseRef`
  fallback), and `--force` lets it reuse the branch even if a *finished* sibling
  task's worktree still references it. The serial-within gate guarantees only one
  task holds the shared branch at a time, so `--force` only ever overrides a stale,
  completed worktree. Non-shared refs keep the existing `agent/<name>/<taskid>`
  branch + timestamp-rename-on-collision path verbatim.
- **`trackSharedUpstream`** sets the worktree branch's upstream to
  `origin/feature/<slug>` when that remote ref exists, so a plain `git push` lands
  on the shared branch with no `agent → feature` self-merge. On the first Run the
  origin branch doesn't exist yet — the call is a no-op and the agent's first push
  creates it (the brief's "first push opens the PR" path).
- **Why this kills the messy history:** previously every task got its own
  `agent/*` branch and self-merged into `feature/<slug>`, cluttering the single PR
  with merge commits and intermediate branches. Now all tasks of an Initiative
  commit straight onto `feature/<slug>`, so the one PR's history is just the
  per-Milestone commits in order — "structural, not code" (issue 13), no extra
  server logic.

### Files changed

- `server/internal/daemon/repocache/cache.go` — `isSharedFeatureBranch` predicate;
  `shared` branch-name selection in `CreateWorktree`; `createWorktree` /
  `updateExistingWorktree` gained a `shared` parameter (shared path resets the
  branch in place, no rename); new `runSharedWorktreeAdd` and `trackSharedUpstream`
  helpers; both call sites set upstream after create/update.
- `server/internal/daemon/repocache/cache_test.go` — three new tests
  (`TestCreateWorktreeSharedFeatureBranchFirstRun`,
  `TestCreateWorktreeSharedFeatureBranchExistingRemote`,
  `TestCreateWorktreeNonFeatureRefKeepsAgentBranch`) plus `currentBranch` /
  `upstreamOf` helpers, asserting: first-Run shared branch is checked out on
  `feature/<slug>` (not `agent/*`) based on the default branch; a subsequent Run
  resets to the remote feature head and tracks `origin/feature/<slug>`; and a
  non-feature ref keeps the isolated `agent/*` branch (single-issue/standalone
  path unchanged).

### Test results / notes

- `go build ./...` and `go vet ./internal/daemon/repocache/` clean on Windows.
- The three new tests pass. The full `repocache` package shows two intermittent
  failures (`TestCreateWorktreeRemovesCoAuthoredByHookWhenDisabled`,
  `TestCreateWorktreeFetchesDespiteAgentBranchOnRemote`) that are the known Windows
  `git clone --bare`/`git push` filesystem flakes ("failed to unlink", "eof before
  pack header") — they re-pass on retry and don't touch the changed
  branch-selection path. Treated as environment noise per the issue.
- This is a Go-only change with no TypeScript impact, so the `npm run test` /
  `npm run typecheck` loops are not exercised — the TS surface is untouched.
- No blockers for the next iteration (issue 05).
