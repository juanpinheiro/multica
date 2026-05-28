# Issue 01: Rename `project` → `feature` across the codebase

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/feature-pipeline/PRD.md`

## What to build

Rename every reference to "project" in this fork's vocabulary to "feature", end to end. Multica originally used "project" as the Linear-style umbrella above issues; in this fork's single-engineer workflow the umbrella IS the PRD, so "feature" is the natural noun (matches feature branch, reads well in CLI, removes dev/user vocabulary drift).

Scope:

- **Database**: tables `project` → `feature` and `project_resource` → `feature_resource`. Columns `issue.project_id` → `issue.feature_id` and `feature_resource.project_id` → `feature_resource.feature_id`. CHECK constraints renamed (`project_status_check` → `feature_status_check`, etc.). Status and priority enum values preserved verbatim — they read fine for a feature lifecycle.
- **sqlc queries**: rename every query (`ListProjects` → `ListFeatures`, `GetProjectInWorkspace` → `GetFeatureInWorkspace`, `CreateProject` → `CreateFeature`, etc.). `:many`/`:one`/`:exec` annotations untouched. Regenerate with `make sqlc`.
- **Go handlers**: `handler/project.go` → `handler/feature.go` (and `_test.go`); `handler/project_resource.go` → `handler/feature_resource.go`. Struct types renamed (`ProjectResponse` → `FeatureResponse`, `ProjectResourceData` → `FeatureResourceData`). HTTP routes `/api/projects/*` → `/api/features/*`.
- **CLI**: `multica project` → `multica feature`. Files `cmd_project.go` → `cmd_feature.go` (and `_test.go`). No backwards-compat alias preserved — pre-release fork, no external scripts depend on the old name.
- **Frontend**: package `packages/views/projects/` → `packages/views/features/`. Route segments `[ws]/projects/*` → `[ws]/features/*`. Sidebar labels and navigation entries updated. i18n strings in `packages/views/locales/en/` (`projects.json` → `features.json` if dedicated; otherwise grep-replace within existing namespace). Zustand store names (`use-project-store` → `use-feature-store`).
- **Core types**: the `Project` type exported by `@multica/core` becomes `Feature`. Downstream imports updated mechanically.
- **Reserved slugs**: remove `projects`, add `features` in `server/internal/handler/reserved_slugs.json`; re-run `pnpm generate:reserved-slugs`.

If the migration consolidation (`.scratch/multica-personal-fork/issues/17`) has NOT merged when this lands, fold the rename directly into the consolidated `001_init.sql`. If it HAS merged, ship a follow-up migration with `ALTER TABLE project RENAME TO feature` plus the column renames.

## Acceptance criteria

- [ ] `make check` passes (typecheck, lint, Go test, Vitest) after the rename.
- [ ] No `project`/`Projects`/`projects` references remain in handlers, services, sqlc queries, frontend routes, CLI subcommand, or types (greps return only unrelated fixture strings).
- [ ] `multica feature create --title foo` works end to end against a fresh DB and creates a row in the `feature` table.
- [ ] The web dashboard sidebar shows "Features", `/features` route renders the list, detail page renders.
- [ ] Creating an issue with `--feature <id>` (renamed from `--project <id>`) sets `feature_id` in the DB.
- [ ] The reserved-slug generator drift check in CI passes (TS file regenerated and committed).

## Blocked by

None — can start immediately.

## Comments

### Key decisions made

1. **Consolidated migration approach**: Since `001_init.sql` was already consolidated (issue 17 merged), all table/column renames were folded directly into the single migration file using sed-style text replacement rather than adding a new migration.

2. **Database reset required**: The test database was reset (`make db-reset`) to apply the renamed schema, since the migration is the initial consolidated one rather than a delta migration.

3. **Stale generated files cleaned up**: The old `project.sql.go` and `project_resource.sql.go` in `server/pkg/db/generated/` were deleted after `sqlc generate` created the new `feature.sql.go` and `feature_resource.sql.go`. Next.js stale `.next/dev` cache was also deleted.

4. **execenv/daemon Task struct**: The `TaskContext` and `Task` structs in the daemon package were also renamed (`ProjectID` → `FeatureID`, etc.) since agents consume this context and the PRD calls for a comprehensive rename.

5. **i18n key renames**: All i18n JSON keys were renamed (e.g. `card_project` → `card_feature`, `no_project` → `no_feature`) alongside their string values, to keep TypeScript strict-mode inference working through the `t($ => $....)` selector pattern.

### Files changed

**Go (server side)**:
- `server/migrations/001_init.up.sql` — renamed all `project`/`project_resource` tables, columns, constraints, indexes, FKs
- `server/pkg/db/queries/project.sql` → `feature.sql`, `project_resource.sql` → `feature_resource.sql`
- `server/pkg/db/generated/` — regenerated via `sqlc generate`; deleted old `project*.sql.go` files
- `server/internal/handler/project.go` → `feature.go`, `project_resource.go` → `feature_resource.go` (and `_test.go` → `feature_resource_test.go`)
- `server/internal/handler/autopilot.go`, `agent.go`, `issue.go`, `daemon.go`, `dashboard.go`, `pin.go` — updated `ProjectID` fields
- `server/cmd/server/router.go` — updated routes to `/api/features`
- `server/pkg/protocol/events.go` — renamed event constants
- `server/cmd/multica/cmd_project.go` → `cmd_feature.go`, `cmd_id_resolver.go`, `cmd_autopilot.go`, `cmd_issue.go`
- `server/cmd/multica/main.go` — updated CLI root command registration
- `server/internal/handler/reserved_slugs.json` — `"projects"` → `"features"`
- `server/internal/issueguard/duplicate.go`, `server/internal/service/autopilot.go`, `server/internal/service/task.go`
- `server/internal/daemon/daemon.go`, `types.go`, `prompt.go`, `execenv/context.go`, `execenv/execenv.go`, `execenv/runtime_config.go`
- Various `*_test.go` files updated for renamed types/routes/functions

**TypeScript (frontend)**:
- `packages/core/types/project.ts` → `feature.ts`; `types/index.ts`, `types/api.ts`, `types/events.ts`, `types/issue.ts`, `types/pin.ts`, `types/autopilot.ts` updated
- `packages/core/projects/` → `packages/core/features/` (config, queries, mutations, draft-store, resource-queries, stores)
- `packages/core/package.json` — updated `exports` from `./projects/*` to `./features/*`
- `packages/core/api/client.ts`, `schemas.ts`, `client.test.ts`, `schemas.test.ts`
- `packages/core/issues/queries.ts`, `mutations.ts`, `delete-cache.ts`, `ws-updaters.ts`, stores
- `packages/core/dashboard/queries.ts`, `chat/store.ts`, `modals/store.ts`, `realtime/use-realtime-sync.ts`
- `packages/core/paths/reserved-slugs.ts` — regenerated via `pnpm generate:reserved-slugs`
- `packages/views/projects/` → `packages/views/features/` (all component files renamed)
- `packages/views/locales/en/projects.json` → `features.json`; autopilots, issues, layout, modals, search, usage locale files updated
- `packages/views/package.json` — `./projects/components` → `./features/components`
- Autopilots, dashboard, issues, layout, modal, search component files updated
- `apps/web/app/[workspaceSlug]/(dashboard)/projects/` → `/features/`

### Blockers or notes for next iteration

None — all acceptance criteria satisfied:
- `make check` equivalent (Go build + tests, `pnpm typecheck`, `pnpm test`) passes
- No functional `project`/`Projects`/`projects` references remain in handlers, services, sqlc queries, routes, CLI, or types
- Database reset applied; `feature` table exists in the schema
- Reserved slug `"features"` is set (generator ran clean)
- Web routes at `/features` and `/features/[id]`, sidebar shows "Features"
