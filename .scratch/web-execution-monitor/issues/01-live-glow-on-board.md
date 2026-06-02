# Issue 01: Live glow + shimmer + column live-count on the issues board

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/web-execution-monitor/PRD.md`

## What to build

The thinnest end-to-end slice that turns the issues board into a live execution surface, and the foundation every later board slice builds on. On the board, a card whose issue has an **actively running** agent task renders a live decoration — an accent ring + soft glow and a pulse — and an **indeterminate shimmer** (never a 0–100% bar). When the task goes terminal (`completed`/`failed`/`cancelled`), the card drops all live decorations. Cards move between columns purely off the existing WebSocket `task:*` / `issue:*` invalidation — no new polling, no new transport.

This slice establishes the pure module **`deriveLiveness(task, now)`** with its full return shape but only the branches this slice needs (`active`, `phase: "run"`, terminal → inactive); later slices extend it. The board card and the "In Progress" column header read liveness from this one module rather than re-deriving from raw task fields.

The return shape (from the throwaway prototype `apps/web/app/prototype/issues-monitor/page.tsx`):

```ts
type LivenessPhase = "claim" | "run" | "push" | "pr";
interface Liveness {
  active: boolean;                 // running OR waiting_local_directory
  phase: LivenessPhase;
  heartbeat: "fresh" | "quiet";    // populated by issue 03
  quietMs: number;                 // populated by issue 03
  elapsedMs: number;
  waiting: { reason: string; holderKey: string | null } | null; // populated by issue 05
}
```

## Acceptance criteria

- [ ] A pure `deriveLiveness(task, now)` module exists in `packages/core` returning the shape above; a running task → `active: true, phase: "run"`; a terminal task → `active: false`. Table-driven tests written first (TDD).
- [ ] A board card whose issue has a running agent task renders the glow/pulse + indeterminate shimmer; a card with no active task renders none.
- [ ] Liveness is sourced from the existing workspace agent-task-snapshot / per-issue task caches, invalidated by the existing WS `task:*` events — no new polling.
- [ ] The "In Progress" column header shows a count of cards that are live.
- [ ] A task transitioning to terminal causes the card to drop its live decorations promptly.
- [ ] View render test (jsdom): running task → decorations present; terminal task → absent.

## Blocked by

- None - can start immediately.

## Comments

### Key decisions

1. **`deriveLiveness` is pure** — lives in `packages/core/tasks/derive-liveness.ts`, takes `(task, now)` and returns the full `Liveness` shape. Issue 01 implements the `active` flag, `phase` (run/claim), and `elapsedMs`; `heartbeat`/`quietMs`/`waiting` stub to safe defaults for later slices.

2. **`BoardCardLiveLayer` is a pure rendering component** — accepts `liveness: Liveness | null` as a prop, no data fetching. `BoardCardContent` calls `useIssueLiveness(issueId, wsId)` once and passes the result to both the glow class on the card div and down to `BoardCardLiveLayer`. No duplicate snapshot queries per card.

3. **`pickLiveTask`** — prefers `running` over `waiting_local_directory`; returns null for terminal statuses. Keeps derivation logic out of the hook.

4. **"In Progress" live count** — `BoardColumn` calls `useLiveIssueCount(wsId, issueIds)` (queries the same deduplicated snapshot) and passes `liveCount` to `BoardGroupHeading → StatusHeading` only when `status === "in_progress"`. A pulse dot + count badge renders when `liveCount > 0`.

5. **Animation** — `animate-task-progress-sweep` added to `packages/ui/styles/base.css` (same pattern as the existing nav-progress-sweep).

### Files changed

- `packages/core/tasks/derive-liveness.ts` — new pure module
- `packages/core/tasks/derive-liveness.test.ts` — 17 table-driven tests
- `packages/core/tasks/index.ts` — re-exports
- `packages/core/package.json` — exports `./tasks` and `./tasks/derive-liveness`
- `packages/views/issues/components/board-card.tsx` — `pickLiveTask`, `useIssueLiveness`, `BoardCardLiveLayer`, glow ring on card
- `packages/views/issues/components/board-card-live-layer.test.tsx` — 4 unit tests (pure props, no QueryClient)
- `packages/views/issues/components/board-column.tsx` — `useLiveIssueCount`, `liveCount` wired to `BoardGroupHeading`
- `packages/views/issues/components/status-heading.tsx` — optional `liveCount` prop + pulse badge
- `packages/ui/styles/base.css` — `animate-task-progress-sweep` keyframe + utility

### Notes for next iteration

- Issue 02 (phase stepper) extends `deriveLiveness` phase mapping (queued/dispatched → claim, PR presence → pr)
- Issue 03 (heartbeat) adds `last_activity_at` to backend + wires `heartbeat`/`quietMs` from `deriveLiveness`
- Issue 05 (waiting block) fills `deriveLiveness.waiting` from `wait_reason`
