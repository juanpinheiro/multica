# Issue 07: AmbientProjectBar (workspace · mode · manifest)

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/web-execution-monitor/PRD.md`

## What to build

A thin, read-only **`AmbientProjectBar`** mounted in the dashboard shell that shows the current workspace, its execution mode (`worktree` | `in_place`), and that it is manifest-anchored — promoting the execution mode from the buried feature-detail badge to a persistent ambient signal, so the owner always knows which project they are looking at and whether it runs serially. The mode value comes from the workspace record (already typed; an absent mode is treated as `worktree`). The bar sets no state and offers no control — the manifest is the only place the mode is set.

## Acceptance criteria

- [x] An `AmbientProjectBar` renders the workspace name, the execution mode, and a manifest-provenance hint.
- [x] `in_place` and `worktree` render distinctly; an absent/undefined mode renders as `worktree`.
- [x] The bar is read-only (no control that mutates the mode).
- [x] It is mounted in the dashboard shell so it shows across workspace-scoped routes.
- [x] View render test covering `worktree`, `in_place`, and absent-mode cases.

## Blocked by

- None - can start immediately.

## Comments

### Key decisions

1. **`ModeTag` is a pure presentational sub-component** — accepts `mode: "worktree" | "in_place"` and renders with semantic tokens (`bg-warning/10 text-warning` for `in_place`, `bg-muted text-muted-foreground` for `worktree`), with a `Layers2` icon only for the serial mode. `data-testid="ambient-mode-tag"` for test assertions.

2. **No i18n** — the mode labels (`worktree`, `in_place`) and manifest hint (`via .multica`) are technical system identifiers, consistent with how they appear in the codebase as constant values.

3. **Null guard** — `AmbientProjectBar` returns `null` when `useCurrentWorkspace()` is null, even though `DashboardGuard` guarantees a workspace inside `DashboardLayout`. Keeps the component composable and safe outside the guard.

4. **Placement in `DashboardLayout`** — mounted after `NavigationProgress` (absolute overlay, 0 height) and before `{children}`, so it renders as a static strip above every page's own `PageHeader`. The `h-7` height keeps it thin and unobtrusive.

### Files changed

- `packages/views/layout/ambient-project-bar.tsx` — new component (`ModeTag`, `AmbientProjectBar`)
- `packages/views/layout/ambient-project-bar.test.tsx` — 5 table-driven tests (TDD-first)
- `packages/views/layout/dashboard-layout.tsx` — `AmbientProjectBar` mounted in `SidebarInset`

### Verification

- `pnpm --filter @multica/views exec vitest run` → 730 passed (86 test files)
- `pnpm --filter @multica/views run typecheck` → clean
