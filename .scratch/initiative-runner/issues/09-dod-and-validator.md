# Issue 09: Definition of Done and validator Runs at Milestone boundaries

**Status:** `done`
**Model:** `opus`

## Parent

`.scratch/initiative-runner/PRD.md` — see ADR-0007.

## What to build

Add the **Definition of Done**: initiative-level assertions, tagged to Milestones (Acceptance Criteria is
the per-Issue view). At each Milestone boundary, dispatch a **validator Run** (`role=validator`, clean
context, a different Agent than the implementing worker) that checks the Milestone's accumulated work
against its DoD assertions. On failure, create a follow-up Issue and keep the Gate holding until the DoD
is green. Add the **DoD evaluation** deep module: `milestoneSatisfied(assertions, validatorResults)`.

Model `opus`: this is the core safety mechanism and a TDD'd deep module.

## Acceptance criteria

- [x] DoD assertion table; assertions tagged to Milestones; per-Issue Acceptance Criteria derived from them
- [x] Validator Run dispatched at a Milestone boundary with clean context, distinct from the worker
- [x] On a failing assertion, a follow-up Issue is created and the next Milestone stays gated
- [x] On all assertions green, the Milestone is marked validated and the Gate opens
- [x] TDD: DoD evaluation has failing-first unit tests, then green
- [x] `go test ./...` and `pnpm test` pass (only pre-existing env-specific Go failures remain)

## Blocked by

- `07-milestone-and-gate`
- `08-handoff-store`

## Comments

### Key decisions

1. **DoD evaluation is the pure deep module (the TDD focus).** `server/internal/dod`
   (`MilestoneSatisfied` / `FailedAssertions`) and its lockstep TS mirror
   `packages/core/dod` were both written failing-first → green. Semantics: an assertion
   passes only with ≥1 verdict and none failing; a Milestone with no assertions is
   vacuously satisfied; stray verdicts for unknown assertions are ignored;
   `FailedAssertions` preserves input order (drives follow-up creation).

2. **Two new tables (migration 009).** `dod_assertion` (workspace/feature/milestone,
   text, position) tags each assertion to a Milestone; `dod_assertion_result` records a
   validator Run's per-assertion verdict (passed + detail), keyed to the run. Re-validation
   appends rows; `ListLatestDodResultsByMilestone` (`DISTINCT ON (assertion_id) … ORDER BY
   created_at DESC`) takes the newest verdict per assertion as authoritative.

