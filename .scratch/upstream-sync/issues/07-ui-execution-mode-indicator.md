# Issue 07: UI execution-mode indicator

**Status:** `done`
**Model:** `sonnet`

## Parent

PRD 2 — Workspace In-Place Execution Mode (`.scratch/upstream-sync/PRD.md`).

## What to build

Surface the active execution mode in the UI so the developer always knows which model is in effect. The runtime/task view and the feature view display whether a run is executing in `worktree` or `in_place` mode, and a task in the waiting state renders its wait reason. This is a read-only surfacing of server state — no control to change the mode (the manifest remains the only place it is set), consistent with the rest of the design.

## Acceptance criteria

- [x] A run/task shows its execution mode (`worktree` or `in_place`) on the runtime/task view.
- [x] The feature view indicates when its workspace runs in-place (explaining why its issues run serially).
- [x] A task in `waiting_local_directory` renders its wait reason.
- [x] No UI control sets or changes the mode.
- [x] View tests: a run renders its mode; a waiting task renders its reason; enum-drift on the mode value renders a generic fallback rather than crashing.

## Blocked by

- Issue 01 (Workspace execution-mode attribute) — mode must be exposed by the API.
- Issue 05 (`waiting_local_directory` task status) — waiting state + reason must exist.

## Comments

### Key decisions

- **Mode badge on the "Now" section header, not per task row.** The execution mode is a workspace property, not per-task. Showing a badge per row would be repetitive (every row in a workspace has the same mode). A single badge on the "Now" section header communicates the mode clearly without clutter. Worktree mode (the default) shows nothing — only `in_place` gets a badge, since that's the exceptional state the developer set intentionally.
- **In-place indicator in the feature detail sidebar as a read-only PropRow.** The feature properties panel already has Status/Priority/Lead rows. Adding an "Exec: In-place" row there follows the existing pattern and lives alongside other workspace-level metadata. The row only renders when `workspace?.mode === "in_place"` — unknown or absent modes (enum drift) render nothing, which is the right fallback.
- **`isNonTerminalStatus` exported as a pure function.** Extracted from the inline `activeTasks` filter to make it testable and reusable. Also means `execution-log-section.tsx` and `agent-live-card.tsx` could adopt it in future, but left for now to avoid over-engineering.
- **`waiting_local_directory` treated as active (shown in "Now"), not terminal.** A waiting task is still in-flight — it will run once the umbrella lock releases. Showing it in "Now" with cancel action is correct behavior; it tells the developer "there is work pending, it's waiting."
- **Wait reason shown inline in the task metadata row**, truncated to 240px with a `title` attribute for the full text. Same visual position as `failureLabel`, which it's semantically analogous to for waiting tasks.
- **Transcript button hidden for `waiting_local_directory`** — same as `queued`, since no agent execution has started yet and the transcript would be empty.
- **`execution-log-section.tsx` and `agent-live-card.tsx` updated** to include `waiting_local_directory` in the exhaustive `Record<AgentTask["status"], ...>` maps — TypeScript required it after the status union was widened. Added appropriate tone (`text-muted-foreground`) and rank (2, same as `queued`).

### Files changed

- `packages/core/types/workspace.ts` — `Workspace.mode?: "worktree" | "in_place"`
- `packages/core/types/agent.ts` — `AgentTask.status` union + `wait_reason?: string | null`
- `packages/views/agents/config.ts` — `waiting_local_directory` entry with `PauseCircle` icon
- `packages/views/locales/en/agents.json` — `waiting_prefix`, `wait_reason_label`, `exec_mode_inplace_badge`, `exec_mode_inplace_tooltip`
- `packages/views/locales/en/features.json` — `exec_mode_label`, `exec_mode_inplace`, `exec_mode_inplace_tooltip`
- `packages/views/locales/en/issues.json` — `status_waiting_local_directory`
- `packages/views/agents/components/tabs/activity-tab.tsx` — `isNonTerminalStatus` export; `waiting_local_directory` in active filter; in-place badge on "Now" section; wait reason in task row; `activeTaskTimeText` case for waiting
- `packages/views/features/components/feature-detail.tsx` — in-place `PropRow` with `Layers2` icon
- `packages/views/issues/components/agent-live-card.tsx` — `waiting_local_directory: 2` in `statusRank`
- `packages/views/issues/components/execution-log-section.tsx` — `waiting_local_directory` in `STATUS_TONE` and `useStatusLabel`
- `packages/views/agents/components/tabs/activity-tab.test.ts` — 4 new `isNonTerminalStatus` tests
- `packages/views/features/components/feature-detail.test.tsx` — 3 new in-place indicator tests (show/hide/enum-drift)

### Notes for next iteration

- Issue 08 (`.multica` namespacing) can proceed independently — no UI dependency on it.
- The exec mode badge could be extended to the "Recent work" section (showing the mode under which completed tasks ran), but this would require persisting mode on the task record server-side, which is out of scope for v1.
