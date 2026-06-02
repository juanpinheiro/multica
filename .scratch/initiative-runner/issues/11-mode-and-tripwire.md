# Issue 11: Mode (HITL/AFK) and the Tripwire/Budget safety net

**Status:** `done`
**Model:** `opus`

## Parent

`.scratch/initiative-runner/PRD.md` — see ADR-0005.

## What to build

Add the Initiative **Mode** (`hitl|afk`) and a per-Initiative **budget** (token/Run/time cap) and failure
tolerance. Add the **Tripwire/Budget** deep module: `shouldPause(initiativeState)` — pause-and-ping when
the same Milestone fails validation repeatedly or the budget cap is hit. A paused Initiative moves to
`blocked` and surfaces to the human rather than burning resources. (Mode primarily records the planning
choice; the runtime safety is the tripwire.)

Model `opus`: safety-critical and a TDD'd deep module.

## Acceptance criteria

- [x] Initiative carries `mode` + budget/tolerance fields
- [x] `shouldPause` pauses on repeated same-Milestone validation failure or budget cap
- [x] A tripped Initiative transitions to `blocked` and is surfaced for human attention
- [x] TDD: Tripwire/Budget has failing-first unit tests covering each trip condition, then green
- [x] `go test ./...` and `pnpm test` pass (only pre-existing env-specific Go failures remain)

## Blocked by

- `09-dod-and-validator`

## Comments

### Key decisions

1. **Tripwire/Budget is the pure deep module (the TDD focus).** `server/internal/tripwire`
   (`ShouldPause(State) (bool, Reason)`) and its lockstep TS mirror `packages/core/tripwire`
   (`shouldPause(state) → { pause, reason }`) were both written failing-first → green. Semantics:
   a fixed precedence — **failure tolerance → token → run → time budget**. Every budget is a hard
   ceiling; a **zero (or negative) cap means "no cap"** (that dimension never trips), and a
   `failureTolerance <= 0` disables the failure tripwire. This mirrors the posture of issues
   06/07/09/10 (pure module alongside the runtime wiring).

2. **Mode + budget live on `feature` (migration 010).** `mode text NOT NULL DEFAULT 'hitl'`
   (CHECK `hitl|afk`), `budget_tokens bigint`, `budget_runs integer`, `budget_seconds bigint` (all
   DEFAULT 0 = "no cap"), and `failure_tolerance integer NOT NULL DEFAULT 3`. sqlc regenerated; the
   fields flow through `SELECT *` automatically. The REST `FeatureResponse` + TS `Feature` type +
   `FeatureSchema`/`EMPTY_FEATURE` carry them so the Mode indicator (issue 18) can read them. Mode
   stays a lenient `z.string()` so a new server value downgrades rather than crashing.

3. **The tripwire is consulted at the top of every Orchestrator reconcile.** `pauseOnTripwire`
   runs in `orchestrateIssue` before `orchestrator.Decide`: it loads the feature once, assembles the
   pure `tripwire.State`, and on a trip moves the Initiative `→ blocked` (via `initiative.Transition`)
   and raises a best-effort `initiative_tripwire` inbox alert (reusing the renamed
   `inboxItemToEventMap`), then stops the reconcile so no further Runs dispatch. Only `ready`/`running`
   Initiatives can trip; an already-paused one is not re-alerted.

4. **Sourcing the State (honest, staged).** `MaxMilestoneValidationFailures(feature)` counts the
   worst single Milestone's distinct failing validator Runs (the "repeated same-Milestone failure"
   signal); `CountRunsByFeature` feeds the Run budget. **Token and wall-clock usage are not yet
   recorded per Run**, so `TokensUsed`/`ElapsedSeconds` are sourced as 0 — those budgets stay inert
   until that tracking lands. The pure module already models all four dimensions as the spec; the
   wiring provides what exists today (same staging as issue 09's daemon→server validation forwarding).

### Files changed

**Go (new)**
- `server/migrations/010_mode_and_tripwire.{up,down}.sql`
- `server/internal/tripwire/{tripwire.go,tripwire_test.go}` — pure pause-decision (TDD, 14 cases)
- `server/internal/handler/{tripwire.go,tripwire_test.go}` — `pauseOnTripwire`, state loading,
  inbox alert + DB-integrated tests (run-budget, under-budget, failure-tolerance)

**Go (modified)**
- `server/pkg/db/queries/agent.sql` — `CountRunsByFeature`
- `server/pkg/db/queries/dod.sql` — `MaxMilestoneValidationFailures`
- regenerated `pkg/db/generated/*` (Feature struct + the two new queries)
- `server/internal/handler/orchestrator.go` — consult `pauseOnTripwire` before `Decide`
- `server/internal/handler/feature.go` — `FeatureResponse` + `featureToResponse` carry mode/budget
- `server/internal/handler/issue_feature_done.go` — `featureReadyInboxItemToMap` → `inboxItemToEventMap` (shared)

**TypeScript**
- `packages/core/tripwire/{index.ts,index.test.ts}` — pure mirror (TDD, 9 cases) + `./tripwire` export
- `packages/core/types/feature.ts` — `InitiativeMode` + the new `Feature` fields
- `packages/core/api/schemas.ts` (+ `schemas.test.ts`) — `FeatureSchema`/`EMPTY_FEATURE` defaults + tests
- `packages/views/features/components/feature-detail.test.tsx` — fixture updated for the new fields

### Blockers / notes

- `pnpm typecheck`, `pnpm test` (677 views + core, incl. new tripwire + FeatureSchema tests),
  and `go test ./internal/tripwire ./internal/handler ./internal/service ./internal/dod ./internal/gate
  ./internal/initiative ./internal/orchestrator ./internal/handoff` (all green). Migration 010 applied
  to the shared dev/test DB. Remaining full-suite Go failures (`TestWebSocketIntegration`, missing
  opencode/kiro binaries, Windows path quirks) are pre-existing and unrelated.
- **For issue 18 (monitor inbox/alerts):** the `initiative_tripwire` inbox item (severity
  `action_required`, `details.reason` + `details.mode`) is the actionable alert; the Mode indicator
  reads `feature.mode`. Blocked Initiatives surface via the existing `blocked` status.
- **For a future token/time-tracking issue:** wire `TokensUsed`/`ElapsedSeconds` in
  `loadTripwireState` once per-Run token + wall-clock are recorded; the budgets already exist on the
  entity and the pure module already enforces them.
- **For issue 14 (MCP) / issue 15 (planning skills):** the MCP `create_initiative` surface should set
  `mode` and the budget/tolerance fields (the columns default sensibly until then).
