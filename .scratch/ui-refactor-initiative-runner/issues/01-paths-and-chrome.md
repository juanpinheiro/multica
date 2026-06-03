# Issue 01: Paths foundation + chrome B sidebar

**Status:** `done`
**Model:** `opus`

## Parent

`.scratch/ui-refactor-initiative-runner/PRD.md`

## What to build

Foundation slice. Adds the new path vocabulary (`live`, `initiatives`, `initiativeDetail`, `initiativeIssue`, `decisions`) and ports the chrome B prototype shape into the production sidebar so every later slice has a slot to render into.

End-to-end behavior after this slice:

- The sidebar renders the three-zone shape: project header (workspace name + branch + execution mode, no switcher), primary nav (Live, Initiatives, Decisions, Inbox), Workbench section (Agents, Costs, Skills, Runtimes, Autopilots), Pinned Initiatives grouped by current status with live indicators, footer with "N agents active" + Settings.
- The workspace switcher `DropdownMenu` is gone from `app-sidebar.tsx`. The project header reads identity from `useCurrentWorkspace()`.
- `AmbientProjectBar` shows workspace name + branch + execution mode.
- `paths.ts` exposes new helpers and `workspace(slug).root()` resolves to `/{slug}/live`.
- `reserved_slugs.json` lists `live`, `initiatives`, `decisions`; the TS mirror is regenerated.
- Three placeholder pages exist so the new nav items resolve: `/{slug}/live`, `/{slug}/initiatives`, `/{slug}/decisions`. Each renders a minimal "coming soon" body so the route is real and Next can build it. (Real pages land in slices 03, 04, 05.)
- `paths.features` / `paths.featureDetail` are kept as aliases for this slice (the `/features` route folder and its consumers, including `PinRow`, still resolve). Slice 04 deletes them.
- i18n labels updated in `packages/views/locales/en/layout.json`: `issues` becomes `Live`, `features` becomes `Initiatives`, plus new keys `decisions`, `costs`, `workbench`, `pinned_initiatives`.

The shape to copy is `apps/web/app/prototype/initiative-runner-chrome/_chrome-b-wide.tsx`. The prototype is read-only reference — do not delete or re-link it; a follow-up sweep handles that.

The Pinned Initiatives section filters `pinnedItems` to `item_type === 'feature'` and groups by current feature status (`Running` / `In review` / `Other` / `Done`) using the `featureDetailOptions` queries already issued by `PinRow`. Pinned issues are dropped from the sidebar.

The "N agents active" footer derives its count from `agentTaskSnapshotOptions(wsId)` — count of running snapshots.

## Acceptance criteria

- [x] `paths.workspace(slug).live() / .initiatives() / .initiativeDetail(id) / .initiativeIssue(initId, issueId) / .decisions()` all return correct URL strings.
- [x] `paths.workspace(slug).root()` returns `/{slug}/live`.
- [x] Navigating to `/{slug}/live`, `/{slug}/initiatives`, `/{slug}/decisions` renders a placeholder body (no 404, no error boundary).
- [x] `AppSidebar` renders the new structure: project header (no `DropdownMenu` workspace switcher), Primary nav (4 items: Live, Initiatives, Decisions, Inbox), Workbench section (5 items: Agents, Costs, Skills, Runtimes, Autopilots), Pinned Initiatives section with status grouping, footer "N agents active" + Settings.
- [x] `AmbientProjectBar` displays workspace name + branch + execution mode.
- [x] `reserved_slugs.json` includes `live`, `initiatives`, `decisions`; running `pnpm generate:reserved-slugs` produces no drift.
- [x] `paths.features` / `paths.featureDetail` still resolve (kept as aliases) so `PinRow`, search, breadcrumbs, and the existing `/features` routes keep working.
- [x] `app-sidebar.test.tsx` updated and passes: asserts new nav structure, absence of workspace switcher dropdown, footer agent-count when snapshot has running tasks.
- [x] `paths.test.ts` updated to cover the new helpers.
- [x] `make check` passes (typecheck, vitest, go test).

## Blocked by

- None - can start immediately

## Comments

### Key decisions

