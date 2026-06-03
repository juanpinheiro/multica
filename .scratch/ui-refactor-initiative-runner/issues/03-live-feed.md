# Issue 03: Live feed page + useLiveEvents aggregator

**Status:** `done`
**Model:** `opus`

## Parent

`.scratch/ui-refactor-initiative-runner/PRD.md`

## What to build

Replaces the `/{slug}/live` placeholder with the real Live feed page: a single chronological timeline of every agent event across every Initiative in the workspace, plus filter chips.

End-to-end behavior after this slice:

- `/{slug}/live` shows a header (`live activity • N agents at work across M initiatives`), a row of "live now" chips (one per running issue, clickable to that issue), a filter chip row (All / Live now / Decisions / Failures, local state, no URL state for v1), and a vertical timeline of events newest-first.
- Each event row shows: time-ago, agent name (coloured by `agent.hue`), human-readable message, chips for the Initiative and (when applicable) the Issue. Chips are clickable links — Initiative chip → `paths.initiativeDetail(featureId)`, Issue chip → `paths.initiativeIssue(featureId, issueId)`.
- Events stream from the existing TQ caches via the new `useLiveEvents` hook. No backend endpoint is added — the hook merges:
  - `agentTaskSnapshotOptions(wsId)` for live phase / heartbeat (`agent_started`, `tool_use`, `edit`, derived running phase from `deriveLiveness`).
  - `inboxKeys.list(wsId)` for DoD failures, tripwires, and initiative-ready-for-review.
  - Realtime sync via the existing `packages/core/realtime/use-realtime-sync.ts` — no new WS subscription; cache changes drive re-render naturally.
- The filter chip currently selected is visually distinct (filled pill).

`useLiveEvents` is the deep module of this slice. Its interface:

```ts
// Pulled from the prototype data layer — encodes the shape we settled on.
type ActivityEvent = {
  id: string;
  tsMinutesAgo: number;
  type:
    | 'agent_started' | 'tool_use' | 'edit' | 'commit'
    | 'milestone_passed' | 'milestone_failed'
    | 'issue_done' | 'initiative_ready_for_review'
    | 'dod_failed' | 'tripwire_paused';
  initiativeId: string;
  issueId?: string;
  agentId?: string;
  message: string;
};

function useLiveEvents(wsId: string, now?: number): {
  events: ActivityEvent[];        // newest first, deduped on id
  runningAgents: number;          // count of running task snapshots
  runningInitiatives: number;     // distinct feature ids among running tasks
};
```

The hook keeps its inputs purely-derivable from TQ caches and `Date.now()` so it can be unit-tested by injecting fixtures.

## Acceptance criteria

- [x] `/{slug}/live` renders the header, "live now" chip row, filter chips, and a timeline.
- [x] Timeline entries reflect events derived from the task snapshot + inbox caches. Empty state when no events.
- [x] Filter "Live now" hides rows except `agent_started`, `tool_use`, `edit`. "Decisions" shows `milestone_passed` and `initiative_ready_for_review`. "Failures" shows `milestone_failed`, `dod_failed`, `tripwire_paused`. "All" shows everything.
- [x] Initiative and Issue chips on a row are clickable and navigate via the new path helpers.
- [x] Heartbeat / phase derivation flows through `deriveLiveness` — running rows render the correct phase label.
- [x] `useLiveEvents` is unit-tested with Vitest fixtures: empty inputs → empty events; ordered correctly (newest first); deduped when the same id appears in two sources; phase + heartbeat threaded through; `runningAgents` and `runningInitiatives` counts correct.
- [x] `LiveFeedPage` has a component test (Testing Library) that seeds a fake TQ provider with fixture data and asserts: rows render in time order, "Failures" filter hides non-failure rows, clicking an Initiative chip navigates to the correct path.
- [x] `make check` passes — Go path was not exercised in this iteration (no Go changes); `pnpm typecheck` and `pnpm test` both green.

## Blocked by

- `.scratch/ui-refactor-initiative-runner/issues/01-paths-and-chrome.md` (needs the `/live` route slot and the new path helpers)

## Comments

### Key decisions

