# Issue 03: Heartbeat with quiet detection (carries backend last_activity_at)

**Status:** `done`
**Model:** `opus`

## Parent

`.scratch/web-execution-monitor/PRD.md`

## What to build

The vertical slice that gives a running card a real **heartbeat** — the last observed activity and how fresh it is — and flips it to an amber **"quiet Ns"** when the agent goes silent, the "did it stall?" signal the owner lacks today. It cuts through the Go backend, the schema, the core derivation, and the view.

- **Backend:** the daemon stamps `last_activity_at` on the `agent_task` when it reports a `task:message`, and the workspace agent-task-snapshot returns it. The schema change folds into the consolidated init (not a standalone migration), consistent with the fork's one-init decision.
- **Frontend:** `deriveLiveness` computes `heartbeat: "fresh" | "quiet"` and a monotonically growing `quietMs` from `now - last_activity_at`; a one-second `useNow()` ticker recomputes freshness on the client between server events, so a silent agent visibly accrues idle time. The card renders the heartbeat line (last activity) and "quiet Ns" when stale. A task with no activity yet (`last_activity_at` null) is handled gracefully.

This respects the hard rule that WS events invalidate queries and never write to stores: freshness is derived from a cached server timestamp + a client ticker, with no ephemeral presence store.

## Acceptance criteria

- [ ] The daemon stamps `last_activity_at` on `agent_task` when reporting a `task:message`; the workspace task snapshot returns it. Go handler/integration test against a test database, written first (TDD).
- [ ] `deriveLiveness` returns `heartbeat: "fresh"` within the threshold and `"quiet"` beyond it, with `quietMs` growing as `now` advances. Table-driven tests with controlled `now` values.
- [ ] A `useNow()` ticker recomputes `quietMs`/`elapsedMs` ~once per second on the client.
- [ ] The running card renders the last-activity heartbeat line and an amber "quiet Ns" when stale; a null `last_activity_at` renders without crashing.
- [ ] View render test: fresh → heartbeat shown; stale → "quiet Ns" shown.

## Blocked by

- Issue 01 (establishes `deriveLiveness` and the live card surface).

## Comments

### Key decisions

1. **Backend `last_activity_at` is the single freshness source.** Added a
   nullable `last_activity_at timestamptz` column to `agent_task_queue` (folded
   into the consolidated `001_init.up.sql`, not a standalone migration, per the
   fork's one-init decision). A new `StampTaskActivity` sqlc query sets it to
   `now()` and is scoped to `status IN ('dispatched','running','waiting_local_directory')`
   so a late message can't stamp a finished run. `ReportTaskMessages` calls it
   **once per batch** after persisting/broadcasting (every message in a batch
   shares the same `now`). `taskToResponse` surfaces it as `last_activity_at`
   on the snapshot, so the board reads it from the existing React Query cache —
   no ephemeral presence store, respecting "WS invalidates, never writes".

2. **`deriveLiveness` heartbeat is pure.** `quietMs = max(0, now - baseline)`
   where `baseline = last_activity_at ?? started_at` — falling back to
   `started_at` keeps a freshly-claimed run reading `fresh` rather than stalled,
   and both-null yields `quietMs: 0`. `heartbeat = quietMs > QUIET_THRESHOLD_MS ? "quiet" : "fresh"`.
   `QUIET_THRESHOLD_MS = 10_000` is an exported, named constant (surfaces a
   *possible* stall, not a failure — long tool calls can exceed shorter windows).
   Clock skew clamps to ≥ 0.

3. **Client ticker (`useNow`) is gated on liveness.** `useNow(enabled, 1000)`
   only starts its 1s interval when the card actually has a live task, so a
   dense board doesn't re-render every card every second. `useIssueLiveness`
   computes `pickLiveTask` first (no `now` needed), then ticks only if a task
   exists, so `quietMs`/`elapsedMs` advance between server events.

4. **Heartbeat rendering** — a small `Heartbeat` sub-component renders a pulsing
   brand dot + "now" when fresh, or an amber (`bg-warning`/`text-warning`) dot +
   `quiet {Ns}s` when quiet (`Math.floor(quietMs/1000)`). Semantic tokens only.

### Files changed

- `server/migrations/001_init.up.sql` — `last_activity_at` column on `agent_task_queue`
- `server/pkg/db/queries/agent.sql` — `StampTaskActivity` query
- `server/pkg/db/generated/*` — regenerated via `sqlc generate`
- `server/internal/handler/daemon.go` — stamp activity once per `task:message` batch
- `server/internal/handler/agent.go` — `LastActivityAt` on `AgentTaskResponse` + `taskToResponse`
- `server/internal/handler/daemon_test.go` — `TestReportTaskMessagesStampsLastActivity`
- `packages/core/types/agent.ts` — `last_activity_at?: string | null` on `AgentTask`
- `packages/core/tasks/derive-liveness.ts` — `QUIET_THRESHOLD_MS`, `deriveQuietMs`, heartbeat/quietMs
- `packages/core/tasks/derive-liveness.test.ts` — 7 new heartbeat tests (29 total)
- `packages/views/issues/components/board-card.tsx` — `useNow`, gated ticker, `Heartbeat`
- `packages/views/issues/components/board-card-live-layer.test.tsx` — 3 new heartbeat tests (15 total)

### Verification

- `pnpm --filter @multica/core test` → 348 passed; `@multica/views` → 710 passed.
- Go: `TestReportTaskMessagesStampsLastActivity` passes against the test DB
  (the local DB was brought in line with the edited init via an idempotent
  `ALTER TABLE ... ADD COLUMN IF NOT EXISTS`; CI runs migrations fresh).
- `pnpm typecheck`: all changed packages pass. The only failures are
  pre-existing type errors in the **untracked** throwaway prototype
  `apps/web/app/prototype/issues-monitor/page.tsx`, which issue 08 deletes —
  not touched by this issue.

### Notes for next iteration

- Issue 04 (counters) renders beneath this heartbeat using `started_at` +
  `task:message` timeline; the live layer already has the slot.
- Issue 05 (waiting block) fills `deriveLiveness.waiting` from `wait_reason`.
- The duplicate `useNowTick` helpers elsewhere (runtimes, autopilots) were left
  alone — consolidating them is a broad refactor outside this slice.