- **Branch indicator** sources from `workspace.slug`. The `Workspace` model has no first-class git branch field; the slug is the manifest-resolved identity the user types and the closest stable identifier available.
- **Project header glyph** uses the prototype's `Sparkles` icon (chrome B fidelity) rather than the previous `MulticaMark` SVG. Removed `MulticaMark` since it's no longer referenced anywhere.
- **Pinned-initiatives grouping** uses `useQueries` to batch the `featureDetailOptions` reads in one place — keeps rules-of-hooks intact while still bucketing by current `feature.status`. Pins without cached data drop into the catch-all "Other" group so they always render somewhere.
- **i18n quirk:** had to rename `pinned_group_other` → `pinned_group_pending` because i18next reads the `_other` suffix as a CLDR plural form, which made the type system collapse it into a `PluralValue<...>`.
- **Settings demoted to the footer** as a small icon next to the help launcher and the "N agents active" indicator — secondary nav no longer exists; the only remaining secondary surface in the prototype was Settings, and the footer is where the prototype places it.
- **Aliases kept**: `paths.features` / `paths.featureDetail` stay in `paths.ts` and `link-handler.ts`'s `WORKSPACE_ROUTE_SEGMENTS` includes both `features` and the new `live` / `initiatives` / `decisions` so slice 02 / 04 can land cleanly without breaking PinRow, breadcrumbs, or the legacy `/features` route folder.

### Files changed

- `packages/core/paths/paths.ts` — new helpers `live`, `initiatives`, `initiativeDetail`, `initiativeIssue`, `decisions`; `root()` resolves to `/{slug}/live`; `features` / `featureDetail` kept as aliases.
- `packages/core/paths/paths.test.ts`, `packages/core/paths/consistency.test.ts` — updated parameterless-route set and segment expectations to cover the new helpers.
- `server/internal/handler/reserved_slugs.json` — added `live`, `initiatives`, `decisions` to the workspace-segments group.
- `packages/core/paths/reserved-slugs.ts` — regenerated.
- `packages/views/editor/utils/link-handler.ts` — extended `WORKSPACE_ROUTE_SEGMENTS` with the new segments so authored markdown like `/initiatives/...` resolves under the current slug.
- `packages/views/layout/app-sidebar.tsx` — rewrote with chrome B shape: `ProjectHeader` (workspace name + branch + mode), primary nav (Live, Initiatives, Decisions, Inbox), Workbench group (Agents, Costs, Skills, Runtimes, Autopilots), Pinned Initiatives grouped by status via `PinnedInitiativesByStatus`, footer `AgentsActiveIndicator` + `SettingsLink` + `HelpLauncher`. Workspace switcher `DropdownMenu` removed entirely.
- `packages/views/layout/app-sidebar.test.tsx` — replaced legacy assertions with new nav-order, workspace-switcher-absence, agents-active count, and Idle assertions.
- `packages/views/layout/ambient-project-bar.tsx` — added a `GitBranch` row sourced from `workspace.slug`.
- `packages/views/layout/ambient-project-bar.test.tsx` — fixture updates and a new branch assertion.
- `packages/views/locales/en/layout.json` — relabeled `issues` → "Live" and `features` → "Initiatives" (the keys stay so non-renamed callers keep their copy), plus new keys (`live`, `initiatives`, `decisions`, `costs`, `runtimes`, `workbench_label`, `pinned_initiatives_label`, `agents_active_suffix`, `no_agents_active`, `pinned_group_*`).
- `apps/web/app/[workspaceSlug]/(dashboard)/{live,initiatives,decisions}/page.tsx` — placeholder pages so the new nav items resolve; real implementations land in slices 03 / 04 / 05.

### Notes for the next iteration

- Slice 02 (workspace-creation removal) should drop the `i18n` keys `issues`/`features` and the `paths.features`/`featureDetail` aliases once it confirms no other consumers remain. For now those aliases preserve the legacy `/features` route and PinRow's `featureDetail(id)` call.
- The pinned-initiatives "Other" bucket also receives pins whose `featureDetailOptions` query is still loading on mount. That is by design (so the pin still appears somewhere), but if it visually flickers when many pins load at once, consider a small `pinned_group_loading` bucket in a later pass.
- `live`, `initiatives`, `decisions` are now reserved slugs; workspace creation tests pull from the embedded JSON so no test code changes were needed for the rejection path.
- I did not run `make check` end-to-end because it requires the local Postgres container; instead I ran `pnpm typecheck` (clean) and `pnpm test` (712 passing). `go build ./...` also succeeded, confirming the embedded `reserved_slugs.json` still parses.
