# Issue 04: Initiatives tiles + URL rename /features → /initiatives

**Status:** `done`
**Model:** `opus`

## Parent

`.scratch/ui-refactor-initiative-runner/PRD.md`

## What to build

Lands the Initiatives tile grid at `/{slug}/initiatives`, renames `/features` to `/initiatives` everywhere, and wires the per-initiative board + issue detail under the new URL shape by reusing the existing `feature-detail.tsx` and `issue-detail.tsx`.

End-to-end behavior after this slice:

- `/{slug}/initiatives` renders a tile grid. One tile per Initiative, sorted with running first, then in_review, ready, blocked, done. Each tile shows: branch slug header, status pill, title, milestone progress (`{passed}/{total} milestones • {issuesDone}/{issuesTotal} issues` + a progress bar), running-agents row with coloured dots, a 3-row mini-feed of recent activity for that Initiative (reuses `LiveEventRow` from slice 03 if available; otherwise an inline minimal renderer that the slice introduces), a footer with last activity + PR-open badge + a blocked / done indicator. The tile is a link to `paths.initiativeDetail(id)`.
- `/{slug}/initiatives/[id]` renders the existing `FeatureDetail` component (PRD section, milestones with DoD, board with live layer, decisions section). No rewrite — just route to it.
- `/{slug}/initiatives/[id]/issues/[issueId]` renders the existing `issue-detail.tsx`.
- `/{slug}/features` and `/{slug}/features/[id]` return 404. The route folder is gone.
- `paths.features` / `paths.featureDetail` are deleted. All callers go through `paths.initiatives` / `paths.initiativeDetail` (including `PinRow`, search results, inbox links, breadcrumbs inside `feature-detail.tsx`).
- Pinned feature items survive the rename — TQ keys use `id`, not URL, and `featureDetailOptions(wsId, id)` is untouched.

The tile grid composition pulls from existing queries: `featureListOptions(wsId)` for the list, `agentTaskSnapshotOptions(wsId)` filtered by `feature_id` for the running-agents dots, `feature-milestones-section` data for milestone progress, `feature-resources-section` data for the PR badge.

`/usage` route stays — only its sidebar label changes ("Costs") in slice 01 i18n; this slice does nothing to it.

## Acceptance criteria

- [ ] `/{slug}/initiatives` renders the tile grid with one tile per Feature in the workspace.
- [ ] Each tile shows: title, status pill, milestone progress bar with `{passed}/{total}` labels, running-agents row when at least one agent is on it, mini-feed of 3 most-recent events, last activity, PR-open badge when `pr_url` is set, blocked-indicator when status is `blocked`.
- [ ] Clicking a tile navigates to `/{slug}/initiatives/<id>`.
- [ ] `/{slug}/initiatives/<id>` renders the existing `FeatureDetail` exactly as the old `/{slug}/features/<id>` did — PRD, milestones, DoD, board with live layer, decision log section, lead/resources/labels.
- [ ] `/{slug}/initiatives/<id>/issues/<issueId>` renders the existing `issue-detail.tsx`.
- [ ] `/{slug}/features` and `/{slug}/features/<id>` return 404 — the route folder is deleted.
- [ ] `paths.features` and `paths.featureDetail` are removed. `rg "paths\\.workspace\\(.*\\)\\.feature" packages apps` returns nothing.
- [ ] Pinned features still appear in the sidebar after the rename — pin TQ keys are id-based and survive.
- [ ] Breadcrumbs inside `feature-detail.tsx` link to the new `paths.initiatives` / `paths.initiativeDetail`.
- [ ] Search results and inbox links pointing at features resolve under the new URL shape.
- [ ] `make check` passes.

## Blocked by

- `.scratch/ui-refactor-initiative-runner/issues/01-paths-and-chrome.md` (needs the `/initiatives` route slot and the new path helpers)

## Comments

### Key decisions

