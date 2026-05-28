# Issue 03: Autopilot dispatch gate by `feature.status`

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/feature-pipeline/PRD.md`

## What to build

Modify the autopilot dispatch path so that issues under a feature whose status is not `in_progress` are not enqueued. This implements the user-facing ritual "approve the PRD → motor starts running" without inventing a new approval table — it reuses the existing `feature.status` column (values `planned | in_progress | paused | completed | cancelled`).

The change lives in the autopilot service's dispatch function (the function that creates an `autopilot_run` row and enqueues into `agent_task_queue`). Before enqueueing:

- Look up the target issue's feature (if any).
- If the feature exists and `feature.status != 'in_progress'`, record the dispatch as a skip on the `autopilot_run` row (the existing audit table) with a reason like `feature_not_in_progress`, and do not insert into `agent_task_queue`.
- If the issue has no feature (`feature_id IS NULL`), behavior is unchanged — enqueue as today.
- If the feature exists and is `in_progress`, behavior is unchanged — enqueue as today.

This is a server-side gate, not a UI restriction. The web dashboard's "Approve" button (Issue 12) is just a PATCH that sets `status = 'in_progress'`; it does not need special endpoints.

## Acceptance criteria

- [x] Dispatch function rejects issues whose parent feature is in any status other than `in_progress`, recording the skip on `autopilot_run`.
- [x] Dispatch function continues to enqueue issues with `feature_id = NULL` (no parent feature) as before.
- [x] Integration test covers: feature `planned` → no row in `agent_task_queue`, skip recorded; feature `in_progress` → row inserted; feature `completed` → skip recorded; issue with no feature → row inserted (unchanged).
- [x] No change to the claim handler, daemon, or task struct — this gate is dispatch-side only.

## Blocked by

- `.scratch/feature-pipeline/issues/01-rename-project-to-feature.md`

## Comments

### Key decisions made

1. **Gate placed in `shouldSkipDispatch`** — This pre-flight function is the canonical admission gate for all autopilot dispatch. Adding the feature status check here means a non-`in_progress` feature records a `skipped` run (via `recordSkippedRun`) rather than a failed run, which is the correct semantic: no work was attempted, nothing broke.

2. **Fail-open on transient DB errors** — If `GetFeature` fails for a reason other than "not found", the gate logs a warning and returns `("", false)` (no skip). This matches the pattern used by the agent readiness checks in the same function: a temporary DB hiccup should never silently swallow a scheduled run.

3. **Hard-skip on missing feature** — If `GetFeature` returns `pgx.ErrNoRows`, the dispatch is hard-skipped with reason `"autopilot feature not found"`. This matches the "assignee agent no longer exists" case: retrying will not help.

4. **Existing test updated** — `TestAutopilotCreateIssueAssociatesConfiguredProject` in `handler_test.go` previously created a feature with default status (`planned`) and then dispatched. With the gate active, the dispatch would be skipped. The INSERT was updated to `status = 'in_progress'` so the test remains valid.

### Files changed

- `server/internal/service/autopilot.go` — Added feature status gate at the top of `shouldSkipDispatch`
- `server/internal/handler/handler_test.go` — Updated `TestAutopilotCreateIssueAssociatesConfiguredProject` feature fixture to use `status='in_progress'`
- `server/internal/handler/autopilot_feature_gate_test.go` — New file: 4 integration tests covering planned/in_progress/completed/no-feature cases

### Blockers or notes for next iteration

None — all acceptance criteria satisfied. 828 handler and service tests pass.
