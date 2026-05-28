# Issue 05: Branch coordination gate + `TargetBranch`/`IsSharedBranch` on Task payload

**Status:** `done`
**Model:** `opus`

## Parent

`.scratch/feature-pipeline/PRD.md`

## What to build

Two coupled changes on the claim path:

**1. Branch coordination gate (Gate 2)** — prevent two queued tasks from being dispatched concurrently if they would target the same git branch. This is what makes `feature.target_branch` actually serialize execution.

The claim query gains a second `NOT EXISTS`:

```sql
AND NOT EXISTS (
  SELECT 1 FROM agent_task_queue t2
  JOIN issue i2 ON t2.issue_id = i2.id
  LEFT JOIN feature f2 ON i2.feature_id = f2.id
  WHERE t2.status = 'dispatched'
    AND t2.id != t.id
    AND COALESCE(f2.target_branch, i2.metadata->>'target_branch', 'issue/' || i2.identifier)
        = COALESCE(f.target_branch, i.metadata->>'target_branch', 'issue/' || i.identifier)
)
```

The `COALESCE` chain mirrors Issue 02's branch resolver — the parity test in Issue 02 guards against drift.

**2. Task payload extension** — add two fields to `server/internal/daemon/types.go::Task`:

```go
TargetBranch   string // resolved branch name (always set)
IsSharedBranch bool   // true when branch came from feature.target_branch
```

The claim handler populates them by calling `feature.Resolve(issue, feature)` from Issue 02 right after the SQL SELECT returns. The daemon consumes them in Issue 06.

## Acceptance criteria

- [ ] Claim query includes the branch-gate `NOT EXISTS` clause.
- [ ] `Task` struct in `daemon/types.go` has `TargetBranch` (string) and `IsSharedBranch` (bool) fields, JSON-tagged.
- [ ] Claim handler populates both fields using `feature.Resolve` from Issue 02.
- [ ] Integration test: two queued issues under the same feature with `target_branch` set → first claim returns one; second claim returns nothing (until first transitions out of `dispatched`).
- [ ] Integration test: two queued issues under different features (different branches) → both claimable concurrently.
- [ ] Integration test: issue under feature with `target_branch` NULL and no per-issue override → derived branch `issue/MUL-XXX`, `IsSharedBranch=false`.
- [ ] Integration test: issue under feature with `target_branch` set → that branch, `IsSharedBranch=true`.
- [ ] Existing claim tests still pass (regression).

## Blocked by

- `.scratch/feature-pipeline/issues/02-feature-target-branch-and-resolver.md`
- `.scratch/feature-pipeline/issues/04-claim-honors-issue-dependency.md`

## Comments

### Key decisions made

1. **Scalar subquery for the current task's branch.** Initial implementation joined `issue`/`workspace`/`feature` directly to `atq` in the candidate select. Postgres rejected this with `FOR UPDATE cannot be applied to the nullable side of an outer join` (SQLSTATE 0A000) — `LEFT JOIN feature` makes the issue/workspace side nullable from the planner's perspective even though they're inner-joined further down. Refactored the gate so the candidate select stays exactly as it was (no extra joins on `atq`), and the current task's resolved branch is computed inside the `NOT EXISTS` via a correlated `SELECT … FROM issue i JOIN workspace w … LEFT JOIN feature f WHERE i.id = atq.issue_id`. The `t2` side still joins normally because it's not the row being locked.

2. **Identifier built from `workspace.issue_prefix || '-' || issue.number`.** The PRD's SQL stub used `i.identifier`, but there's no `identifier` column on `issue` — `IssueResponse.Identifier` is computed at the handler boundary (`issueToResponse` in `issue.go`). The branch gate SQL builds the same string by joining `workspace` and concatenating. The Go side uses `Handler.getIssuePrefix(workspaceID) + "-" + strconv.Itoa(int(issue.Number))`. The branch parity test (Issue 02) keeps the two formulas in sync.

3. **`NULLIF(... , '')` in the SQL COALESCE chain.** Mirrors the empty-string handling in `feature.Resolve` (Issue 02), so a feature with `target_branch=''` falls through to the per-issue override and not to a literal empty branch name. The parity test from Issue 02 already covered this case.

4. **Branch resolution lives on the issue-task path only.** Chat / autopilot / quick-create tasks have no `issue_id`, so the gate's `atq.issue_id IS NOT NULL` short-circuit leaves them claimable, and the daemon-side population only runs inside the `if task.IssueID.Valid` block. `TargetBranch` stays empty for non-issue tasks, which is the correct shape — the daemon's branch logic in Issue 06 only fires for issue tasks anyway.

5. **Each test subtest uses its own pair of agents.** Per-(issue, agent) serialization fires when one agent has a dispatched task. With one agent and two queued tasks on different issues, both gates collapse to a single condition and you can't tell which one rejected. Two agents isolate the assertion to the branch gate.

### Files changed

- `server/pkg/db/queries/agent.sql` — Added the branch-gate `NOT EXISTS` clause inside `ClaimAgentTask`, plus a comment block explaining the gate and the empty-string handling.
- `server/pkg/db/generated/agent.sql.go` — Regenerated via `sqlc generate`.
- `server/internal/daemon/types.go` — Added `TargetBranch` and `IsSharedBranch` fields to the `Task` struct with JSON tags + doc comments.
- `server/internal/handler/agent.go` — Mirrored the two fields on `AgentTaskResponse` so the wire shape matches.
- `server/internal/handler/daemon.go` — Imported `server/internal/feature`, loaded the feature struct into a `feature.FeatureForBranch` inside the issue block, and called `feature.Resolve` to populate `resp.TargetBranch` and `resp.IsSharedBranch` once the issue + (optional) feature were available.
- `server/internal/handler/claim_branch_gate_test.go` — New file. Four subtests over `Queries.ClaimAgentTask` (shared feature branch serializes, distinct branches don't, no-feature unaffected, NULL `target_branch` unaffected) plus three subtests over `ClaimTaskByRuntime` HTTP (shared/derived/per-issue-override payloads).

### Blockers or notes for next iteration

None — all eight acceptance criteria satisfied:
- Claim query includes the branch-gate `NOT EXISTS`.
- `Task` and `AgentTaskResponse` carry `TargetBranch` and `IsSharedBranch` with JSON tags.
- Claim handler populates both via `feature.Resolve`.
- Integration tests cover all four acceptance scenarios + a per-issue-override variant.
- Existing claim tests pass (32 claim-related tests, 783 handler tests total, 61 service tests, 17 feature/branch tests, all green).
- Frontend `pnpm typecheck` and `pnpm test` pass (666 frontend tests).

Pre-existing Windows-specific failures in `repocache/*` and `execenv/TestPrepareOpenclawConfigExpandsTilde` are documented in Issue 04's notes; they depend on POSIX path/symlink semantics and are unchanged by this issue.
