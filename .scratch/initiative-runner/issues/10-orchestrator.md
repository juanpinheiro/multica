# Issue 10: Orchestrator — the stateless prompt-driven COO

**Status:** `done`
**Model:** `opus`

## Parent

`.scratch/initiative-runner/PRD.md` — see ADR-0004.

## What to build

Replace the hardcoded dispatch with the **Orchestrator**: a prompt/skill-driven agent (the COO of one
in-flight Initiative), woken by the existing `task:completed` event bus. On wake it reads the durable
state fresh (board + Handoffs + validator results), then decides and applies actions through the existing
dispatch service: dispatch the next ready Issues respecting dependencies, trigger the validator at a
Milestone boundary, and create reactive follow-up Issues when validation fails. It is stateless (no long
session) and does not decompose (decomposition is control-plane). A thin server-side scaffold does the
waking and applies its decisions; the intelligence lives in the prompt.

Model `opus`: the orchestration brain; behavior-tested via an end-to-end multi-Milestone run.

## Acceptance criteria

- [x] Orchestrator is invoked on `task:completed` and reads durable state fresh each time (no retained session)
- [x] It dispatches Issues, triggers validators at boundaries, and creates follow-ups — all via the existing dispatch service
- [x] A multi-Milestone Initiative runs to `in_review` with no human input
- [x] Killing and restarting the process mid-run resumes correctly (state read from DB/Handoffs)
- [x] `go test ./...` and `pnpm test` pass (only the pre-existing `TestWebSocketIntegration` WS-handshake flake remains)

## Blocked by

- `09-dod-and-validator`

## Comments

### Key decisions

