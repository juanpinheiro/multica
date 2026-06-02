# Issue 16: Mission Monitor — Initiative view

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/initiative-runner/PRD.md` — see ADR-0005, and the web-execution-monitor work.

## What to build

The Initiative view: PRD, ordered Milestones, the DoD, and current status in one place, plus a control to
flip Initiative status (a UI mirror of the MCP). Extends the existing live board (Variant A, honest
progress) so cards move as agents claim/work/finish. The board's columns carry Issue status; the live
layer shows which cards have an active Run.

## Acceptance criteria

- [x] An Initiative page shows its PRD, Milestones, DoD, and status
- [x] A status-flip control changes Initiative status (mirrors the MCP); reflects in the execution plane
- [x] The board renders Issue status as columns with the live-Run layer overlaid
- [x] `pnpm typecheck` and `pnpm test` pass

## Blocked by

- `06-reshape-entities-and-status-claim`

## Comments

### Key decisions

1. **`FeatureMilestonesSection`** (new `packages/views/features/components/feature-milestones-section.tsx`) — shows ordered Milestones for a feature with their validation status (pending/passed/failed icon + label) and the DoD assertions under each Milestone (pass ✓ / fail ✗ / pending ○ markers). Fetches via `featureMilestonesOptions` (new feature-scoped milestone query) and `milestoneDodOptions` per milestone.

2. **`FeatureBoardView`** (new `packages/views/features/components/feature-board.tsx`) — a compact, read-only board showing all `BOARD_STATUSES` as columns. Each card links to the issue detail and renders `BoardCardLiveLayer` (exported via `useIssueLiveState`) so the liveness layer (phase stepper, shimmer, heartbeat, activity counters) appears on any card with an active Run. Fetches via `featureIssueListOptions` — a new query that filters issues by `feature_id` and groups them by status.

3. **Status-flip control** — the existing sidebar status dropdown (`DropdownMenu`) in `feature-detail.tsx` already mirrors the MCP's `set_feature_status` tool and routes through the server-side state-machine guard (issue 14). No new UI needed; the "Approve & start" button (draft → ready) was already present. This AC is satisfied by the existing sidebar.

4. **Replaced `FeatureIssuePipelineView`** with `FeatureBoardView` in `feature-detail.tsx`. The pipeline view (ready-now / blocked grouping) was removed since the board provides a more informative full-lifecycle view. The existing `featureIssuesOptions` query is retained for PR badges in the breadcrumb.

5. **`useIssueLiveState` exported** from `board-card.tsx` so `feature-board.tsx` can use the same liveness hook without duplicating the snapshot/message/derivation logic.

6. **New query functions:**
   - `featureMilestonesOptions(wsId, featureId)` in `packages/core/milestones/queries.ts` — filters milestones by `feature_id` and sorts by `position`
   - `featureIssueListOptions(wsId, featureId)` in `packages/core/issues/queries.ts` — re-uses `fetchFirstPages` with `feature_id` filter

### Files changed

- **New:** `packages/views/features/components/feature-milestones-section.tsx`
- **New:** `packages/views/features/components/feature-milestones-section.test.tsx` (9 tests)
- **New:** `packages/views/features/components/feature-board.tsx`
- **Modified:** `packages/views/features/components/feature-detail.tsx` — replaced pipeline view with milestones + board sections; removed `FeatureIssuePipelineView` and all its helpers
- **Modified:** `packages/views/features/components/feature-detail.test.tsx` — removed pipeline tests, added milestone/board smoke tests, mocked new components
- **Modified:** `packages/core/milestones/queries.ts` — added `featureMilestonesOptions` + `feature` key
- **Modified:** `packages/core/issues/queries.ts` — added `featureIssueListOptions`
- **Modified:** `packages/views/issues/components/board-card.tsx` — exported `useIssueLiveState`
- **Modified:** `packages/views/locales/en/features.json` — added `detail.section_milestones` and `milestones.empty`

### Blockers / notes

- `pnpm typecheck` and `pnpm test` (685 → 695 total tests across packages) both pass clean.
- The board is read-only (no DnD). Issue 17 (Run/Handoff timeline) can reuse `FeatureMilestonesSection` data via `milestoneDodOptions` and will add the per-Run timeline to the issue detail.
- The board uses `featureIssueListOptions` which paginates by `BOARD_STATUSES` × `ISSUE_PAGE_SIZE`. For very large Initiatives the column would need pagination; not required for current scale.
