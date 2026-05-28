# Issue 04: Honor `issue_dependency` in the claim handler (Gate 1)

**Status:** `done`
**Model:** `opus`

## Parent

`.scratch/feature-pipeline/PRD.md`

## What to build

Make the claim handler refuse to dispatch an issue whose `blocked_by` dependencies are unsatisfied. The `issue_dependency` table already exists (`server/migrations/001_init.up.sql:928`) since the initial migration but is not read by any handler today — this issue gives it behavior.

The claim query (in the task service that powers `POST /api/runtime/{id}/tasks/claim`) gains a `NOT EXISTS` clause:

```sql
AND NOT EXISTS (
  SELECT 1 FROM issue_dependency d
  JOIN issue b ON d.depends_on_issue_id = b.id
  WHERE d.issue_id = i.id
    AND d.type IN ('blocks', 'blocked_by')
    AND b.status != 'done'
)
```

The `related` dependency type is non-gating and is left out of the filter.

Failure mode by design: if a blocking issue fails (status stuck in `in_progress` or moves to `failed` — never `done`), dependents stay `queued` indefinitely. No time-out, no silent auto-promote — the user must intervene. This is documented behavior, not a bug.

`SELECT ... FOR UPDATE SKIP LOCKED` semantics already in place are preserved.

## Acceptance criteria

- [x] Claim query includes the dependency-gate `NOT EXISTS` clause as described above.
- [x] Integration test (extending the existing daemon/claim test file): issue A depends on B, B not done → A is NOT returned by claim. Mark B done → next claim returns A.
- [x] Integration test: issue A depends on B and C, both done → A IS returned.
- [x] Integration test: issue A depends on B (done) and C (in_progress) → A is NOT returned.
- [x] Integration test: issue A has a `related` link to B (not done) → A IS returned (related is non-gating).
- [x] Issue with no dependency rows is dispatched as before (regression check).
- [x] No change to the `Task` payload contract — this is a query-only change.

## Blocked by

- `.scratch/feature-pipeline/issues/01-rename-project-to-feature.md`

## Comments

### Key decisions made

1. **Gate placed inside `ClaimAgentTask`, not `ListQueuedClaimCandidatesByRuntime`.** The candidate-list query enumerates queued tasks for the runtime so the service loop can try each candidate's agent. Putting the gate only in `ClaimAgentTask` keeps the spec-mandated behavior (blocked tasks are never claimed) without changing the candidate-enumeration contract; the service loop simply discovers no claim and moves on. The cost is one extra `ClaimAgentTask` call per blocked-task agent per poll cycle, which is bounded by the per-runtime candidate count and dwarfed by the `FOR UPDATE SKIP LOCKED` already paid for serialization.

2. **`atq.issue_id IS NOT NULL` short-circuits the gate for chat / quick-create tasks.** The `NOT EXISTS` would also be satisfied by `d.issue_id = NULL` (returns nothing) but the explicit guard is clearer to read and avoids surprise if future schema changes make the join behavior subtle.

3. **Integration test calls `Queries.ClaimAgentTask` directly.** Going through the HTTP `ClaimTaskByRuntime` path would also exercise `CountRunningTasks`, agent capacity, and runtime routing — none of which the dependency gate touches. Calling the SQL directly isolates the assertion to the SQL claim behavior; existing tests cover the HTTP wrapper.

4. **Each subtest uses a fresh agent.** `ClaimAgentTask` enforces per-(issue, agent) serialization (one active task per pair). Reusing an agent across scenarios would let earlier subtests' dispatched tasks bleed into later assertions; per-scenario agents keep each subtest hermetic.

### Files changed

- `server/pkg/db/queries/agent.sql` — Added the `NOT EXISTS (… issue_dependency …)` clause inside `ClaimAgentTask`, plus a comment block explaining the gate and the `related`-is-non-gating rule.
- `server/pkg/db/generated/agent.sql.go` — Regenerated via `sqlc generate`.
- `server/internal/handler/claim_dependency_gate_test.go` — New file. Six subtests covering each acceptance-criteria case (blocks pending, blocks all done, blocks mixed, related non-gating, no deps regression, blocked_by alias).

### Blockers or notes for next iteration

None — all six acceptance criteria satisfied:
- Handler tests pass (774 total) including the six new dependency-gate subtests.
- Service tests pass (61 total).
- Frontend typecheck and Vitest tests pass.
- The `Task` payload contract is unchanged (this is a query-only change).
- Existing unrelated Go test failures on the local Windows environment (`local_skills_test.go`, `redact_test.go`, `repocache/*`, etc.) are pre-existing and untouched by this issue — they depend on POSIX-style paths, symlink permissions, and homedir layout that don't exist on the local machine.
