# PRD: Web UI as a Real-Time Agent-Execution Monitor

**Status:** `ready-for-agent`
**Owner:** Juan Pinheiro
**Created:** 2026-05-31
**Depends on:** `.scratch/upstream-sync/PRD.md` (PRD 2 — the `waiting_local_directory` task status, `wait_reason`, and `work_dir` already exist on `AgentTask`; this PRD consumes them) and `.scratch/multi-repo-features/PRD.md` (workspace-as-context, the per-workspace execution `mode`, and the `.multica/workspace.toml` manifest).

## Problem Statement

The fork owner already runs the whole loop today: they talk to Claude in a terminal, Claude does the setup, writes the PRDs, decomposes them into issues, and the daemon dispatches agents that produce PRs. But that orchestration is a **black box**. Once work is handed to the agents, the owner has no real-time picture of what is happening: which agent is on which issue right now, whether it is actively working or quietly stalled, which task is parked waiting for the in-place umbrella lock, and which PRs have come back and are waiting on a human decision. The information exists — the daemon streams it over WebSocket — but the UI does not surface it as a live picture.

Worse, the web UI is still shaped like the tool it was forked from: a Linear-style app built for a **human operator** who creates issues, assigns them to themselves, organizes their backlog, manages a multi-tenant workspace, and switches between accounts. None of that matches the new reality. There is one implicit user, the human is no longer an assignee (only agents are), issues are created by Claude rather than through a "New Issue" form, and the workspace is filesystem-anchored context discovered from a manifest — not an account you create through a web form. So the chrome that frames the app — the user-identity menu, the workspace switcher-as-manager, "My Issues", the "New Issue" call-to-action — furnishes a role that no longer exists, and it crowds out the one thing the owner actually needs the web for: **seeing the agents work**.

Concretely, today: the board groups issues by status or assignee but never shows the live execution layer; the only place an active agent is surfaced richly is a sticky card on the issue *detail* page, not on the board itself; there is no honest liveness signal (no heartbeat, no "is it stalled?", no phase of the run); the per-workspace execution mode is buried as a passive badge on the feature-detail page; and the sidebar leads with identity and account chrome.

## Solution

Reframe the web UI from a human-operated planning tool into a **real-time visibility surface for agent execution**, and let the issues board be its home — because the board is where execution becomes visible as cards move while agents claim, work, and finish.

- **The issues board becomes a live execution monitor.** Columns stay status (the existing model); liveness is overlaid as a property of each card. A card whose issue has an actively running agent glows and pulses; a card whose agent is parked shows an amber `waiting_local_directory` block naming the holder; cards move between columns in real time as the daemon's `task:*` events flow in (the invalidation plumbing already exists). The owner opens the app and *sees* the fleet working.

- **Liveness is shown honestly — never as a fake percentage.** Agent task duration is unknowable and varies by orders of magnitude, so there is no 0–100% progress bar. Instead each running card shows: a discrete **phase stepper** (`claim → run → push → PR`); a **heartbeat** — the last observed activity plus its freshness, which flips to an amber "quiet Ns" when the agent goes silent (the "did it stall?" signal the owner lacks today); **monotonic counters** (activity/edits, elapsed) that measure work already done, not work remaining; and an **indeterminate shimmer** that says "active, duration unknown". Leaving a column is event-driven (a PR was actually pushed), not driven by a bar reaching 100%.

- **The dead operator chrome is removed.** The user-identity menu (one implicit user), the workspace switcher-as-account-manager, "My Issues" (no human assignee), and "New Issue" as a primary CTA (Claude creates issues) all go. The sidebar shrinks to a thin icon rail led by the Multica brand mark.

- **Project identity becomes ambient, not a destination.** A thin ambient bar shows the current workspace, its execution mode (`worktree` / `in_place`), and that it is manifest-anchored — promoting the execution mode from a buried badge to a persistent, read-only ambient signal. Switching projects is "recent projects", never "create a workspace".

The result: the owner opens the web, lands on the issues board, and immediately sees which agents are running, which are breathing vs. stalled, which are waiting on the serial in-place lock, and which PRs have come back for them — with the chrome stripped down so the work is the protagonist.

## User Stories