- **Tile composition reuses the four caches already in the workspace**:
  `featureListOptions`, `milestoneListOptions(wsId)` (the workspace-wide variant —
  one fetch, then bucket by `feature_id` client-side), `agentTaskSnapshotOptions`,
  and the issue list. The mini-feed pulls from `useLiveEvents` (built in slice 03)
  so the running-agents/recent-activity story stays consistent with `/live`. No
  per-tile queries.
- **PR-open badge skipped for this slice.** The `Feature` model exposes no
  `pr_url`; the per-feature PR data lives behind `featureIssuesOptions(wsId, id)`,
  and fanning that out across every tile would be N extra queries. The blocked
  indicator + status pill carry the "needs you" signal until the backend lifts a
  PR summary onto the list response.
- **Milestone progress falls back to issue progress** when no milestones exist
  for an initiative. New initiatives without milestones still show a usable
  progress bar instead of a permanently-zero one.
- **`/issues/[id]` kept reachable, `/issues` (top-level listing) deleted.**
  Many existing callers — search results, mentions, gantt, list rows, comments —
  hold only an `issueId` and have no clean way to look up a `feature_id`. Keeping
  the legacy detail URL means those callers don't need a synchronous lookup
  before navigating; new code uses `paths.initiativeIssue(featureId, issueId)`,
  which `FeatureBoardView` now does because it already has the `featureId` on
  hand. The board listing page itself (`IssuesPage`) was deleted along with the
  cross-initiative flat board the PRD called out as "incoherent the moment 3–5
  initiatives run in parallel".
- **`resolvePostAuthDestination(first)` now lands on `/live`** instead of
  `/issues`. Slice 02 left this for slice 04; the PRD wants the app to open
  on the live execution view when there's a workspace cookie path.
- **`/features` segment removed from every list it lived in**: reserved-slugs
  JSON + the regenerated TS mirror, `WORKSPACE_ROUTE_SEGMENTS` in
  `link-handler.ts`, and the `LEGACY_ROUTE_SEGMENTS` proxy redirector. The proxy
  now redirects bare `/initiatives/...`, `/live/...`, `/decisions/...` to
  `/{slug}/{segment}/...` so legacy markdown links keep resolving.
- **Search NavKey vocabulary follows the chrome.** `search-command.tsx`'s
  `pages.features` palette entry became `pages.initiatives`; the keyword list
  retains `features` so users typing the old word still find it.
- **`useFeatureViewStore` deleted** along with `FeaturesPage`. The list-vs-grid
  toggle had only one consumer; no surface still needs it.

### Files changed

- `packages/views/initiatives/components/initiatives-tiles-page.tsx` — new tile
  grid (status sort, milestone+issue progress, running-agents row, 3-row mini
  feed, blocked indicator).
- `packages/views/initiatives/components/initiatives-tiles-page.test.tsx` — 8
  Testing-Library cases covering empty state, tile-per-feature, status sort
  order, click navigation, milestone progress, agent row, mini-feed slicing,
  blocked indicator.
- `packages/views/initiatives/index.ts` — public export.
- `packages/views/package.json` — adds the `./initiatives` export.
- `packages/views/locales/en/layout.json` — adds the `initiatives_page` block
  (headline, counters, progress strings, status labels, empty state copy).
- `packages/views/locales/en/search.json` — relabels the search palette entry
  to "Initiatives" and updates description / type-to-search copy.
- `apps/web/app/[workspaceSlug]/(dashboard)/initiatives/page.tsx` — placeholder
  swapped for `<InitiativesTilesPage />`.
- `apps/web/app/[workspaceSlug]/(dashboard)/initiatives/[id]/page.tsx` — new
  route rendering the existing `FeatureDetail`.
- `apps/web/app/[workspaceSlug]/(dashboard)/initiatives/[id]/issues/[issueId]/page.tsx`
  — new route rendering the existing `IssueDetail` (wrapped in `ErrorBoundary`
  with the issueId as reset key, mirroring `/issues/[id]`).
- `apps/web/app/[workspaceSlug]/(dashboard)/features/` (folder) — deleted.
- `apps/web/app/[workspaceSlug]/(dashboard)/issues/page.tsx` — deleted (the
  flat-board listing surface goes; `/issues/[id]/page.tsx` stays).
