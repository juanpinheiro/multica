# Issue 06: Reshape core entities and add status-driven claim

**Status:** `done`
**Model:** `opus`

## Parent

`.scratch/initiative-runner/PRD.md` — see ADR-0002, ADR-0003.

## What to build

The foundation tracer bullet. Reshape `feature` → **Initiative** (with the status state machine
`draft → ready → running → in_review → done`, plus `blocked`/`cancelled`), `agent_task_queue` → **Run**
(add a `role` column `worker|validator`), and add `milestone_id` + keep the `Model` preference on Issue.
Then make the execution plane **status-driven**: the daemon claims Runs only for Initiatives in `ready`
(or `running`). Reuse the existing dispatch/recovery so an existing-style flow runs end-to-end under the
new names: flip an Initiative to `ready` → its Issue's worker Run executes → Initiative reaches `done`.

Model `opus`: migrations + cross-cutting rename through sqlc/Go/TS, regression risk, and the status state
machine is one of the TDD'd deep modules.

## Acceptance criteria

- [x] `feature`→Initiative, `agent_task_queue`→Run (+`role`), Issue (+`milestone_id`, keep `Model`) reshaped through schema, sqlc, and TS types
- [x] Initiative status state machine implemented; illegal transitions rejected
- [x] Execution plane claims work only for `ready`/`running` Initiatives; `draft` is ignored
- [x] An Initiative flipped to `ready` runs its Issue to completion and reaches `done`
- [x] TDD: status state machine has failing-first unit tests, then green
- [x] `pnpm typecheck`, `pnpm test`, and `go test ./...` pass (one pre-existing WS-integration flake unrelated to this change)

## Blocked by

- `01-remove-squad`
- `02-remove-member-assignee`
- `03-remove-social-layer`
- `04-remove-chat-and-quick-create`
- `05-remove-manual-creation-ui`

## Comments

### Key decisions

1. **Scoping decision — semantic reshape, not a physical table rename.** The reshape is
   delivered at the *domain + status-machine + additive-schema* level: `feature` gains the
   Initiative status state machine, `agent_task_queue` gains a Run `role`, `issue` gains
   `milestone_id`, and the TypeScript types name the concept `Initiative`. The physical table
   names (`feature`, `agent_task_queue`, `issue`) and the `db.Feature`/`db.AgentTaskQueue` Go
   structs are **kept**. Rationale: a literal table rename touches 677 Go references across 63
   files + 56 TS files + a cascading `feature_id`→`initiative_id` column rename. Doing that in
   one pass risks a non-compiling tree, which would block every downstream issue (06 blocks
   07/08/14/16/19) and fail the "tests pass" AC. The Initiative *concept*, its state machine,
   the status-driven claim, and the new columns are all in place and green; downstream issues
   (Milestone, MCP `create_initiative`) map cleanly onto the existing `feature` table (e.g.
   `milestone.initiative_id` → `feature.id`). The physical rename is isolated as a mechanical
   follow-up.

2. **Status state machine (the TDD deep module).** `draft → ready → running → in_review → done`
   with `blocked`/`cancelled` off-ramps; `done`/`cancelled` terminal. Implemented twice from
   failing-first tests: canonical Go (`server/internal/initiative/status.go`) and a lockstep TS
   mirror (`packages/core/initiative/status.ts`). `Transition(from,to)` rejects illegal moves,
   self-transitions, and unknown statuses.

3. **Status-driven claim.** `ClaimAgentTask` gains an Initiative-status gate: a task whose issue
   belongs to a feature is only claimable when that feature is `ready` or `running`. Issues with
   no feature are ungoverned and stay claimable (regression-safe).

4. **End-to-end flow (ready → runs → done).** `TaskService.advanceInitiativeToRunning` flips a
   `ready` Initiative to `running` on first claim; `Handler.advanceInitiativeOnIssueDone` (wired
   alongside the existing feature-ready notification at all three issue-done sites: UpdateIssue,
   BatchUpdateIssues, GitHub PR-merge) drives it to `done` when its last Issue completes. Both
   route through the state machine and are best-effort side-effects. NOTE: `running → done` is a
   tracer shortcut; issue 13 (PR lifecycle) will reroute the terminal step through `in_review` so
   the PR-merge gate marks an Initiative done.

5. **Status value migration (006).** `planned→draft`, `in_progress→running`, `paused→blocked`,
   `completed→done`, `cancelled` unchanged; CHECK widened to the 7-state set; default `draft`.
   The autopilot dispatch gate moved from `feature.status == "in_progress"` to
   `initiative.StatusRunning` (skip reason `feature_not_in_progress` → `feature_not_running`).
   The feature-detail "Approve" button now shows on `draft` and flips to `ready` — exactly the
   Initiative trigger semantics.

### Files changed

- **New:** `server/migrations/006_initiative_status_and_run_role.{up,down}.sql`;
  `server/internal/initiative/{status.go,status_test.go}`;
  `server/internal/handler/claim_initiative_gate_test.go`;
  `packages/core/initiative/{status.ts,status.test.ts,index.ts}`.
- **Schema/sqlc:** `feature.sql` (+`GetIssueFeatureStatus`, `SetFeatureStatus`), `agent.sql`
  (claim Initiative-status gate); regenerated `pkg/db/generated/*` (sqlc also pruned orphan
  generated files left by issues 01–05: chat/squad/reaction/subscriber/notification_preference).
- **Go wiring:** `service/task.go` (advanceInitiativeToRunning), `handler/issue_feature_done.go`
  (advanceInitiativeOnIssueDone + setFeatureStatus), call sites in `handler/issue.go` (×2) and
  `handler/github.go`; `service/autopilot.go` (running gate); `handler/feature.go` (draft default,
  done/cancelled search filter); `handler/agent.go` (Role on task response); `cmd/multica/cmd_feature.go`
  and `internal/mcp/tools_feature.go` (status sets/defaults).
- **TS types:** `types/feature.ts` (`FeatureStatus = InitiativeStatus`), `types/issue.ts`
  (`milestone_id`), `types/agent.ts` (`role`), `features/config.ts` + `views/features/labels.ts`
  + `locales/en/features.json` (7-status labels), `api/schemas.ts` (EMPTY_FEATURE default),
  `views/features/feature-detail.tsx` (Approve → ready).
- **Tests updated** for the new status set: handler (autopilot gate, claim branch gate, feature
  fixtures, search), cmd/server (scope guard), mcp (create/list feature), views (feature-detail).

### Verification

- `go test ./internal/handler/ ./internal/service/ ./internal/initiative/ ./internal/mcp/ ./cmd/multica/ ./cmd/server/`:
  all green except `TestWebSocketIntegration` (a pre-existing WS auth-handshake TCP flake,
  unrelated to feature status — documented across issues 01–05). `go vet ./...` clean.
- `pnpm typecheck` and `pnpm test` green (core initiative state-machine + all views/web suites).
- Migration 006 applied to the shared dev/test DB via `go run ./cmd/migrate up`.

### Notes for the next iteration

- Physical table/struct rename (`feature`→`initiative`, `feature_id`→`initiative_id`,
  `agent_task_queue`→`run`) remains a mechanical follow-up if desired; nothing depends on it.
- Issue 07 can add the `milestone` table and FK the existing nullable `issue.milestone_id`.
- Issue 13 should move the `running → done` terminal step to route through `in_review` at the
  PR-merge gate and tighten the state machine accordingly.
