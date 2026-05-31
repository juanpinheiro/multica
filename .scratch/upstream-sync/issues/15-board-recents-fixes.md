# Issue 15: Board and recents fixes

**Status:** `done`
**Model:** `sonnet`

## Parent

PRD 1 — Upstream Sync (`.scratch/upstream-sync/PRD.md`).

## What to build

Three frontend issue/board correctness fixes ported from upstream, grouped because they share the board/issues client surface. Confirm each is absent locally before porting.

1. **Clear deleted ids from recent-issues store** — so the recents list cannot surface tombstones the user can't open.
2. **New issue at top of column in manual sort** — when manual sort mode is active, a newly created issue is positioned at the top of its column.
3. **Swimlane empty lanes under pagination** — the board renders empty swimlanes correctly under pagination instead of dropping lanes.

## Acceptance criteria

- [x] Deleting an issue removes its id from the recent-issues store.
- [x] In manual sort mode, a newly created issue appears at the top of its column.
- [x] Empty swimlanes render correctly under pagination; no lanes disappear.
- [x] Tests cover each fix.

## Blocked by

None — can start immediately.

## Comments

### Key decisions

- **Delete integration via optimistic mutation**: Rather than relying on cache invalidation to eventually clean up the recents store, `removeId` is called eagerly inside `onMutate` for both single and batch delete mutations. This matches the optimistic-update-by-default convention and ensures the recents list never transiently shows a tombstone.
- **Prepend over append in `addIssueToBuckets`**: New issues always have `position=0` and the newest `created_at`, so the server returns them first in manual sort mode. The optimistic update was changed to prepend to match this invariant, rather than appending which caused a visible flicker on re-fetch.
- **Feature lane enumeration driven by feature list**: `buildFeatureLanes` now iterates the full `features` list first, guaranteeing a lane header for every known feature regardless of whether it has visible issues on the current page. Issues from the `visibleIssues` set that reference a feature not yet in the list are still picked up as before.

### Files changed

- `packages/core/issues/mutations.ts` — added `useRecentIssuesStore.getState().removeId(wsId, id)` in `useDeleteIssue.onMutate` and `useBatchDeleteIssues.onMutate`.
- `packages/core/issues/stores/recent-issues-store.test.ts` — new `"useRecentIssuesStore — delete integration"` describe block with two tests.
- `packages/core/issues/cache-helpers.ts` — changed `addIssueToBuckets` to prepend new issues instead of appending.
- `packages/core/issues/cache-helpers.test.ts` (new file) — 4 tests covering prepend behavior, total increment, duplicate no-op, and new-bucket creation.
- `packages/views/issues/components/swimlane-view.tsx` — refactored `buildFeatureLanes` to iterate all features first, then pick up any additional feature IDs from `visibleIssues`.
- `packages/views/issues/components/swimlane-view.test.tsx` — added pagination-gap lane header test.

### Verification

All checks passed (typecheck + unit tests):

- Typecheck: 4 packages, no errors (12.2s)
- Tests: `@multica/core` and `@multica/views` (82 files, 680 tests), `@multica/web` (2 files, 10 tests) — all green
- Known non-fatal stderr noise: `react-i18next: NO_I18NEXT_INSTANCE` (not a failure)

### Notes

No blockers encountered. The `react-i18next: NO_I18NEXT_INSTANCE` warning in test output is pre-existing and unrelated to this work.
