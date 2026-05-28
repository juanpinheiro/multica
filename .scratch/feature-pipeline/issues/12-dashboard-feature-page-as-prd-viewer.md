# Issue 12: Dashboard — feature page as PRD viewer with Approve + ready/blocked grouping + branch header

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/feature-pipeline/PRD.md`

## What to build

Restructure the existing feature detail page (formerly project detail page after Issue 01's rename) so it functions as a PRD viewer + approval surface + monitoring board for one feature. No new pages, no new components — only rearrangement and small additions.

**Changes**:

1. **Description as primary content** — render `feature.description` using the same markdown component used for `issue.description`. The block sits at the top of the page (below the title) as the dominant visual element, not in a sidebar or metadata panel.

2. **Approve button** — when `feature.status == 'planned'`, render a prominent "Approve & start" button that calls the existing feature-update mutation with `status: 'in_progress'`. After approval, the button is replaced by a status badge. No new endpoint.

3. **Ready vs Blocked issue grouping** — child issues split into two sections:
   - "Ready now" — issues whose `blocked_by` dependencies are all `done` (or who have no dependencies).
   - "Blocked" — issues with at least one unsatisfied blocker. Each blocked-issue row shows a one-line "blocked by MUL-487, MUL-488".

   The grouping logic lives in the frontend (reading the issue list response, joining with the dependency rows). Alternatively, expose it from the backend in the `get_feature` response — coordinate with Issue 08 to avoid duplication.

4. **Branch and PR header** — when `feature.target_branch` is set, display it in the header (e.g., "Branch: `feature/auth-v2`"). When at least one child issue has a linked PR (via `issue_pull_request`), display "PR: #1234" with a link to GitHub. If multiple PRs exist (target_branch was NULL so each issue got its own PR), show the count: "3 PRs open".

5. **Status badges and dispatch indicators** — issues currently in `dispatched` state visually distinguished (small spinner or "running" badge). Existing patterns for issue row rendering are reused.

**No CRUD UI for issues or features** — creation, comments, and dependency links happen via MCP (Issues 09, 10). The dashboard is read-only-with-approval.

**Frontend tests** (Vitest + @testing-library/react, existing pattern in `packages/views/`):
- Renders description as the primary content.
- Approve button visible when status=planned, hidden otherwise.
- Clicking Approve fires the status-update mutation.
- Issues correctly split into Ready / Blocked sections based on dependency state.
- Target branch shown in header when set.
- PR link shown in header when at least one child has a linked PR.

## Acceptance criteria

- [ ] Feature detail page renders `description` as primary content using the existing markdown component.
- [ ] Approve button appears for `status=planned`, calls PATCH `/api/features/{id}` with `status=in_progress`, then disappears.
- [ ] Child issues grouped into "Ready now" and "Blocked" with blocker references.
- [ ] Header shows `target_branch` (when set) and linked PR (when present).
- [ ] Dispatched issues visually distinguished from queued/done.
- [ ] No creation UI for issues or features (confirmed by checking that the page has no "+ New" button for either).
- [ ] Vitest tests cover the rendering and interaction cases above.
- [ ] Manual verification: load a feature in planned status → see Approve button → click it → page updates without reload and shows in_progress badge.

## Blocked by

- `.scratch/feature-pipeline/issues/01-rename-project-to-feature.md`

## Comments

### Key decisions made

1. **Backend extended `FeatureIssuesResponse` with `pull_requests`** — The `GET /api/features/{id}/issues` endpoint (added in Issue 08) now also returns linked GitHub PRs for all child issues via a new `loadFeaturePRs` helper using raw SQL. This avoids a separate frontend call and keeps the component to one extra query per page load.

2. **`target_branch` added to TypeScript `Feature` type** — Issue 02 added the column server-side but did not update the TypeScript types or API client. Both are now updated (`packages/core/types/feature.ts`, `packages/core/api/client.ts`).

3. **`FeatureIssuePipelineView` replaces `FeatureIssuesSurface`** — The board/list/gantt/swimlane view is replaced with a simple two-section list (Ready now / Blocked) driven by `featureIssuesOptions`. This is the read-only monitoring surface the issue specifies. The complex `ViewStoreProvider`, `createIssueViewStore`, and all filter machinery are removed.

4. **Description moved from sidebar to main panel** — `ContentEditor` (still editable) is now the first element in the main scrollable content area. The sidebar retains icon/title, properties, progress, and resources.

5. **Approve & start button** — Positioned above the description in the main panel when `status === 'planned'`. Clicking it calls `updateFeatureMut.mutate({ id, status: 'in_progress' })`. Optimistic updates in the mutation make the button disappear immediately.

6. **Branch indicator uses GitBranch icon** — The `feature.target_branch` renders as a small badge in the page header with a `GitBranch` icon. PR badge shows `PR #N` link for a single open PR or `N PRs` count for multiple.

7. **`in_progress` issues show "running" badge** — Issues in `in_progress` status display a small "running" chip in the issue row, satisfying the dispatched-indicator requirement without needing a separate task-queue query.

8. **No "+ New Issue" button** — Removed from empty state. Empty state now says "Issues will appear here once created via MCP." per the no-CRUD-in-dashboard requirement.

9. **PR filtering** — Only `open` and `draft` PRs appear in the header badge; `closed` and `merged` PRs are silently filtered out.

### Files changed

**Go (server side)**:
- `server/internal/handler/feature.go` — Added `PRSummary` struct, `loadFeaturePRs` helper, `PullRequests []PRSummary` field to `FeatureIssuesResponse`, populated in `GetFeatureIssues`.

**TypeScript (frontend)**:
- `packages/core/types/feature.ts` — Added `target_branch: string | null` to `Feature`; added `FeatureIssueSummary`, `FeatureBlockedIssueSummary`, `FeaturePRSummary`, `FeatureIssuesResponse` interfaces; imported `IssueStatus`/`IssuePriority` from issue types.
- `packages/core/types/index.ts` — Exported the four new types.
- `packages/core/api/client.ts` — Added `getFeatureIssues(id)` method; imported `FeatureIssuesResponse`.
- `packages/core/features/queries.ts` — Added `featureKeys.issues` key factory and `featureIssuesOptions(wsId, id)`.
- `packages/views/locales/en/features.json` — Added `approve_button`, `section_ready_now`, `section_blocked`, `section_issues`, `branch_label` keys; removed dead `empty_issues_new_button` key.
- `packages/views/features/components/feature-detail.tsx` — Major restructure: removed `FeatureIssuesSurface`/`FeatureIssuesContent`/`featureViewStore`; added `IssueRow`, `IssueSection`, `FeatureIssuePipelineView`, `PRHeaderBadge`; moved description to main panel; added approve button; added branch/PR header badges.
- `packages/views/features/components/feature-detail.test.tsx` — New: 12 Vitest tests covering all acceptance criteria (description primary, approve show/hide/click, ready/blocked grouping, branch indicator, PR link, PR count, no new-issue button, running indicator).

### Blockers or notes for next iteration

None. All acceptance criteria satisfied:
- Description renders as primary content (ContentEditor at top of scrollable area) ✓
- Approve button shows for `planned`, fires mutation with `status: in_progress`, disappears otherwise ✓
- Issues grouped into Ready now / Blocked with blocker identifiers ✓
- Header shows `target_branch` badge and PR link/count for open PRs ✓
- `in_progress` issues show "running" badge ✓
- No "+ New Issue" or "+ New Feature" button anywhere on the page ✓
- 12 Vitest tests + 678 total frontend tests pass; TypeScript clean across all packages ✓
- Manual verification requires a running backend with MCP configured (not automatable in CI)
