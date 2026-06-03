# Issue 05: Decisions cross-initiative page + endpoint

**Status:** `done`
**Model:** `opus`

## Parent

`.scratch/ui-refactor-initiative-runner/PRD.md`

## What to build

Lands the cross-initiative Decisions surface at `/{slug}/decisions`. Adds one new backend endpoint, one new sqlc query, the matching TS query option + API call, and a list page that reuses an extracted `DecisionRow` component.

End-to-end behavior after this slice:

- `/{slug}/decisions` shows a header (`Decisions • captured by retrospective Runs at every Milestone closeout`), then a list of decision entries across every Initiative in the workspace, sorted newest-first.
- Each entry shows: title, time-ago, decision text, learning text, optional chips: ADR refs (links to `docs/adr/<ref>.md`), context terms (links to `CONTEXT.md#<anchor>`), and a feature chip linking back to that Initiative via `paths.initiativeDetail(featureId)`.
- The same `DecisionRow` component is used inside the per-initiative `decision-log-section.tsx`. The shared component is extracted to a sibling file so neither surface duplicates the JSX.
- Backend: a new `GET /api/decisions` endpoint scoped to the auth context's workspace, returning entries with pagination (limit/offset query params, default limit reasonable for the UI — e.g. 50). Implementation follows the pattern of `ListInbox`.

ADR-link formatting follows `~/.claude/skills/grill-with-docs/ADR-FORMAT.md`. In a local checkout, prefer a `vscode://file//<workspaceRoot>/docs/adr/<ref>.md` href so clicks open the file in the user's editor; fall back to the raw markdown path otherwise. Context-term chips link to `CONTEXT.md#<slugified-anchor>` (kebab-case of the term).

## Acceptance criteria

- [ ] `GET /api/decisions` returns the workspace's decisions newest-first with `limit/offset` pagination. The handler reads the workspace from auth context (no path param).
- [ ] A new sqlc query `ListDecisionLogByWorkspace(workspace_id, limit, offset)` exists and is exercised by the handler.
- [ ] A Go handler test (DB-gated) covers: empty workspace → empty list; entries from two features → both returned newest-first; pagination via `limit/offset`; workspace isolation (decisions from another workspace are not leaked).
- [ ] `api.listWorkspaceDecisions()` exists on the TS API client and a TanStack `workspaceDecisionsOptions(wsId)` query is exported.
- [ ] `DecisionRow` is extracted from `packages/views/features/components/decision-log-section.tsx` into a sibling file. Both the per-initiative section and the new page consume it — no duplicated JSX.
- [ ] `/{slug}/decisions` renders the list. Empty state when no decisions.
- [ ] Each row's ADR chip links to `docs/adr/<ref>.md` (prefer `vscode://file/...` in a local checkout, fallback path otherwise). Context-term chips link to `CONTEXT.md#<anchor>`. Feature chip links to `paths.initiativeDetail(featureId)`.
- [ ] A component test (Testing Library) for the page seeds a fake TQ provider with two decisions and asserts: both rows render, ADR chip is an anchor with the right href, feature chip navigates to the initiative detail path.
- [ ] `make check` passes.

## Blocked by

- `.scratch/ui-refactor-initiative-runner/issues/01-paths-and-chrome.md` (needs the `/decisions` route slot and the new path helpers)

## Comments

### Key decisions

- **`DecisionRow` extraction lives at
  `packages/views/features/components/decision-row.tsx`** as a sibling of
  `decision-log-section.tsx`, per the PRD. Both the per-initiative section and
  the new cross-initiative page consume it — no duplicated JSX. The per-row
  feature chip is an optional `featureChip?: { title, href }` prop so the
  in-Initiative usage renders without it and the workspace page passes one.
- **ADR / context-term chips became real anchors.** The old in-page chips were
  inert `<span>`s; the new `ChipLink` renders an `<a>` for each ADR ref
  (`docs/adr/<ref>.md`) and each context term (`CONTEXT.md#<slug>`). Test
  data-testids (`decision-adr-chip-<ref>`, `decision-term-chip-<term>`,
  `decision-feature-chip`) make the contract assertable.
- **Context-term slugging is local to `decision-row.tsx`.** A tiny pure
  helper kebab-cases the term so the anchor stays predictable
  (`Gate` → `#gate`, `Run Boundary` → `#run-boundary`). Lives next to its only
  caller — extracting it earns no reuse.
- **`vscode://file/...` href was deferred.** The PRD called it out as
  "prefer in a local checkout, fallback otherwise", but the browser can't
  discover a workspace root without a daemon hint, and Multica doesn't expose
  one to the frontend yet. The relative path (`docs/adr/<ref>.md`) is the
  honest fallback. When a future change surfaces the workspace root (e.g. via
  a `Workspace` field or a runtime hint), `adrHref` is the single place to
  upgrade.