3. **Validator dispatch is a thin deterministic boundary trigger; the brain stays in
   issue 10.** `dispatchValidatorOnMilestoneBoundary` (best-effort side-effect wired beside
   `advanceInitiativeOnIssueDone` at all three issue-done sites) fires when the last open
   Issue of a not-yet-passed Milestone reaches done: it marks an assertion-free Milestone
   `passed` directly, else dispatches one validator Run via the reusable
   `TaskService.DispatchValidatorRun` (`role=validator`, `force_fresh_session=TRUE` for clean
   context, on the Milestone's Issue). `CountActiveValidatorRunsByMilestone` prevents
   duplicate validators. `resolveValidatorAgent` picks a non-archived agent other than the
   worker (`GetValidatorAgent`), falling back to the worker with a fresh session — the
   creator-verifier separation is enforced by clean context, distinct-agent is best-effort.

4. **Completion path records verdicts and evaluates.** `recordValidationOnCompletion`
   (wired into `CompleteTask` after `writeHandoffOnCompletion`, guarded to `role=validator`)
   persists verdicts then runs `evaluateMilestoneDoD`: on green → `SetMilestoneValidationStatus`
   `passed` (opens `gate.MilestoneGateOpen` for the next Milestone); on failure → `failed`
   (keeps the next Milestone gated) + a follow-up Issue (`CreateDodFollowUpIssue`, assigned to
   the worker agent, status `backlog`) so the Milestone work stays non-done until redone and
   re-validated. The follow-up uses the canonical `IncrementIssueCounter` allocator.

5. **`TaskCompleteRequest.Validation`** is the inbound contract for verdicts. The daemon→server
   forwarding of validator output is intentionally left to issue 12 (validator execution /
   sub-agent fan-out), where the validator agent actually emits results; the server mechanism
   and handler tests post it directly, mirroring how the handoff field was staged.

6. **Read surface (API-compatibility rule):** `GET /api/milestones/{id}/dod` (assertions +
   latest pass/fail/pending status — the monitor view, issue 17) and `GET /api/issues/{id}/dod`
   (the per-Issue Acceptance-Criteria derivation). Both run through `DodAssertionSchema` /
   `parseWithFallback` with malformed-response tests.

### Files changed

**Go (new)**
- `server/migrations/009_dod.{up,down}.sql`
- `server/internal/dod/{dod.go,dod_test.go}` — pure DoD evaluation (TDD)
- `server/internal/handler/{dod.go,dod_test.go}` — validator dispatch, verdict recording,
  follow-up creation, read endpoints + DB-integrated tests
- `server/pkg/db/queries/dod.sql` (+ regenerated `generated/dod.sql.go`, `models.go`)

**Go (modified)**
- `server/pkg/db/queries/agent.sql` — `CreateValidatorTask`, `CountActiveValidatorRunsByMilestone`,
  `GetValidatorAgent`
- `server/pkg/db/queries/issue.sql` — `CountNonDoneMilestoneSiblings`, `CreateDodFollowUpIssue`
- `server/internal/service/task.go` — `DispatchValidatorRun`
- `server/internal/handler/daemon.go` — `TaskCompleteRequest.Validation`, calls `recordValidationOnCompletion`
- `server/internal/handler/issue.go` (×2) + `github.go` — wire `dispatchValidatorOnMilestoneBoundary`
- `server/cmd/server/router.go` — `/api/milestones/{id}/dod`, `/api/issues/{id}/dod`

**TypeScript**
- `packages/core/dod/{index.ts,dod.test.ts,queries.ts}` — pure eval mirror + query options
- `packages/core/types/dod.ts` (+ `types/index.ts` export)
- `packages/core/api/schemas.ts` (+ `schemas.test.ts`) — `DodAssertionSchema`,
  `ListDodAssertionsResponseSchema`, malformed tests
- `packages/core/api/client.ts` — `listMilestoneDoD`, `listIssueDoD`
- `packages/core/package.json` — `./dod`, `./dod/queries` exports

### Blockers / notes

- `pnpm typecheck`, `pnpm test` (677 views + core, incl. new DoD eval + schema tests),
  `go test ./internal/dod ./internal/handler ./internal/service ./internal/gate ./internal/initiative`
  (836 passed), `go vet ./...`, `go build ./...` all green. Migration 009 applied to the shared
  dev/test DB. Remaining full-suite Go failures (`TestWebSocketIntegration`, openclaw tilde,
  missing opencode/kiro/kimi/hermes binaries) are pre-existing and environment-specific,
  unrelated to this change.
- **Test gotcha:** shared `makeIssue` fixtures allocate `number` via `MAX(number)+1`, bypassing
  the workspace `issue_counter`, so the counter lags `MAX`. Counter-based allocation (the
  production path, used by the follow-up) then collides on `uq_issue_workspace_number`. Tests
  that allocate through the counter must `syncIssueCounter()` first.
- **For issue 10 (Orchestrator):** `DispatchValidatorRun`, `recordValidationOnCompletion`, and the
  `dod` module are the reusable mechanism. The orchestrator should move the *decision* of when to
  dispatch a validator / create a follow-up into the prompt, calling these instead of the hardcoded
  `dispatchValidatorOnMilestoneBoundary` trigger. Auto-enqueuing the follow-up Issue's worker Run is
  left to the orchestrator (issue 09 creates the Issue but does not dispatch it).
- **For issue 12 (validator fan-out):** wire the daemon `TaskResult` → `TaskCompleteRequest.Validation`
  so the validator agent's per-assertion verdicts reach `recordValidationOnCompletion` in production.
- **For issue 17 (monitor):** `milestoneDodOptions` / `issueDodOptions` and the `/dod` endpoints feed
  the DoD pass/fail display.