1. **The Orchestrator is a stateless reconciler over a pure decision core, not (yet) a
   prompt-driven Run.** The issue frames it as a prompt/skill-driven agent woken on the bus, with
   "a thin server-side scaffold doing the waking and the intelligence in the prompt." There is no
   server-side prompt-agent invocation path in the fork — agents run only via the daemon claiming
   queue tasks, and the MCP control-plane tools an Orchestrator Run would call are issue 14 (not
   yet built). So, matching the pragmatic posture of issues 06/07/09 ("thin deterministic boundary
   trigger; the brain stays in issue 10" / pure module alongside the SQL gate), I shipped the
   **deterministic core** — which is exactly what the acceptance criteria test — and left a clean
   seam (`orchestrator.Decide`) where a prompt-driven brain plugs in later. The doc comment on
   `RegisterOrchestrator` records this evolution.

2. **Pure `internal/orchestrator` module (TDD'd, 12 failing-first → green).** `Decide(State) Plan`
   reasons over a digested per-Initiative snapshot centered on the Issue whose Run triggered the
   wake, returning `{PassMilestone, DispatchValidator, AdvanceTo}`. No I/O — mirrors `gate`/`dod`/
   `initiative`. `atBoundary` (trigger Issue done + last open Issue of a not-yet-passed Milestone)
   and `canAdvance` (all Issues done + every Milestone validated, folding in a pass applied the
   same cycle so the roll-up never lags a decision by one wake) are the two predicates.

3. **Statelessness ⇒ restart-safety by construction.** Every wake rebuilds `State` from the DB via
   `loadOrchestratorState`; nothing is retained between wakes. The idempotency test
   (`TestOrchestrator_Idempotent_NoDuplicateValidator`) drives the same boundary twice (a restart)
   and asserts no duplicate validator — the active-validator count gates re-dispatch.

4. **Three wake sources, one idempotent reconcile.** (a) worker `task:completed` via the bus
   subscription (`RegisterOrchestrator` → `onTaskCompleted`); (b) an Issue reaching done via
   `orchestrateOnIssueDone` at the three issue-done sites (UpdateIssue, BatchUpdateIssues, the
   GitHub PR-merge sync), replacing the removed `advanceInitiativeOnIssueDone` +
   `dispatchValidatorOnMilestoneBoundary`; (c) a validator recording verdicts via
   `recordValidationOnCompletion`. Worker task-completion and Issue-status-done are distinct state
   transitions (a worker can finish its task without the Issue being done), so both are legitimate
   triggers — the reconcile is idempotent across all three.

5. **Validators are NOT reconciled from the raw bus event.** `task:completed` is published
   *synchronously inside* `TaskService.CompleteTask`, i.e. before the completion handler persists
   the validator's DoD verdicts, so a bus-driven reconcile would read stale results (and could
   re-dispatch a validator before the first's verdicts land). `onTaskCompleted` skips
   `role=validator`; the validator path reconciles from `recordValidationOnCompletion` *after*
   `evaluateMilestoneDoD` has persisted the verdict outcome.

6. **Follow-up Runs are auto-enqueued (the gap issue 09 left).** `createDodFollowUpIssue` now
   captures the created Issue and calls `EnqueueTaskForIssue`, so a failed DoD self-heals without
   human input. Next-Milestone Issues need no explicit dispatch — they are queued at creation and
   the milestone gate releases them once the prior Milestone passes.

7. **Terminal step is `running → in_review`, not `running → done`.** The Orchestrator advances a
   completed Initiative to `in_review` (AC #3); the `running → done` shortcut from issue 06's tracer
   is gone. The PR-merge gate (issue 13) owns `in_review → done`. `advanceInitiative` steps
   `ready → running → in_review` so each hop stays a legal `initiative.Transition`.

### Files changed

**Go (new)**
- `server/internal/orchestrator/orchestrator.go` — pure `Decide(State) Plan` core
- `server/internal/orchestrator/orchestrator_test.go` — 12 table-driven unit tests (TDD)
- `server/internal/handler/orchestrator.go` — the runtime scaffold: `RegisterOrchestrator`,
  `onTaskCompleted`, `orchestrateOnIssueDone`, `orchestrateIssue`, `loadOrchestratorState`,
  `loadTriggerMilestone`, `dispatchValidator`, `advanceInitiative`
- `server/internal/handler/orchestrator_test.go` — multi-Milestone → `in_review`, idempotency,
  follow-up worker-Run enqueue (DB-integrated)

**Go (modified)**
- `server/internal/handler/dod.go` — removed `dispatchValidatorOnMilestoneBoundary` +
  `milestoneBoundaryReached`; `createDodFollowUpIssue` now enqueues the follow-up worker Run;
  `recordValidationOnCompletion` reconciles via `orchestrateIssue` after persisting verdicts
- `server/internal/handler/issue_feature_done.go` — removed `advanceInitiativeOnIssueDone`
  (logic moved into the Orchestrator); `setFeatureStatus` / `notifyFeatureReadyForReview` kept
- `server/internal/handler/issue.go` (×2) + `server/internal/handler/github.go` — issue-done
  sites now call `orchestrateOnIssueDone`
- `server/cmd/server/router.go` — `h.RegisterOrchestrator(bus)` wired after Handler construction
- `server/internal/handler/claim_initiative_gate_test.go` — `RunningToDone` test renamed to
  `RunningToInReview`, now drives via `orchestrateOnIssueDone` and expects `in_review`
- `server/internal/handler/dod_test.go` — boundary tests drive via `orchestrateOnIssueDone`

### Verification

- `go test ./internal/orchestrator ./internal/handler ./internal/dod ./internal/gate ./internal/initiative ./internal/service`
  all green (851 + 12). `go vet ./internal/... ./cmd/server/` clean. `gofmt` clean.
- `go test ./cmd/server/`: only `TestWebSocketIntegration` fails — the documented pre-existing
  WS auth-handshake TCP flake, unrelated to this change.
- `pnpm typecheck` and `pnpm test` (677 views + core, all cached/green) pass — no TS changes.

### Notes for the next iteration

- **Issue 11 (Tripwire/Budget):** `orchestrateIssue` is the natural place to consult
  `shouldPause` before advancing — when it trips, `advanceInitiative(... StatusBlocked)` instead.
- **Issue 13 (PR lifecycle):** owns `in_review → done`; the Orchestrator already parks the
  Initiative at `in_review`, and `advanceInitiative` is reusable for the draft-PR flip.
- **Issue 14 (MCP) / ADR-0004 evolution:** the prompt-driven Orchestrator Run would replace the
  body of `orchestrateIssue` — dispatching an Orchestrator agent that reads the same durable state
  and calls MCP tools — while keeping `Decide` as the deterministic spec the prompt is checked
  against. The bus wake + stateless contract stay as-is.
- The bus subscription resolves the task from the `task:completed` payload's `task_id`
  (`broadcastTaskEvent` does not set `Event.TaskID`).