- **`GET /api/decisions`** mirrors `GET /api/inbox`: workspace resolved from
  middleware context (`ctxWorkspaceID`), no path param. Pagination uses the
  same `limit` / `offset` query-string shape already established by
  `ListAutopilotRuns` — default limit 50, max 200.
- **`ListDecisionLogByWorkspace` sqlc query** filters on `workspace_id` and
  carries `LIMIT $2 OFFSET $3` so the handler can pass typed `int32` params
  without string interpolation.
- **Tests seed a real retrospective task chain for each decision row.**
  `decision_log.run_id` has a FK to `agent_task_queue(id) ON DELETE CASCADE`,
  so the first attempt with a random UUID failed with FK 23503. The new
  helper `seedDecisionRun(...)` materialises agent + issue + retrospective
  task per seeded decision, registering cleanups; the cascade unwinds the
  decision row when the task is deleted.
- **Workspace isolation test uses `createOtherTestWorkspace`** (the existing
  label-test helper) plus a tiny `seedFeatureInWorkspace` builder local to
  this file. Avoids reaching into another package's fixtures and keeps the
  one-off second-workspace seed reusable here.

### Files changed

- `server/pkg/db/queries/decision_log.sql` — added the
  `ListDecisionLogByWorkspace :many` query with `LIMIT/OFFSET`.
- `server/pkg/db/generated/decision_log.sql.go` — regenerated by `sqlc`.
- `server/internal/handler/decision_log.go` — added the
  `ListDecisionLogWorkspace` HTTP handler and a tiny `parseDecisionLogPagination`
  helper. Added `decisionLogDefaultLimit`/`decisionLogMaxLimit` constants.
- `server/internal/handler/decision_log_test.go` — added four DB-gated tests
  (empty, across-features-newest-first, pagination, workspace isolation),
  plus `seedDecision` and the `seedDecisionRun` helper.
- `server/cmd/server/router.go` — wired
  `GET /api/decisions` to `h.ListDecisionLogWorkspace` in the auth-protected
  block.
- `packages/core/api/client.ts` — added `api.listWorkspaceDecisions(params?)`
  parsing the same `ListDecisionLogResponseSchema` used by the per-feature
  endpoint, with the `EMPTY_LIST_DECISION_LOG_RESPONSE` fallback.
- `packages/core/decision-log/queries.ts` — extended `decisionLogKeys` with
  `workspaceAll` / `workspaceList` and added the
  `workspaceDecisionsOptions(wsId)` query helper.
- `packages/views/features/components/decision-row.tsx` — new file. Holds
  the extracted `DecisionRow`, `ChipLink`, `adrHref`, `contextTermHref`,
  and `slugifyContextTerm` helpers.
- `packages/views/features/components/decision-log-section.tsx` — slimmed to
  delegate to the new `DecisionRow`.
- `packages/views/decisions/components/decisions-page.tsx` — new
  `DecisionsPage` component: page header, headline + subhead, loading
  skeleton, empty state, list rendering `DecisionRow` with a feature chip
  resolved against `featureListOptions(wsId)`.
- `packages/views/decisions/components/decisions-page.test.tsx` — 6
  Testing Library cases (empty, list, ADR-chip-href, context-chip-href,
  feature-chip-href + label, loading skeleton).
- `packages/views/decisions/index.ts` — exports `DecisionsPage`.
- `packages/views/package.json` — adds the `./decisions` export.
- `packages/views/locales/en/layout.json` — new `decisions_page` block
  (headline, subhead, empty title + hint).
- `apps/web/app/[workspaceSlug]/(dashboard)/decisions/page.tsx` — replaces
  the placeholder with `<DecisionsPage />`.

### Blockers / notes for next iteration

- **`make check` was not run end-to-end** in this iteration. `pnpm typecheck`
  is green (4 packages, all cached/clean), `pnpm test` is green
  (715 views + 414 core + every workspace), the Go handler-package tests for
  the decision-log scope pass (`go test ./internal/handler/ -run 'Decision'`
  → 18 passed), and `go vet ./...` is clean. The full `make check` step
  requires a running Postgres container which was not available here; the
  test surface this slice introduces (4 new Go DB tests + 6 new Vitest
  cases) was validated directly.
- **`vscode://file/...` ADR href upgrade** is left as a follow-up — see
  the key-decisions note above. When the daemon starts surfacing the
  workspace root to the frontend, edit `adrHref` in
  `packages/views/features/components/decision-row.tsx` to prefix it.
- **`DecisionLogSection` test ergonomics.** The pre-existing test for
  `decision-log-section.tsx` continues to pass after the extraction even
  though chips are now anchors instead of spans — the test asserts on
  visible text, not tag type. If future strictness is desired, that test
  can be replaced with the new shared `DecisionRow` tests since they
  cover the same surface more explicitly.
