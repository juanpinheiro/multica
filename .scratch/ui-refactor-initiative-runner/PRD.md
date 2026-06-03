# PRD: Web UI refactor for the Initiative Runner model

**Status:** `ready-for-agent`

**Parent direction:** the Initiative Runner fork. See `CONTEXT.md`, `docs/adr/0001`–`0008`, and the existing PRDs under `.scratch/initiative-runner/`, `.scratch/initiative-runner-autonomy/`, `.scratch/web-execution-monitor/`, `.scratch/multi-repo-features/`, and `.scratch/standalone-install/`. This PRD aligns the web app with the autonomous Initiative Runner the backend now executes — replacing the Linear-like workspace-first chrome with a manifest-driven, live-execution monitor.

## Problem Statement

The backend now runs the new model. The web UI still reflects the old one, so the developer sitting at their machine sees a product that contradicts what it actually is.

Concretely, today:

- Opening the app redirects to `/workspaces/new`, which is a form asking for a slug and name. In the new model there is no workspace creation by the human — workspaces are discovered from `.multica/workspace.toml` by the manifest resolver.
- The sidebar still surfaces "Issues" (a flat board of every issue across every initiative), "Features", "My Issues", and a workspace switcher dropdown. Three of those four are residue.
- There is no place to *watch* what Claude+agents are doing. The flat issue board mixes issues from all initiatives, which becomes incoherent the moment 3–5 initiatives run in parallel — exactly the case the new model enables.
- Decisions written by the retrospective Run are nested inside feature detail, so the developer cannot read the institutional memory across initiatives without already knowing which initiative produced each decision.
- Agents, Runtimes, Skills, Costs are arranged as peer top-level surfaces with the same prominence as the day-to-day execution view, even though they are inventory / status surfaces in the new model.

The product is not live, so we replace rather than layer.

## Solution

Three-level hybrid landing, slim chrome, top-level Decisions surface, and a full deletion of the workspace creation flow.

**Landing — three levels of depth**, each answering a different question:

- `/{slug}/live` — *"What is happening NOW across everything?"* — Activity feed timeline (agent started, tool use, edit, commit, milestone passed, DoD failed, tripwire fired). Filters: All / Live now / Decisions / Failures. Chips per row link to the Initiative or Issue.
- `/{slug}/initiatives` — *"What initiatives are in flight?"* — Grid of tiles, one per Initiative, each showing the PRD title, status pill, milestone progress, running-agents row, PR open badge, last activity, and a 3-row mini-feed.
- `/{slug}/initiatives/[id]` — *"How is THIS initiative progressing?"* — Per-initiative Kanban board scoped to that initiative's issues, with the existing live-execution layer (heartbeat, phase, agent dot) on each card. Reuses `feature-detail` as-is — it already renders PRD, milestones, DoD, board, and decisions.
- `/{slug}/initiatives/[id]/issues/[issueId]` — Issue detail unchanged.

**Decisions — top-level cross-Initiative**: `/{slug}/decisions` aggregates the `decision_log` table across every Initiative in the workspace. Each entry shows title / decision / learning + chips for ADR refs (links to `docs/adr/<slug>.md`) and context terms (links to `CONTEXT.md` anchors). The format honours `~/.claude/skills/grill-with-docs/ADR-FORMAT.md` so decisions promoted to ADRs flow cleanly between the two.

**Chrome — slim, manifest-driven**: workspace switcher removed (one workspace per Claude session, resolved from the manifest). Project header shows name + branch + mode. Primary nav: Live, Initiatives (count), Decisions, Inbox (unread badge). Workbench section groups Agents, Costs, Skills, Runtimes, Autopilots (the last demoted from primary). Pinned Initiatives section lists pinned features grouped by current status with live indicators. Footer shows ambient "N agents active".

**Workspace creation flow — deleted**: the `/workspaces/new` route, the `CreateWorkspaceForm`, the `NewWorkspacePage`, the `useCreateWorkspace` mutation, and the `paths.newWorkspace` helper all go. The root page, when no `last_workspace_slug` cookie is present, renders a `NoWorkspacePage` empty state telling the developer to run `/setup-multica` in Claude Code.