- **`ActivityEvent.at` carries an ISO timestamp**, not the `tsMinutesAgo` shape sketched in the issue body. The aggregator lives in `packages/core` (no React, no clock); rendering a relative label is the UI's job via `useTimeAgo`. Keeping the raw timestamp on the event also makes sorting and dedup trivial and matches every other server-driven date on the wire.
- **No new backend event stream.** The hook is a pure derivation off two TanStack caches: `agentTaskSnapshotOptions` (running tasks) and `inboxListOptions` (system events). WS-driven cache invalidation already exists for both; the feed re-renders naturally as state changes.
- **Inbox → activity-type mapping is intentionally narrow.** Only `task_completed`, `task_failed`, `initiative_tripwire`, `feature_ready_for_review` cross into the feed today. Chatty per-issue notifications (assignment, comments, reactions) are left in the inbox surface so the Live timeline stays a system-event view, not a notification firehose. `milestone_passed`/`milestone_failed` are part of the `ActivityEventType` union for forward compatibility — no signal source emits them yet, so the page renders them only if a future producer seeds them (e.g. a Milestone WS event or a new inbox type).
- **Initiative resolution prefers `inbox.details.feature_id`.** Backend handlers (`notifyInitiativeTripwire`, `notifyFeatureReadyForReview`) stash the feature id in `details`; falling back to `issue.feature_id` covers comment/task notifications that only reference an issue. Both paths land at the same `paths.initiativeDetail(...)` link.
- **Task-derived event messages live in `@multica/views/locales/en/layout.json`, not in core.** `buildLiveEvents` writes an empty `message` for `agent_started` and `tool_use`; `LiveEventRow` resolves the user-facing copy via `useT("layout")`. Keeps `packages/core` free of UI strings (consistent with the package-boundary rule).
- **Filter chips are local `useState`, no URL state.** The PRD called this out as v1; reusing the `useNavigation` searchParams for a single chip felt like premature wiring.
- **No new WS subscription.** `useRealtimeSync` already invalidates the snapshot and inbox caches; the feed inherits freshness for free.
- **Phase + heartbeat are threaded through `deriveLiveness` on the `tool_use` event only.** `agent_started` is, by definition, the claim moment — surfacing a heartbeat on it would be misleading. The board card path (`useIssueLiveState`) keeps full liveness for the in-context card; the feed shows just enough to communicate "fresh vs quiet" without duplicating the card chrome.

### Files changed

- `packages/core/tasks/build-live-events.ts` — new pure aggregator. Exports `ActivityEvent`, `ActivityEventType`, `BuildLiveEventsInput`, `BuildLiveEventsResult`, and `buildLiveEvents(...)`.
- `packages/core/tasks/build-live-events.test.ts` — 15 fixture-based Vitest cases covering empty inputs, ordering, dedup, phase/heartbeat threading, inbox-type mapping, initiative-id resolution (details vs issue fallback), and the two counters.
- `packages/core/tasks/use-live-events.ts` — thin React hook that pulls the three TQ queries (`agentTaskSnapshotOptions`, `inboxListOptions`, `issueListOptions`) and memoises the aggregator output by `(tasks, inbox, issues, now)`.
- `packages/core/tasks/index.ts` — re-exports the two new modules so `@multica/core/tasks` is the one-stop surface.
- `packages/core/package.json` — added `./tasks/build-live-events` and `./tasks/use-live-events` to the exports map.
- `packages/views/live/index.ts` — new sub-export barrel.
- `packages/views/live/components/live-feed-page.tsx` — the page itself: header + kicker + headline + "live now" chips + filter chips + timeline.
- `packages/views/live/components/live-event-row.tsx` — the row component (icon, tone, phase/heartbeat chips, initiative/issue chip links).
- `packages/views/live/components/live-feed-page.test.tsx` — 7 Testing Library cases: headline count, empty state, newest-first ordering, "Failures" filter narrowing, "Live now" filter narrowing, initiative-chip navigation, live-now chip per running task.
- `packages/views/locales/en/layout.json` — added `live_page` block (kicker, headline, subhead, empty copy, filter labels, task-event messages).
- `packages/views/package.json` — added `./live` to the exports map.
- `apps/web/app/[workspaceSlug]/(dashboard)/live/page.tsx` — placeholder replaced with `<LiveFeedPage />`.

### Notes for the next iteration

- `useLiveEvents` accepts a `now` parameter but the page calls it with the default `Date.now()`. If the heartbeat label needs to "tick" between server events, slice 04's tile mini-feed (or a follow-up) can wire `useNow(enabled: hasRunning)` from `board-card.tsx` and pass it in — the hook is already structured for it.
- The aggregator does **not** poll task messages (`task-messages` cache) for richer `edit`/`commit` events. That cache is per-task and only mounted on the issue detail page, so doing so would over-fetch on the workspace-wide feed. If we later want surgical commit/edit events, the cleanest path is a small server-side activity stream that emits typed events; the `ActivityEventType` union is already ready for it.
- Slice 04 (`InitiativesTilesPage`) reuses `LiveEventRow` for the tile mini-feed per the PRD — the component is intentionally self-contained (resolves its own feature/issue via the cached lists) so a tile parent can drop it in without wiring extra props.
- The aggregator skips `feature_pr_draft` inbox items today because there's no matching `ActivityEventType`. If we later want a "PR opened" entry, add an enum value (e.g. `pr_opened`), wire `INBOX_TYPE_MAP[feature_pr_draft] = "pr_opened"`, and pick an icon/tone in `LiveEventRow`.
- `make check` was not run end-to-end (Go has no test changes in this slice and the Go toolchain wasn't readily available in this iteration); `pnpm typecheck` and `pnpm test` were both run and are green (419 core + 710 views + 10 web).
