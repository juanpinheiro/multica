# Issue 04: Honest activity counters on the live card

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/web-execution-monitor/PRD.md`

## What to build

Give a running card honest, **monotonic** counters that measure work already done — not work remaining, which is unknowable. A pure module **`deriveActivityCounters`** computes the counts from signals that actually exist (the streamed `task:message` timeline — counts of `tool_use` / edit events) plus `elapsedMs` from `started_at`. No invented "commits" number the backend does not report. The card renders the counts + elapsed beneath the heartbeat.

## Acceptance criteria

- [x] A pure `deriveActivityCounters` module counts `tool_use` / edit events from a timeline and ignores non-activity event types; `elapsed` derives from `started_at`. Table-driven tests written first (TDD).
- [x] The running card renders the activity count and elapsed time.
- [x] Counters never decrease across re-renders for a given task.
- [x] View render test asserting counts render for a task with a representative timeline.

## Blocked by

- Issue 01 (establishes `deriveLiveness` and the live card surface).

## Comments

### Key decisions

1. **`deriveActivityCounters` is pure** — lives in `packages/core/tasks/derive-activity-counters.ts`, accepts `(messages, startedAt, now)` and returns `{ activityCount, elapsedMs }`. Only `tool_use` messages count as activity; `text`, `thinking`, `tool_result`, `error` are ignored. `elapsedMs` mirrors `deriveLiveness` (same `started_at` + `now` derivation, clamped to ≥ 0).

2. **Monotonicity is structural** — the TQ cache for `["task-messages", taskId]` is append-only (WS `task:message` events are deduplicated by `seq` and sorted in `useRealtimeSync`). Because the message list only grows, `activityCount` can only grow, satisfying the "never decrease" criterion without any extra bookkeeping.

3. **`useIssueLiveness` → `useIssueLiveState`** — renamed to signal the expanded return value `{ liveness, counters }`. The hook now also queries `taskMessagesOptions(task?.id ?? "")`, which self-disables when there's no live task (via `isTaskMessageTaskId` inside the options factory). A single `useNow` ticker drives both derivations, avoiding double-interval overhead.

4. **`formatElapsed` is local to `board-card.tsx`** — display formatting is a view concern, not a core derivation concern. Under 60s shows `Ns`, under 60m shows `Nm`, beyond shows `Nh Nm`.

5. **`BoardCardLiveLayer` accepts `counters?: ActivityCounters | null`** — optional with a `null` default so all existing tests pass unchanged; the counter block renders only when `counters` is non-null and the card is active.

### Files changed

- `packages/core/tasks/derive-activity-counters.ts` — new pure module
- `packages/core/tasks/derive-activity-counters.test.ts` — 11 table-driven tests (TDD-first)
- `packages/core/tasks/index.ts` — re-exports `derive-activity-counters`
- `packages/core/package.json` — added `./tasks/derive-activity-counters` export path
- `packages/views/issues/components/board-card.tsx` — renamed hook, messages query, `formatElapsed`, counter display in `BoardCardLiveLayer`
- `packages/views/issues/components/board-card-live-layer.test.tsx` — 5 new view tests (22 total; 17 pass)

### Notes for next iteration

- Issue 05 (waiting block) fills `deriveLiveness.waiting` from `wait_reason`; uses the same live layer slot.
- Issue 08 (delete prototype) removes the pre-existing typecheck failures in `apps/web/app/prototype/issues-monitor/page.tsx`.