## User Stories

1. As a developer, I want the app to open straight to live execution, so that I don't have to fill in a workspace form before I can see anything happen.
2. As a developer, when no `.multica/workspace.toml` has been discovered yet, I want a clear empty state with the `/setup-multica` command, so that I know exactly the next step instead of being prompted to create something the system should resolve for me.
3. As a developer with a manifest already written, I want the app to land me on the Live view of my project, so that the manifest's intent is honoured without ceremony.
4. As a developer running 3 initiatives in parallel, I want a single Live feed that shows me what every agent is doing right now, so that I can scan across initiatives without diving in.
5. As a developer scanning the feed, I want filters for Live-now / Decisions / Failures, so that I can quickly narrow to a question (what just broke, what's running, what was decided).
6. As a developer, I want each feed event to link to its Initiative and (when applicable) its Issue, so that I can pivot from "something happened" to the full context in one click.
7. As a developer, I want an Initiatives grid view showing every in-flight initiative as a tile, so that I see the overall portfolio without scrolling a feed.
8. As a developer, I want each tile to show milestone progress, running agents, last activity, and whether a PR is open, so that I can triage which initiatives need attention.
9. As a developer, when I click a tile, I want to drop into the Kanban board for THAT initiative, so that I see only its issues — not a mix from every initiative.
10. As a developer on the per-initiative board, I want the live-execution layer (phase, heartbeat, agent dot) on each issue card, so that I can see at a glance which issue an agent is mid-edit on.
11. As a developer, I want a top-level Decisions page aggregating decisions across every initiative, so that I can reread institutional memory without remembering which initiative produced each lesson.
12. As a developer, I want each decision to link to its referenced ADR file in `docs/adr/`, so that decisions that turned into architecture rules are one click away from the source of truth.
13. As a developer, I want context terms in decisions to link to anchors in `CONTEXT.md`, so that the glossary stays a living document that the decision log reinforces.
14. As a developer, I want the sidebar to no longer offer me a workspace switcher, so that the chrome doesn't suggest a workspace selection model that contradicts the manifest-driven reality.
15. As a developer, I want the project header in the sidebar to show the active workspace name + branch + execution mode, so that the manifest's resolved identity is ambient and visible.
16. As a developer, I want Agents, Costs, Skills, Runtimes, and Autopilots grouped under a Workbench section in the sidebar, so that day-to-day execution surfaces (Live / Initiatives / Decisions / Inbox) lead and the inventory surfaces sit below.
17. As a developer, I want Pinned Initiatives listed in the sidebar grouped by current status (Running / In review / Other / Done), so that I can switch between my current focus initiatives without leaving the sidebar.
18. As a developer, I want the footer of the sidebar to show "N agents active" as an ambient indicator, so that I always know whether the daemon is busy or idle.
19. As a developer, I want the Inbox to surface only system events — initiative_tripwire, feature_ready_for_review, dod_failed — so that the inbox is a real attention queue, not a noisy social feed.
20. As a developer, I want the `/features` URL renamed to `/initiatives` everywhere, so that the URL vocabulary matches the domain language the rest of the system uses.
21. As a developer, when I land on a stale `/{slug}/features/...` link from outside the app, I want a 404, so that the URL rename is honest and search engines / docs converge on the new shape.
22. As a developer, I do not want a "New Issue" or "Create Workspace" button anywhere in the UI, so that the chrome reflects "Claude creates things via MCP, you read".
23. As a developer, I want all the existing reuse paths preserved — pins, search, breadcrumbs — to keep working through the rename, so that the refactor doesn't churn unrelated workflows.
24. As a developer with multiple feature pins, I want my pins to survive the URL rename, so that I don't have to repin everything after the refactor lands.
25. As an agent (Claude via MCP), I want the web app to not expose creation surfaces, so that the user is never tempted to create things by hand that should flow through the manifest + MCP plane.
26. As a maintainer, I want the workspace-creation form, `useCreateWorkspace` mutation, and `paths.newWorkspace` helper deleted, so that the codebase no longer carries dead UI for a flow that doesn't exist anymore.
27. As a maintainer, I want the `live`, `initiatives`, and `decisions` segments added to `reserved_slugs.json` and regenerated for the TS side, so that a future workspace cannot collide with the new route shape.
28. As a developer, I want the existing `feature-detail.tsx`, `feature-board.tsx`, `issue-detail.tsx`, `decision-log-section.tsx`, and `derive-liveness.ts` reused, so that the refactor consolidates onto the strongest pieces of the existing code rather than rebuilding them.
29. As a developer, I want the prototype at `/prototype/initiative-runner/*` left untouched in this PRD, so that the production refactor doesn't get entangled with throwaway code (a follow-up sweep deletes the prototype).
30. As a maintainer, I want every load-bearing module (live aggregator, decision log workspace query, sidebar shape, path helpers) covered by tests that fail closed, so that a future change can't silently break "the app opens on Live" or "decisions render across initiatives".

## Implementation Decisions

### Information architecture

- The sidebar adopts the shape prototyped in `apps/web/app/prototype/initiative-runner-chrome/_chrome-b-wide.tsx`. Three vertical zones: **Primary** (Live, Initiatives, Decisions, Inbox), **Workbench** (Agents, Costs, Skills, Runtimes, Autopilots), **Pinned Initiatives** (pinned features grouped by current status with live indicators). Header is a slim project bar (no workspace switcher). Footer shows "N agents active" + Settings.
- Three-level hybrid landing replaces the flat board: Live (feed) → Initiatives (tiles) → Initiative detail (board). The board is no longer a top-level surface — issues are only viewed in the context of their Initiative.
- Issue detail page survives intact at `/{slug}/initiatives/[id]/issues/[issueId]`.
- Decisions surface as top-level at `/{slug}/decisions`, integrated with ADR files in `docs/adr/` and context terms in `CONTEXT.md`. Format alignment follows `~/.claude/skills/grill-with-docs/ADR-FORMAT.md`.

### URL renames

- `/features` → `/initiatives` (full rename, no alias). Backend schema and TS type names stay `Feature` — URL-only rename. Path helpers (`paths.features` → `paths.initiatives`, etc.) renamed in one commit; every internal link converges through `paths.ts`.
- `/issues` route deleted as a top-level — issues live under `/initiatives/[id]/issues/[issueId]`.
- `/workspaces/new` deleted entirely.
- `/usage` stays at the same route; only the sidebar label changes to "Costs" (deferring a route rename to a later cleanup).

### Modules to build

- **`useLiveEvents`** — pure aggregator hook. Inputs: cached `agentTaskSnapshotOptions(wsId)` rows, cached `inboxKeys.list(wsId)` items, and `now`. Output: `ActivityEvent[]` sorted newest-first with stable IDs. Internally calls `deriveLiveness` (already shipping) to attach phase + heartbeat to running rows. Deep module: simple inputs in, structured events out; the merge/dedup/sort logic is the testable substance. No new backend endpoint — client-side aggregation off existing TQ caches.

  Event shape (from the prototype data layer):

  ```ts
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
  ```

- **`LiveFeedPage` + `LiveEventRow`** — UI components over `useLiveEvents`. Filter chips (All / Live now / Decisions / Failures) are local `useState`; no URL state in v1.

- **`InitiativesTilesPage`** — grid component over `featureListOptions(wsId)` + `agentTaskSnapshotOptions(wsId)`. Each tile shows: title, status pill, milestone progress, running-agents dots, PR badge, last activity, top-3 mini-feed (reuses `LiveEventRow`).

- **`DecisionsPage`** — list component over a new `workspaceDecisionsOptions(wsId)` query. Renders the extracted `DecisionRow` component with a feature chip linking to that initiative.

- **`DecisionRow`** — extracted from `packages/views/features/components/decision-log-section.tsx` into a sibling so both the per-initiative section and the cross-initiative page consume one component.

- **`NoWorkspacePage`** — static empty state for the root when no workspace is discovered. Renders the literal `/setup-multica` command in a code block and a link to standalone-install docs.

### Backend additions

- New sqlc query `ListDecisionLogByWorkspace(workspace_id, limit, offset)` in `decision_log.sql`. Mirrors the existing per-feature query.
- New HTTP handler `ListDecisionLogWorkspace` at `GET /api/decisions`, workspace-scoped via auth context (same pattern as `ListInbox`). Reuses the existing `decisionLogToResponse`.

### Sidebar refactor specifics

- Workspace switcher (`DropdownMenu` block in `app-sidebar.tsx`) deleted. Project header reads name + branch + mode from the current `Workspace` (extends `AmbientProjectBar` accordingly).
- `NavKey` types change: primary = `'live' | 'initiatives' | 'decisions' | 'inbox'`; workbench = `'agents' | 'costs' | 'skills' | 'runtimes' | 'autopilots'`.
- Pinned section filters `pinnedItems` to `item_type === 'feature'` and groups by current status via cached `featureDetailOptions`. Issue pins (if any remain) are dropped from the sidebar — issues are visited from their initiative.
- i18n labels updated in `packages/views/locales/en/layout.json` (issues→Live, features→Initiatives, plus new keys for `decisions`, `costs`, `workbench`, `pinned_initiatives`).

### Workspace creation removal

- Delete `apps/web/app/workspaces/new/page.tsx`, `packages/views/workspace/new-workspace-page.tsx`, `packages/views/workspace/create-workspace-form.tsx`, the `useCreateWorkspace` block in `packages/core/workspace/mutations.ts`, `paths.newWorkspace()`, and the related `GLOBAL_PREFIXES` entry.
- Root page (`apps/web/app/page.tsx`): when no `last_workspace_slug` cookie, render `<NoWorkspacePage />` instead of redirecting to `/workspaces/new`. Update `resolvePostAuthDestination([])` and its tests.
- Regenerate `reserved_slugs.json` (and the TS mirror via `pnpm generate:reserved-slugs`) so `workspaces` is no longer reserved if it was only kept for `/workspaces/new`. Add `live`, `initiatives`, `decisions` to the reserved list.

### What survives the refactor (reuse)

- `feature-detail.tsx`, `feature-board.tsx`, `feature-milestones-section.tsx`, `decision-log-section.tsx` (with `DecisionRow` extracted), `issue-detail.tsx`, `BoardCardLiveLayer`, `useIssueLiveState`, `derive-liveness.ts`, `agentTaskSnapshotOptions`, the inbox queries, the pins infrastructure, `AmbientProjectBar`, `WorkspaceSlugProvider`, the auth flow (`LoopbackAuth` + singleton user), the route guard structure.

### Slicing for execution

Five slices, sequenced to land foundation before consumers:

1. **Paths + sidebar** (chrome B port) — adds new path helpers, rewrites `AppSidebar`, extends `AmbientProjectBar`, updates labels. Placeholder pages for `/live`, `/initiatives`, `/decisions` so nav resolves.
2. **Delete workspace-creation flow** — files, mutation, paths, root-page empty state. Parallelisable with slice 1.
3. **Live feed** — `useLiveEvents` + `LiveFeedPage` + `LiveEventRow`.
4. **Initiatives tiles + per-initiative board rename** — `InitiativesTilesPage`; URL rename `/features` → `/initiatives`; reuse `feature-detail.tsx` at the new path.
5. **Decisions cross-initiative page** — backend query + handler, frontend page, extracted `DecisionRow`.

## Testing Decisions

Good tests here assert **external behavior**: given inputs (TQ cache snapshots, query params, URL state, fixture data) does the module produce the right observable outcome (correct event order, correct path string, correct DOM marker, correct DB-backed list)? Avoid asserting private call sequences or struct internals.

- **`useLiveEvents`** — Vitest with fixtures (mock task snapshots + inbox items + `now`). Cases: empty inputs → empty output; tool_use after agent_started → ordered correctly; deduped events when the same id appears in two sources; phase/heartbeat threaded through from `deriveLiveness`. The aggregator is the single deep module new to this PRD — it gets the most coverage. Prior art: `derive-liveness.test.ts` and `agent-task-snapshot.test.ts` patterns.

- **`ListDecisionLogByWorkspace` handler** — Go test, DB-gated, mirroring `server/internal/handler/decision_log_test.go`. Cases: empty → empty list; one decision per feature across two features → both returned newest-first; pagination via `limit/offset`; workspace isolation (decisions of another workspace not leaked). The handler test exercises the SQL too.

- **Path helpers** — `paths.test.ts` updated for the new shape (`live`, `initiatives`, `initiativeDetail`, `initiativeIssue`, `decisions`; `newWorkspace` removed; `features` removed). `resolve.test.ts` updated for the changed `resolvePostAuthDestination([])` return.

- **Sidebar** — `app-sidebar.test.tsx` updated to assert the new nav structure: primary group has 4 items in order, Workbench group has 5 items including Autopilots, workspace switcher dropdown is absent, "N agents active" footer is rendered when the snapshot has running tasks.

- **`DecisionsPage`** — Testing Library component test with a mocked TQ provider seeded with two decisions. Asserts: both rows render; ADR chip is a link to `docs/adr/<ref>.md`; context-term chip is a link to `CONTEXT.md#<anchor>`; feature chip links to `paths.initiativeDetail`.

- **`LiveFeedPage`** — Testing Library component test with a mocked TQ provider seeded with fixture task snapshots + inbox items. Asserts: rows render in time order; filter chip "Failures" hides non-failure rows; clicking an initiative chip navigates to `paths.initiativeDetail`.

Skipped on purpose:
- Visual / snapshot tests for the new pages — too brittle for layout iteration.
- E2E tests for the rename — covered by typecheck + the targeted unit tests + manual verification listed below.

## Out of Scope

- Renaming the `Feature` TS type or Go model to `Initiative`. URL-only rename in this PRD.
- A backend `/api/activity` event stream. Client-side aggregation over existing caches is sufficient until performance shows otherwise.
- Restructuring the Costs page (`/usage`) beyond a sidebar relabel.
- Removing the "assign agent to issue" affordance from `assignee-picker.tsx` / `create-agent-dialog.tsx` — power-user surface, follow-up PRD.
- Touching the prototype directory at `apps/web/app/prototype/initiative-runner*` — reference only; a follow-up sweep deletes it.
- Backwards-compatibility aliases for any of the renamed/removed routes.
- Standalone install (`multica up`, embedded Postgres, supervisor, plugin scaffold) — covered by `.scratch/standalone-install/PRD.md`, independent track.
- Multi-workspace support via the sidebar switcher. The manifest model resolves one workspace per Claude session; opening Claude in a different folder switches workspaces automatically. If multi-project switching proves necessary later, it's a follow-up.

## Further Notes

- The 3-level hybrid landing was chosen during a `/prototype` session that produced 6 variants (3 landing × 3 chrome). The user picked: chrome B (wide nav + initiative list) and a hybrid combining the strengths of the three landing variants — Feed (B) as Live, Tiles (C) as Initiatives, Board (A) as per-initiative detail. The prototypes live at `apps/web/app/prototype/initiative-runner*` for reference until the production refactor lands.
- Decisions as top-level reflects its growing centrality. The retrospective Run already writes to `decision_log` at Milestone closeout; that surface is currently nested per-feature, so the institutional memory is fragmented. Promoting it to top-level closes the loop with `/grill-with-docs`: decisions accumulate, the most-cited ones get promoted to ADRs in `docs/adr/`, and the glossary in `CONTEXT.md` absorbs evolving vocabulary referenced by the decisions.
- Workspace switcher removal is a one-way door we believe is correct. Manifest-driven resolution means one Claude Code session targets one workspace; switching means opening Claude in a different folder. If we discover the multi-project switcher is genuinely needed (e.g. a user juggling `meu-produto` + `side-projects` in one Claude session), it's a 1-day add. We ship without it first to keep the chrome honest.
- The autonomy-hardening work (orchestrator dispatch, validator fan-out, DoD evaluation, tripwire enforcement, completion-driven teardown) is already wired in the backend and observed in production traces. This PRD makes that work *visible* — the web view becomes the read surface for an execution layer that already runs.