1. As the fork owner, I want the app to open directly on the issues board, so that the first thing I see is execution, not a setup form or a backlog I manage.
2. As the fork owner, I want a card whose issue has a running agent to visibly glow/pulse on the board, so that I can spot active work at a glance across all columns.
3. As the fork owner, I want cards to move between columns in real time as agents progress, so that I watch the fleet advance without refreshing.
4. As the fork owner, I want each running card to show a discrete phase (`claim → run → push → PR`), so that I know where in the pipeline the agent is rather than a meaningless percentage.
5. As the fork owner, I want a heartbeat on each running card showing the last activity and how fresh it is, so that I can tell the agent is actually doing something.
6. As the fork owner, I want the heartbeat to flip to an amber "quiet Ns" when an agent goes silent, so that I can catch a stalled run instead of staring at a spinner that never resolves.
7. As the fork owner, I want monotonic counters (activity/edits and elapsed) on a running card, so that I see how much work has been done so far, since how much remains is unknowable.
8. As the fork owner, I want an indeterminate shimmer rather than a filling bar while an agent runs, so that the UI never lies to me about a duration it cannot predict.
9. As the fork owner, I want a card whose agent is parked to show a `waiting_local_directory` block naming the issue that holds the umbrella lock, so that I understand why an in-place task is serial and waiting.
10. As the fork owner, I want a card to leave its column only when the underlying work actually transitions (e.g. a PR was pushed), so that movement on the board reflects real events.
11. As the fork owner, I want the board's "In Progress" column to indicate how many of its cards are live right now, so that I gauge concurrency at a column level.
12. As the fork owner, I want the sidebar to be a thin icon rail led by the Multica brand mark, so that the board gets the screen and the chrome recedes.
13. As the fork owner, I do NOT want a user-identity menu, because there is a single implicit user and there is no account to manage.
14. As the fork owner, I do NOT want a "My Issues" destination, because I am never an assignee — only agents are.
15. As the fork owner, I do NOT want a "New Issue" call-to-action in the chrome, because Claude creates issues, not a web form.
16. As the fork owner, I do NOT want the workspace switcher to offer "Create workspace", because workspaces are manifest-anchored and set up by Claude, not created through the web.
17. As the fork owner, I want the switcher to behave like "recent projects" I can jump between, so that switching is navigation, not account management.
18. As the fork owner, I want an ambient bar showing the current workspace, its execution mode, and that it is manifest-anchored, so that I always know which project I am looking at and whether it runs serially.
19. As the fork owner, I want the execution mode promoted from a hidden feature-detail badge to the persistent ambient bar, so that `in_place` vs `worktree` is always visible, not buried.
20. As the fork owner, I want an Inbox entry in the rail with a badge counting PRs/issues awaiting my decision, so that I can see at a glance when the agents need me.
21. As the fork owner, I want the rail nav to lead with Issues (home) and Features (planning), so that the execution and planning layers are the primary surfaces.
22. As the fork owner, I want the supporting surfaces (Agents, Autopilots, Skills, Usage, Settings) to remain reachable but secondary in the rail, so that they don't compete with the board.
23. As the fork owner, I want the liveness derivation to be a single, well-tested piece of logic, so that the glow, stepper, heartbeat, and waiting block all agree on one source of truth.
24. As the fork owner, I want the heartbeat to keep counting "quiet" time even when no new event arrives, so that a silent agent visibly accrues idle time instead of looking frozen at "now".
25. As the fork owner, I want a card that transitions to terminal (completed/failed/cancelled) to drop its live decorations promptly, so that the board doesn't show ghosts of finished runs.
26. As the fork owner on Windows, I want the live monitor to behave correctly regardless of how fast or slow my WSL2 daemon streams events, so that freshness is computed from real timestamps, not assumptions.

## Implementation Decisions

> Vocabulary (used verbatim): **workspace** = manifest-anchored context grouping repos; **execution mode** = `worktree` | `in_place`; **task** = an `AgentTask` dispatched to a runtime; **liveness** = the derived real-time state of a task (phase + heartbeat + waiting). The `waiting_local_directory` status, `wait_reason`, and `work_dir` already exist on `AgentTask` — this PRD consumes them and does not redefine them.

### Deep module — task liveness derivation (pure)

The centerpiece. A pure function that, given a task and the current time, returns the full real-time state the board needs. It has no I/O and is exercised entirely over data, so the glow, stepper, heartbeat label, and waiting block all read from one tested source rather than each re-deriving state from raw fields.