- `packages/views/features/components/features-page.tsx` + its `index.ts` entry
  — deleted.
- `packages/views/issues/components/issues-page.tsx` + `issues-page.test.tsx` +
  its `index.ts` entry — deleted.
- `packages/core/features/stores/view-store.ts` + `view-store.test.ts` and the
  `useFeatureViewStore` re-export in `packages/core/features/index.ts` —
  deleted (consumed only by the now-removed `FeaturesPage`).
- `packages/core/paths/paths.ts` — drops the `features` / `featureDetail`
  aliases.
- `packages/core/paths/paths.test.ts` — drops the corresponding assertions.
- `packages/core/paths/consistency.test.ts` — drops `features` from the
  parameterless-route set and the segment table.
- `packages/core/paths/resolve.ts` + `resolve.test.ts` — first-workspace lands
  on `paths.workspace(slug).live()`.
- `packages/views/features/components/feature-detail.tsx` — breadcrumb and
  post-delete redirect go through `wsPaths.initiatives()`.
- `packages/views/features/components/feature-detail.test.tsx` — the mocked
  `useWorkspacePaths` exposes `initiatives` / `initiativeDetail`.
- `packages/views/features/components/feature-board.tsx` — `FeatureIssueCard`
  links via `paths.initiativeIssue(featureId, issue.id)`; `FeatureBoardColumn`
  threads `featureId` through.
- `packages/views/issues/components/issue-detail.tsx` — breadcrumb feature
  segment links to `paths.initiativeDetail`.
- `packages/views/issues/components/issue-detail.test.tsx` — asserts the new
  `/test/initiatives/p-1` href.
- `packages/views/inbox/components/inbox-page.tsx`,
  `packages/views/autopilots/components/autopilot-detail-page.tsx`,
  `packages/views/search/search-command.tsx`,
  `packages/views/layout/app-sidebar.tsx` — switched feature path helpers to
  the initiative ones.
- `packages/views/layout/app-sidebar.test.tsx` — mocked `useWorkspacePaths`
  exposes `initiativeDetail` instead of `featureDetail`.
- `packages/views/editor/utils/link-handler.ts` — drops `features` from
  `WORKSPACE_ROUTE_SEGMENTS`.
- `apps/web/proxy.ts` — drops `features` from the legacy-segment redirector
  and adds `initiatives` / `live` / `decisions` so bare URLs land under the
  current slug.
- `server/internal/handler/reserved_slugs.json` — drops `features`.
- `packages/core/paths/reserved-slugs.ts` — regenerated.

### Blockers / notes for next iteration

- **PR-open badge on the tile.** A follow-up could lift a tiny PR summary
  (open count + first open URL) onto the `feature_list` response so the tile
  can light a `PR open` chip without paying N queries. Today the badge is
  omitted by design — see "PR-open badge skipped" above.
- **`paths.issueDetail(id)` / `/{slug}/issues/[id]` survive intentionally.**
  The PRD body said "issues live under `/initiatives/[id]/issues/[issueId]`",
  and that URL exists now, but the legacy permalink keeps working so that
  mentions, search hits, gantt rows, list rows, etc. don't all need to learn
  a feature-id lookup in this slice. A later sweep can either delete
  `/{slug}/issues/[id]` (and migrate every caller) or replace it with an
  in-route redirect that resolves the feature and forwards.
- **Unrelated Go failure:** `cmd/server` `TestWebSocketIntegration` fails
  with an i/o-timeout on `auth_ack` both with and without these changes
  (verified by re-running against a clean tree via `git stash`). Not
  introduced by this slice; flagging it for a separate investigation.
- **`make check` was not run end-to-end** (the local Postgres container is
  not up). I ran `pnpm typecheck` (clean — 4 packages), `pnpm test`
  (1133 passing — 414 core + 709 views + 10 web), and `go vet ./...` /
  `go test ./...` from the `server/` tree; the only failure is the
  pre-existing WebSocket integration test noted above.
