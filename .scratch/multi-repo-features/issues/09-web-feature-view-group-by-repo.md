# Issue 09: Web feature view — group by repo + multi-PR header

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/multi-repo-features/PRD.md`

## What to build

Restructure the feature detail page so a cross-repo feature reads coherently. No new pages or components — a rearrangement of the feature-pipeline feature view consuming the new `get_feature` shape (Issue 07).

- Group child issues by **repo**, then by dependency layer (ready / blocked) within each repo, reusing the grouping the feature-pipeline view already introduced.
- The header shows the feature branch name and the **set of PRs — one per repo touched** — each with its status, instead of a single PR link.
- Apply the API Response Compatibility rules: parse the `get_feature` response through a zod schema with a fallback, optional-chain every field, and render a generic fallback for unknown enum values. A feature with a single repo renders as the degenerate one-group, one-PR case.

## Acceptance criteria

- [ ] The feature page groups issues by repo, then by ready/blocked within each repo.
- [ ] The header lists one PR per repo with status; a single-repo feature shows exactly one.
- [ ] The branch name is shown.
- [ ] The response is parsed via a schema with a fallback (no bare `as` cast); a malformed response (missing field, null array) renders without a white-screen, covered by a test feeding a malformed payload through the schema.
- [ ] A `packages/views/` test renders the page from a mocked `@multica/core` and asserts the grouped-by-repo layout and multi-PR header.
- [ ] `make check` passes.

## Blocked by

- Issue 01 (`issue.repo_id`, `github_pull_request.repo_id`).
- Issue 07 (`get_feature` grouped/multi-PR response).

## Comments

### Iteration 1 — implemented (Sonnet)

**Key decisions**

- **Types extended, not replaced.** `FeatureIssueSummary` and `FeaturePRSummary` gained `repo_id?: string | null` and `repo_name?: string | null` as optional fields so the degenerate single-repo case (no repo fields) continues to work without any migration or compatibility shim.
- **Grouping is front-end only.** The REST API (`GetFeatureIssues`) already included `repo_id`/`repo_name` on each issue and `repo_id` on each PR after Issue 07. No backend change was needed — the grouping happens in `groupIssuesByRepo` (pure function, no I/O) and `groupOpenPRsByRepo`.
- **Repo headers only when multiple groups.** `groupIssuesByRepo` returns sorted groups (named repos A–Z, unassigned last); `showRepoHeaders` is true only when there are 2+ groups. A single-repo feature renders identically to the old flat view — no header noise. Existing tests pass unchanged.
- **`PRHeaderBadge` now emits one badge per repo.** Each repo group's open PRs produce one badge: a link if 1 PR, a count if multiple. Two PRs in different repos → two `pr-link` elements. Two PRs in the same repo → one `pr-count`. Existing single-repo PR tests all still pass.
- **`FeatureIssuesResponseSchema` added to `packages/core/api/schemas.ts`.** The `getFeatureIssues` client method now uses `parseWithFallback` with defaults on arrays (`z.array(...).default([])`) and `.loose()` throughout. A malformed response (empty object, null, missing `blocked_by`) degrades to the fallback rather than throwing.
- **Clean-code pass:** `groupOpenPRsByRepo` extracted from `PRHeaderBadge` so each function has a single responsibility.

**Files changed**

- `packages/core/types/feature.ts` — `repo_id`/`repo_name` added to `FeatureIssueSummary`; `repo_id` added to `FeaturePRSummary`.
- `packages/core/api/schemas.ts` — `FeatureIssuesResponseSchema`, `EMPTY_FEATURE_ISSUES_RESPONSE`, and supporting sub-schemas added; `FeatureIssuesResponse` imported.
- `packages/core/api/client.ts` — `getFeatureIssues` uses `parseWithFallback` with the new schema.
- `packages/views/features/components/feature-detail.tsx` — `RepoGroup` type, `groupIssuesByRepo`, `groupOpenPRsByRepo`, updated `FeatureIssuePipelineView` (group-by-repo render), updated `PRHeaderBadge` (per-repo badges).
- `packages/views/features/components/feature-detail.test.tsx` — 3 new tests: multi-repo group headers, single-repo no-header, multi-repo PR badges.
- `packages/core/api/schemas.test.ts` — 4 new tests for `FeatureIssuesResponseSchema`: missing arrays default to `[]`, null input falls back, well-formed response parses repo fields, missing `blocked_by` defaults to `[]`.

**Verification**

- `pnpm typecheck`: 4/4 tasks pass.
- `pnpm test` (Vitest): 672 tests pass (82 test files, all packages).
- All 15 `feature-detail` tests pass (12 existing + 3 new).
- All 27 `schemas.test.ts` tests pass (23 existing + 4 new).