The return shape (encoded by the throwaway prototype at `apps/web/app/prototype/issues-monitor/page.tsx`, trimmed to the decision):

```ts
// deriveLiveness(task, now) -> Liveness
type LivenessPhase = "claim" | "run" | "push" | "pr";
interface Liveness {
  active: boolean;                 // running OR waiting_local_directory
  phase: LivenessPhase;            // queued/dispatched -> claim; running -> run; PR present/in_review -> pr
  heartbeat: "fresh" | "quiet";    // quiet once (now - lastActivityAt) exceeds the threshold
  quietMs: number;                 // how long since last observed activity (grows while silent)
  elapsedMs: number;               // since started_at
  waiting: { reason: string; holderKey: string | null } | null; // from wait_reason when waiting_local_directory
}
```

- **Phase** maps from task status (+ the PR / `in_review` signal): `queued`/`dispatched` → `claim`; `running` → `run`; a present PR or `in_review` issue status → `pr`. `push` is the transient between `run` and `pr`. A `waiting_local_directory` task is rendered as blocked at `claim`.
- **Heartbeat** is `fresh` when `now - lastActivityAt` is within a threshold and `quiet` beyond it; `quietMs` grows monotonically while no new activity arrives, which is what makes a silent agent visibly accrue idle time without any new event.
- **Terminal** tasks (`completed`/`failed`/`cancelled`) return `active: false` so the board drops live decorations.

### Heartbeat data source — backend `last_activity_at` (respects cache-as-truth)

Liveness freshness needs a timestamp of the agent's last observed activity. Rather than a high-frequency ephemeral store fed by WS (which would violate the hard rule that WS events invalidate queries and never write to stores, and would create a second source of presence truth), the **daemon stamps `last_activity_at` on the `agent_task`** when it reports a `task:message`, and the workspace agent-task-snapshot carries it. The board reads it from the React Query cache (already invalidated on `task:*` events), and a one-second `useNow()` ticker recomputes `quietMs`/`elapsedMs` against it on the client — so "quiet Ns" advances between server events without any store. The schema change folds into the fork's consolidated init, consistent with the one-init decision; it is not a standalone migration.

### Deep module — activity counters (pure)

A pure derivation of the honest, monotonic counters a running card shows (`elapsed`, and an activity/edit count) from signals that actually exist — the streamed `task:message` timeline (counts of `tool_use` / edit events) and `started_at` — rather than inventing a "commits" number the backend does not report. Isolated so the card renders counts without re-deriving them.

### Board card execution surface (Variant A)

The board card gains a live execution surface driven entirely by `deriveLiveness`. The chosen layout is **Variant A** from the prototype (liveness embedded per card, one dense board), explicitly over Variant B (a separate top "live" rail): a running card glows (accent ring + soft shadow) and renders the phase stepper, the indeterminate shimmer, the heartbeat line, and the counters; a `waiting_local_directory` card renders an amber block with the wait reason and the holding issue key; an `in_review` card notes that its PR awaits review. The existing status-grouped board, drag-drop, infinite scroll, and the other view modes are preserved; only the card's live layer is added. The "In Progress" column header shows a live count.

### Sidebar reduction and ambient project bar

The sidebar collapses to a thin icon rail led by the Multica brand mark (the favicon mark, not a project glyph). Removed: the user-identity menu, "My Issues", the "New Issue" CTA, and the workspace switcher's "Create workspace" affordance. The switcher is reframed as "recent projects" — navigation between known workspaces with no creation path. Nav order: Issues (home) · Inbox (badge = items awaiting the human) · Features · — · Agents · Autopilots · Skills · Usage · Settings. A new **`AmbientProjectBar`** renders a thin line — `workspace · mode (worktree|in_place) · manifest provenance` — promoting the execution mode from the feature-detail badge to a persistent, read-only ambient signal. The mode value comes from the workspace record (already typed; absent → treat as `worktree`).

### Real-time wiring (reuse, not rebuild)

No new event transport. The board's live layer reads the existing workspace agent-task-snapshot and the per-issue task caches, which are already invalidated by the central WS sync on `task:*` events. The only addition to the data flow is `last_activity_at` on the task and the client `useNow()` ticker for freshness.

## Testing Decisions

Everything in this PRD is built **TDD** (red-green-refactor): a failing test in the correct package first, then the implementation. A good test here asserts external behavior, not implementation detail — pure modules as table-driven unit tests over data, view behavior via render + simulated state, and the backend field over real HTTP against a test database. Tests live with the code, not the app (per the monorepo testing rule).

