# Issue 19: Views — shared BreadcrumbHeader + React hygiene

**Status:** `done`
**Model:** `sonnet`

## Parent

PRD 1 — Upstream Sync (`.scratch/upstream-sync/PRD.md`).

## What to build

Two views-layer refactors ported from upstream, grouped because both are cross-view cleanups that should land late so they rebase cleanly over the other view changes.

1. **Unify detail/list headers into a shared breadcrumb header** — remove duplication by consolidating the detail-page and list-page headers into one shared component with consistent behavior.
2. **React-hygiene cleanup sweep** — apply the upstream React cleanup (correct button types, modern context consumption, non-mutating array sorts, error-boundary fixes) selectively to files the fork still has; skip anything that only existed for deleted platforms.

## Acceptance criteria

- [x] Detail and list pages share one breadcrumb header; behavior is consistent across them.
- [x] The React-hygiene fixes are applied only to files present in the fork; no deleted-platform files are reintroduced.
- [x] Existing view tests pass; no behavior regressions.

## Blocked by

None — can start immediately (land late to ease rebasing over other view changes).

## Comments

### Key decisions

- **`BreadcrumbHeader` already existed** at `packages/views/layout/breadcrumb-header.tsx` but was unused. The refactor adopts it for both detail pages rather than introducing a new component.

- **`feature-detail.tsx`**: Replaced the manual `PageHeader` + inline `AppLink`/`ChevronRight` breadcrumb with `BreadcrumbHeader`. The workspace name becomes segment 0 (linked to features list); the feature title + branch badge + PR badge become the last segment with a composite ReactNode label (`className: "flex items-center gap-1.5 min-w-0"` on the segment so the flex items don't wrap oddly inside `BreadcrumbPage`). The `AppLink` import was removed (no longer used in this file).

- **`issue-detail.tsx`**: Replaced the same pattern with `BreadcrumbHeader`. The segments are built inline as a spread array: workspace (conditional, linked) → feature (conditional, linked, with `FeatureIcon` in the label) → parent issue (conditional, linked) → combined identifier+title (always last, no href). Identifier and title are placed in the same segment to avoid adding a separator arrow between them, with the identifier carrying `text-muted-foreground tabular-nums shrink-0` to preserve the existing muted color despite the parent's `text-foreground`. The `BreadcrumbSegment` type was imported to use `satisfies` for inline type checking.

- **React hygiene**: Two `<button>` elements in `packages/views/modals/create-feature.tsx` were missing `type="button"` (lines 245 and 262 — expand/collapse and close buttons inside a `TooltipTrigger` render prop within a dialog). Both now have `type="button"` to prevent accidental form submission. No context consumption patterns, mutating sorts, or error boundary issues were found in the fork's files that still needed fixing.

- **Test mocks updated**: Both `feature-detail.test.tsx` and `issue-detail.test.tsx` now mock `../../layout` (the index export) with a `BreadcrumbHeader` that renders each segment's label, using an `<a href>` when the segment has an href and a `<span>` otherwise. This preserves existing test assertions that check link hrefs (e.g., the feature breadcrumb link test in issue-detail.test.tsx).

### Files changed

- `packages/views/features/components/feature-detail.tsx` — swap `PageHeader` → `BreadcrumbHeader`; remove `AppLink` import (unused after refactor)
- `packages/views/features/components/feature-detail.test.tsx` — update layout mock from `PageHeader` to `BreadcrumbHeader`; add 2 breadcrumb segment tests
- `packages/views/issues/components/issue-detail.tsx` — swap `PageHeader` → `BreadcrumbHeader`; add `BreadcrumbSegment` type import
- `packages/views/issues/components/issue-detail.test.tsx` — add `../../layout` mock with link-aware `BreadcrumbHeader` stub
- `packages/views/modals/create-feature.tsx` — add `type="button"` to two `<button>` elements

### Verification

- `pnpm --filter @multica/views test` — 686 tests, 83 files, all pass
- `pnpm typecheck` — 4 packages checked, 0 errors
