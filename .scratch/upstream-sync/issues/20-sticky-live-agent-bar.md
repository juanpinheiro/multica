# Issue 20: Sticky live-agent bar on issue view

**Status:** `done`
**Model:** `sonnet`

## Parent

PRD 1 — Upstream Sync (`.scratch/upstream-sync/PRD.md`).

## What to build

Add a sticky live-agent bar to the issue view, ported from upstream, so the developer can see at a glance which agents are running on the issue they're looking at, with a multi-agent accordion. Port the net final upstream state — after the follow-up that adjusted its placement into the activity section — not both revisions.

## Acceptance criteria

- [x] The issue view shows a sticky live-agent bar reflecting agents currently running on the issue.
- [x] Multiple concurrent agents are presented via the accordion.
- [x] The bar reflects the net final upstream placement, not the superseded revision.
- [x] View test covers rendering for zero, one, and multiple live agents.

## Blocked by

None — can start immediately.

## Comments

### Key decisions

- **Implementation was already present** — `AgentLiveCard` (`packages/views/issues/components/agent-live-card.tsx`) was implemented as part of the broader upstream-sync work. The component was wired into `issue-detail.tsx` at line 1936, inside the activity section immediately before the timeline, which matches the "net final upstream placement" requirement.
- **Sticky positioning** — the primary task banner uses `sticky top-4 z-10` with a backdrop-blur glass effect (`bg-background/80 supports-[backdrop-filter]:bg-background/55 backdrop-blur-md`).
- **Multi-agent accordion** — when more than one agent is active, secondary tasks collapse behind a `+N more agents running` toggle button; clicking expands them, and a second click collapses back. Running tasks sort above queued ones so the sticky slot stays on the most active task.
- **`waiting_local_directory` included** — the status rank map explicitly handles the new status (rank 2, same as queued) so the component covers all `AgentTask["status"]` values exhaustively.
- **Reconcile race guard** — a monotonic counter (`reconcileSeq`) ensures a slow stale fetch can never re-add a banner that a newer empty fetch already cleared.

### Files changed

- `packages/views/issues/components/agent-live-card.tsx` — full component implementation (sticky banner, accordion, elapsed timer, transcript/stop actions, WS subscriptions, reconcile loop)
- `packages/views/issues/components/agent-live-card.test.tsx` — 10 tests: reconcile race, reconnect self-heal, queued rendering, stop confirm dialog, sort order, and the zero/one/multiple agent rendering suite
- `packages/views/issues/components/issue-detail.tsx` — `<AgentLiveCard key={id} issueId={id} />` placed in the activity section
- `packages/views/issues/components/execution-log-section.tsx` — `waiting_local_directory` added to status maps
- `packages/views/locales/en/issues.json` — `agent_live.*` strings (is_working, is_queued, queued_elapsed_prefix, more_agents_one/other, hide_agents, etc.)

### Verification

- `pnpm --filter @multica/views exec vitest run issues/components/agent-live-card.test.tsx` — 10 passed
- `pnpm --filter @multica/views exec vitest run issues/components/issue-detail.test.tsx` — 25 passed
- `pnpm typecheck` — 4 tasks successful, 0 errors