- **`deriveLiveness`** (pure, `packages/core/*.test.ts`). Table-driven: `queued`/`dispatched` → phase `claim`, inactive-by-status handling; `running` with fresh `last_activity_at` → phase `run`, heartbeat `fresh`; `running` with stale `last_activity_at` → heartbeat `quiet` with a growing `quietMs`; `waiting_local_directory` → `waiting` populated with `reason`/`holderKey`, blocked at `claim`; a present PR / `in_review` → phase `pr`; `completed`/`failed`/`cancelled` → `active: false`. Highest-value target. Prior art: the existing pure config/store tests in `packages/core`.
- **Activity counters** (pure, `packages/core`). Counts `tool_use`/edit events from a timeline correctly and ignores non-activity event types; elapsed derives from `started_at`.
- **Board card execution surface** (view, `packages/views/*.test.tsx`, jsdom). A running task renders the stepper, shimmer, heartbeat, and counters; a `waiting_local_directory` task renders the amber block with reason and holder key; a quiet task renders "quiet Ns"; a terminal task renders no live decorations. Prior art: existing `board-card` / `agent-live-card` view tests.
- **Sidebar** (view, `packages/views`, jsdom). The Multica brand mark renders; "My Issues", the "New Issue" CTA, the "Create workspace" affordance, and the user-identity menu are absent; the new nav order renders; the ambient bar shows the workspace mode. Prior art: existing `app-sidebar` view tests.
- **`AmbientProjectBar`** (view). Renders `worktree` vs `in_place`; an absent mode renders as `worktree`.
- **Backend `last_activity_at`** (handler/integration, Go). Reporting a `task:message` stamps `last_activity_at`; the workspace task snapshot returns it; a task with no activity yet returns a null/zero value handled gracefully. Prior art: existing task-report handler tests.

## Out of Scope

- **Variant B** (a separate top "live" rail) — evaluated in the prototype and rejected in favor of Variant A.
- **A 0–100% progress bar** — explicitly rejected; duration is unknowable.
- **Adding the `waiting_local_directory` status / `wait_reason` / `work_dir`** — already exist on `AgentTask` from the in-place PRD; this PRD only consumes them.
- **The no-workspace onboarding / manifest-first entry reframe** — the empty-state "connect your project" flow and killing the `/workspaces/new` form were discussed but not resolved (the manifest may or may not exist; setup is done by Claude). Deferred to a follow-up PRD. This PRD removes the *switcher's* create affordance and the New Issue CTA, but does not redesign the cold-start entry.
- **List / swimlane / gantt redesign** — only the board card gains the execution surface; the other view modes are preserved as-is (they may inherit the liveness atoms later).
- **Per-repo or per-issue execution mode** — mode is a workspace attribute (in-place PRD); unchanged here.
- **Stalled-task auto-recovery / cancellation policy** — the heartbeat surfaces "quiet"; acting on it automatically is out of scope.

## Further Notes

- **Provenance:** the liveness shape, the phase stepper, the heartbeat-with-quiet, the honest counters, the shimmer, and the thin-rail sidebar with the Multica mark were all validated in the throwaway prototype `apps/web/app/prototype/issues-monitor/page.tsx` (Variant A chosen over B). Fold the validated decisions into the real board card and sidebar, then **delete the prototype route**.
- **Why the board is home:** Features carry the plan; Issues carry the execution. The owner plans with Claude and watches execution on the board, so the app opens on the execution layer because that is the part that is invisible today.
- **Why honest signals over a percentage:** the owner explicitly flagged that a session can run a long time and varies wildly task to task. The UI therefore answers "is it alive, what phase, how much done, how long" — all knowable — and never "how much remains", which is not.
- **Order of execution:** land `deriveLiveness` and the activity-counters module (pure, TDD) first; add the backend `last_activity_at` field + snapshot wiring; build the board card execution surface against the tested modules; strip the sidebar and add the `AmbientProjectBar`; delete the prototype route last. The pure modules land before the card so the card imports tested code.
- **Risk:** the sharpest risk is freshness correctness across a slow/fast WSL2 event stream — the `quietMs` must grow from real `last_activity_at` timestamps on a client ticker, asserted in the `deriveLiveness` table tests with controlled `now` values.
