# Issue 02: Phase stepper on the live card (claim → run → push → pr)

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/web-execution-monitor/PRD.md`

## What to build

Extend `deriveLiveness` with the phase mapping and render a discrete **phase stepper** (`claim → run → push → pr`) on a running card, so the owner sees where in the pipeline the agent is instead of a meaningless percentage. Phase maps from task status (+ the PR / `in_review` signal): `queued`/`dispatched` → `claim`; `running` → `run`; a present PR or `in_review` issue status → `pr`; `push` is the transient between `run` and `pr`. A `waiting_local_directory` task is rendered as blocked at `claim` (its amber block is issue 05; here only the step position).

## Acceptance criteria

- [x] `deriveLiveness` returns the correct `phase` for each status / PR combination; table-driven tests written first (TDD).
- [x] The running card renders the stepper with the active step highlighted and prior steps marked reached.
- [x] A `waiting_local_directory` task shows the stepper blocked at `claim`.
- [x] View render test asserting the active step for representative tasks.

## Blocked by

- Issue 01 (establishes `deriveLiveness` and the live card surface).

## Comments

### Key decisions

1. **`deriveLiveness` extended with `ctx?: LivenessCtx`** — an optional context object `{ issueStatus?: IssueStatus; hasPr?: boolean }` passed as a third argument. `derivePhase` is a private helper that encodes the priority: `waiting_local_directory` → `claim` first (overrides even `in_review`); then `hasPr || issueStatus === "in_review"` → `pr`; then `running` → `run`; else `claim`. `push` is defined in `LivenessPhase` as a possible value but is never currently returned (it's a visual intermediate step in the stepper between `run` and `pr`).

2. **Issue status wired at the board-card level** — `useIssueLiveness` accepts a third `issueStatus?: IssueStatus` parameter and passes `{ issueStatus }` to `deriveLiveness`. The call site in `BoardCardContent` passes `issue.status`, so `in_review` issues automatically show the `pr` phase. No separate PR-fetch per card.

3. **Phase stepper in `BoardCardLiveLayer`** — renders above the shimmer bar when liveness is active. Four steps (`claim → run → push → pr`) connected by `›` separators. Active step gets `aria-current="step"` and `text-brand font-medium` styling; reached steps get `text-muted-foreground/60`; future steps get `text-muted-foreground/35`. Each step has `data-testid="phase-step-{phase}"` for test assertions.

4. **`waiting_local_directory` correctly blocks at `claim`** — the `derivePhase` check for `waiting_local_directory` runs before the `in_review` check, so a waiting task always shows at `claim` in the stepper even if the issue is also in_review.

### Files changed

- `packages/core/tasks/derive-liveness.ts` — added `LivenessCtx` interface, `derivePhase` helper, updated `deriveLiveness` signature to `(task, now, ctx?)`
- `packages/core/tasks/derive-liveness.test.ts` — 5 new table-driven tests for the extended phase mapping (22 tests total)
- `packages/views/issues/components/board-card.tsx` — `LivenessPhase` import, `PHASES` constant, `BoardCardLiveLayer` updated with phase stepper, `useIssueLiveness` accepts `issueStatus`, call site passes `issue.status`
- `packages/views/issues/components/board-card-live-layer.test.tsx` — 5 new tests for the stepper (9 tests total)

### Notes for next iteration

- Issue 03 (heartbeat) needs to wire `last_activity_at` from the backend and fill `heartbeat`/`quietMs` in `deriveLiveness`
- Issue 05 (waiting block) will add the amber block inside `BoardCardLiveLayer` using the `waiting` field already on `Liveness`
