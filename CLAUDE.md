# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Naming Conventions

### Routes

Pre-workspace routes (the routes that exist before the user is in a workspace) MUST use either a single word or the `/{noun}/{verb}` pattern.

- OK: `/login`, `/inbox`, `/workspaces/new`
- NOT OK: `/new-workspace`, `/create-team`, `/accept-invite`

Hyphenated word groups at the root collide with user-chosen workspace slugs and force endless reserved-slug audits. Reserving the noun (`workspaces`) automatically protects the entire `/workspaces/*` subtree.

### Workspace-scoped routes

Always live under `/{slug}/{section}` — `/{slug}/issues`, `/{slug}/agents`, `/{slug}/settings`. Use `useNavigation().push()` from shared code, never duplicate workspace routing logic.

### Files and components

- Files: `kebab-case.tsx` / `kebab-case.ts` (e.g. `agent-row-actions.tsx`)
- Components: `PascalCase` (e.g. `AgentRowActions`)
- Hooks: `useCamelCase` (e.g. `useWorkspaceId`)
- Tests: colocated as `<file>.test.ts(x)`
- Stores (Zustand): `<feature>-store.ts`, exported as `use<Feature>Store`

### Database (Go + sqlc)

- Tables: `snake_case` singular (`user`, `workspace`, `agent_runtime`)
- Columns: `snake_case` (`workspace_id`, `created_at`, `last_seen_at`)
- Foreign keys: `<table>_id`
- Booleans: `is_<state>` or `<state>_at` (timestamp form preferred for state changes)
- Migration files: `NNN_descriptive_name.up.sql` + `.down.sql` — always provide both directions

### Go

- Standard `gofmt` + `go vet`. No exceptions.
- Handler files mirror domain: `agent.go`, `auth.go`, `runtime.go`
- Tests: `<file>_test.go` colocated
- For UUID parsing in handlers follow the rules in the "Backend Handler UUID Parsing Convention" section below.

### TypeScript

- API responses on the wire are `snake_case`; the api client converts to `camelCase` at the boundary. Inside TS code, **always camelCase**.
- Types: `PascalCase` (`Issue`, `AgentRuntime`); never `IPrefix`, never `_t` suffix.
- Enums: prefer string literal unions; reserve `enum` for runtime-iterable cases.
- TanStack Query keys: factory functions in `<feature>/queries.ts`, e.g. `issueKeys.detail(id)`.

### Issue keys

Every issue has a human-readable key like `MUL-123`: workspace `issue_prefix` (uppercase letters and digits, typically 3 chars, max 10) + sequence number. Workspace admins can change the prefix in Settings → General; changing it renumbers every existing issue, so external references that embed the old prefix (PR titles, branch names, links in docs and chat) stop resolving.

### Comments in code

English only.

### Commit messages

Conventional format: `feat(scope)`, `fix(scope)`, `refactor(scope)`, `docs`, `test(scope)`, `chore(scope)`. Atomic commits grouped by intent.

## Project Context

Multica is a personal AI agent management platform — like Linear, but with AI agents as first-class citizens and a single implicit user. Agents are assigned issues, create issues, comment, and change status. The local daemon runs on the developer's machine and executes work via whatever agent CLIs are on PATH (Claude Code, Codex, Gemini, etc.).

## Architecture

**Go backend + monorepo frontend (pnpm workspaces + Turborepo) with shared packages.**

- `server/` — Go backend (Chi router, sqlc for DB, gorilla/websocket for real-time)
- `apps/web/` — Next.js frontend (App Router)
- `packages/core/` — Headless business logic (zero react-dom)
- `packages/ui/` — Atomic UI components (zero business logic)
- `packages/views/` — Shared business pages/components (web only)
- `packages/tsconfig/` — Shared TypeScript configuration

### Key Architectural Decisions

**Internal Packages pattern** — all shared packages export raw `.ts`/`.tsx` files (no pre-compilation). The consuming app's bundler compiles them directly. This gives zero-config HMR and instant go-to-definition.

**Dependency direction:** `views/ → core/ + ui/`. Core and UI are independent of each other. `packages/views/` is web-only and may import `next/navigation` and `next/link` directly.

**Platform bridge:** `packages/core/platform/` provides `CoreProvider` — initializes API client, auth/workspace stores, WS connection, and QueryClient. The web app wraps its root with `<CoreProvider>` and `<NavigationProvider>` (from `@multica/views/navigation`).

**pnpm catalog** — `pnpm-workspace.yaml` defines `catalog:` for version pinning. All shared deps use `catalog:` references to guarantee a single version across all packages. When adding new shared deps (including test deps), add to catalog first.

### State Management

The architecture relies on a strict split between server state and client state. Mixing them is the most common way to break it.

- **TanStack Query owns all server state.** Issues, users, workspaces, inbox — anything fetched from the API lives in the Query cache. WS events keep it fresh via invalidation; no polling, no `staleTime` workarounds.
- **Zustand owns all client state.** UI selections, filters, drafts, modal state, navigation history. Stores live in `packages/core/` (never in `packages/views/`) so they're shared.
- **React Context** is reserved for cross-cutting platform plumbing — `WorkspaceIdProvider`, navigation transition state. Don't reach for it for general state.
- **Auth and workspace stores are the only stores allowed to call `api.*` directly**, because they manage critical state that must exist before queries can run. They're created via factory + injected dependencies, registered by the platform layer.

**Hard rules — these are how the architecture stays coherent:**

- **Never duplicate server data into Zustand.** If it came from the API, it belongs in the Query cache. Copying it into a store creates two sources of truth and they will drift.
- **Workspace-scoped queries must key on `wsId`.** This is what makes workspace switching automatic — the cache key changes, the right data appears, no manual invalidation needed.
- **Mutations are optimistic by default.** Apply the change locally, send the request, roll back on failure, invalidate on settle. The user shouldn't wait for the server.
- **WS events invalidate queries — they never write to stores directly.** This keeps the cache as the single source of truth and avoids race conditions.
- **Persist what's worth preserving across restarts** (user preferences, drafts, tab layout). **Don't persist ephemeral UI state** (modal open/close, transient selections) or server data.

**Common Zustand footguns to avoid:**

- Selectors must return stable references. Returning a freshly built object or array on every call (e.g. `s => ({ a: s.a, b: s.b })` or `s => s.items.map(...)`) triggers infinite re-renders. Either select primitives separately or use shallow comparison.
- Hooks that need workspace context should accept `wsId` as a parameter, not call `useWorkspaceId()` internally — this lets them work outside the `WorkspaceIdProvider` (e.g. in a sidebar that renders before workspace is loaded).

## Commands

```bash
# One-command dev (auto-setup + start everything)
make dev              # Auto-creates env, installs deps, starts DB, migrates, launches app

# Explicit setup & run (if you prefer separate steps)
make setup            # First-time: ensure shared DB, create app DB, migrate
make start            # Start backend + frontend together
make stop             # Stop app processes for the current checkout
make db-down          # Stop the shared PostgreSQL container

# Frontend (all commands go through Turborepo)
pnpm install
pnpm dev:web          # Next.js dev server (port 3000)
pnpm build            # Build all frontend apps
pnpm typecheck        # TypeScript check (all packages + apps via turbo)
pnpm lint             # ESLint
pnpm test             # TS tests (Vitest, all packages + apps via turbo)

# Backend (Go)
make server           # Run Go server only (port 8080)
make daemon           # Run local daemon
make build            # Build server + CLI binaries to server/bin/
make cli ARGS="..."   # Run multica CLI (e.g. make cli ARGS="config")
make test             # Go tests
make sqlc             # Regenerate sqlc code after editing SQL in server/pkg/db/queries/
make migrate-up       # Run database migrations
make migrate-down     # Rollback migrations

# Run a single TS test (works for any package with a test script)
pnpm --filter @multica/views exec vitest run autopilots/components/autopilot-dialog-i18n.test.ts
pnpm --filter @multica/core exec vitest run i18n/pick-locale.test.ts
pnpm --filter @multica/web exec vitest run lib/locale-routing.test.ts

# Run a single Go test
cd server && go test ./internal/handler/ -run TestName

# shadcn — config lives in packages/ui/components.json (Base UI variant, base-nova style)
pnpm ui:add badge                # Adds component to packages/ui/components/ui/

# Infrastructure
make db-up            # Start shared PostgreSQL (pgvector/pg17 image)
make db-down          # Stop shared PostgreSQL
make db-reset         # Drop + recreate current env's DB, then re-run migrations (local only; stop backend first)
```

### CI Requirements

CI runs on Node 22 and Go 1.26.1 with a `pgvector/pgvector:pg17` PostgreSQL service. See `.github/workflows/ci.yml`.

### Worktree Support

All checkouts share one PostgreSQL container. Isolation is at the database level — each worktree gets its own DB name and unique ports via `.env.worktree`. Main checkouts use `.env`.

`make dev` auto-detects worktrees and handles everything. For explicit control:

```bash
make worktree-env       # Generate .env.worktree with unique DB/ports
make setup-worktree     # Setup using .env.worktree
make start-worktree     # Start using .env.worktree
```

## Coding Rules

- TypeScript strict mode is enabled; keep types explicit.
- Go code follows standard Go conventions (gofmt, go vet).
- Keep comments in code **English only**.
- Prefer existing patterns/components over introducing parallel abstractions.
- Unless the user explicitly asks for backwards compatibility, do **not** add compatibility layers, fallback paths, dual-write logic, legacy adapters, or temporary shims **for internal, non-boundary code** (a function calling another function in the same package, a component reading its own state, a store helper, etc.).
- If a flow or API is being replaced and the product is not yet live, prefer removing the old path instead of preserving both old and new behavior.
- Avoid broad refactors unless required by the task.
- New global (pre-workspace) routes MUST use a single word (`/login`, `/inbox`) or a `/{noun}/{verb}` pair (`/workspaces/new`). NEVER add hyphenated word-group root routes (`/new-workspace`, `/create-team`) — they collide with common user workspace names and force endless reserved-slug audits. Reserving the noun (`workspaces`) automatically protects the entire `/workspaces/*` subtree.
- The reserved-slug list lives in **one** place: `server/internal/handler/reserved_slugs.json`. The Go side embeds the JSON; `packages/core/paths/reserved-slugs.ts` is generated from it by `pnpm generate:reserved-slugs`. Edit the JSON, run the generator, commit both. CI re-runs the generator and fails on any drift, so a stale TS file cannot land.

### API Response Compatibility

Untyped JSON crossing the network is not `T`. Defensive parsing keeps the frontend resilient to backend drift and avoids white-screens when a response shape changes:

- **Parse, don't cast.** Use `parseWithFallback` in `packages/core/api/schema.ts` with a `zod` schema and an explicit fallback. On validation failure it logs a warning and returns the fallback; it never throws into the UI.
- **No bare `as` casts on response bodies.** Every endpoint method whose response is consumed by UI logic must run through a schema before returning.
- **Optional-chain and default everywhere downstream.** Treat every field as possibly missing. Use explicit boolean checks (`=== true`) over truthy/falsy negation, which silently treats `undefined` and `null` as `false`.
- **Don't pin a UI affordance to a single backend field.** If a button or indicator depends on exactly one boolean from the server, a backend bug deletes it. Combine signals (cursor presence, page length, etc.) so the affordance stays available in the worst case.
- **Enum drift downgrades, not crashes.** A new server-side enum value should render a generic fallback. `switch` statements on server-driven strings must have a `default` branch.
- **When you add or change an endpoint:** add the schema in the same PR, and write at least one test that feeds a malformed response through it (missing field, wrong type, `null` array). The test fails closed if a future change breaks the contract.

### Backend Handler UUID Parsing Convention

Every Go handler in `server/internal/handler/` follows these rules. The convention exists because `util.ParseUUID` used to silently return a zero UUID on invalid input, which caused #1661 — a `DELETE` returning 204 success while the SQL `DELETE` matched zero rows.

- **Resource path params that accept either a UUID or a human-readable identifier** (e.g. `chi.URLParam(r, "id")` for an issue, which accepts both `MUL-123` and a UUID) MUST be resolved through the dedicated loader (`loadIssueForUser` / `loadSkillForUser` / `loadAgentForUser` / `requireDaemonRuntimeAccess`). After resolution, all subsequent DB calls — especially `Queries.Delete*` / `Queries.Update*` — MUST use `entity.ID` from the resolved object. Never round-trip the raw URL string through `parseUUID` for a write query.
- **Pure-UUID inputs from request boundaries** (URL params that are always UUIDs, request body fields, query params, headers) MUST be validated with `parseUUIDOrBadRequest(w, s, fieldName)`. On invalid input it writes a 400 and returns `ok=false` — return immediately.
- **Trusted UUID round-trips** (sqlc-returned UUIDs being passed back into queries, test fixtures) use `parseUUID(s)` which calls `util.MustParseUUID` and panics on invalid input. A panic here means an unguarded user-input string slipped in — that is a real bug. `chi`'s `middleware.Recoverer` translates the panic into a 500 so the process keeps running.
- **`util.ParseUUID(s) (pgtype.UUID, error)`** is the only safe variant outside the handler package. Always check the error.

When adding a `Queries.Delete*` or `Queries.Update*` call, ask: "Where did this UUID come from?" If the answer is "raw user input that hasn't been validated," route it through `parseUUIDOrBadRequest` or a loader first.

### Dependency Declaration Rule

Every workspace (`apps/` and `packages/` directories) must explicitly declare all directly imported external packages in its own `package.json`. Relying on pnpm hoist to resolve undeclared imports (phantom deps) is prohibited — it causes production build failures when pnpm creates peer-dep variants.

- Use `"pkg": "catalog:"` to reference the shared version from `pnpm-workspace.yaml`.
- CI enforces this via `eslint-plugin-import-x/no-extraneous-dependencies`.

### Package Boundary Rules

These are hard constraints. Violating them breaks the architecture:

- `packages/core/` — zero react-dom, zero localStorage (use StorageAdapter), zero process.env, zero UI libraries, zero `next/*` imports. **Shared Zustand stores live here**, even view-related ones (filters, view modes) — stores are pure state, not UI.
- `packages/ui/` — zero `@multica/core` imports (pure UI, no business logic), zero `next/*` imports.
- `packages/views/` — web-only. May import `next/navigation` and `next/link` directly. Stores still live in `packages/core/`.

### Web Development Rules

When adding a new page or feature:

1. **New page component** → add to `packages/views/<domain>/`. Use `next/navigation` directly (or the `useNavigation()` / `<AppLink>` helpers in `@multica/views/navigation` if you want the global transition signal).
2. **Wire it** → add a route in `apps/web/app/` (Next.js page file) that renders the view.
3. **Navigation transition signal** → `useNavigation()` from `@multica/views/navigation` wraps `router.push/replace` in `useTransition` so the global `<NavigationProgress />` bar lights up. Plain `router.push` from `next/navigation` bypasses it.
4. **Shared guards/providers** → use `DashboardGuard` from `packages/views/layout/`.
5. **New hooks that need workspace context** → accept `wsId` as parameter instead of reading from `useWorkspaceId()` Context, so they work both inside and outside `WorkspaceIdProvider`.

### CSS Architecture

- **Design tokens** → use semantic tokens (`bg-background`, `text-muted-foreground`). Never use hardcoded Tailwind colors (`text-red-500`, `bg-gray-100`).
- **Shared styles** → `packages/ui/styles/`. Never duplicate scrollbar styling, keyframes, or base layer rules in app CSS.
- **`@source` directives** → the web app scans shared packages so Tailwind sees all class names.

## UI/UX Rules

- Prefer shadcn components over custom implementations. Install via `pnpm ui:add <component>` from project root — adds to `packages/ui/components/ui/`. All components use Base UI primitives (`@base-ui/react`), not Radix.
- Use shadcn design tokens for styling. Avoid hardcoded color values.
- Do not introduce extra state (useState, context, reducers) unless explicitly required by the design.
- Pay close attention to **overflow** (truncate long text, scrollable containers), **alignment**, and **spacing** consistency.

## Testing Rules

### Where to write tests

Tests follow the code, not the app. This is the most important testing principle in this monorepo:

| What you're testing | Where the test lives | Why |
|---|---|---|
| Shared business logic (stores, queries, hooks) | `packages/core/*.test.ts` | No DOM needed, pure logic |
| Shared UI components (pages, forms, modals) | `packages/views/*.test.tsx` | jsdom, no framework mocks |
| Platform-specific wiring (cookies, redirects, searchParams) | `apps/web/*.test.tsx` | Needs framework-specific mocks |

**Never test shared component behavior in an app's test file.** If a test requires mocking framework-specific behavior (cookies, server actions) to test a component from `@multica/views`, the test is in the wrong place — move it to `packages/views/` and mock `@multica/core` instead.

### Test infrastructure

- `packages/core/` — Vitest, Node environment (no DOM)
- `packages/views/` — Vitest, jsdom environment, `@testing-library/react`
- `apps/web/` — Vitest, jsdom environment, framework-specific mocks
- `server/` — Go standard `go test`

All test deps are in the pnpm catalog for unified versioning.

### Mocking conventions

- Mock `@multica/core` stores with `vi.hoisted()` + `Object.assign(selectorFn, { getState })` pattern (Zustand stores are both callable and have `.getState()`).
- Mock `@multica/core/api` for API calls.
- In `packages/views/` tests: never mock `next/*` or `react-router-dom` — those don't exist here.
- In `apps/web/` tests: mock framework-specific APIs only for platform-specific behavior.

### TDD workflow

1. Write failing test in the **correct package** first.
2. Write implementation.
3. Run `pnpm test` (Turborepo discovers all packages).
4. Green → done.

### Go tests

Standard `go test`. Tests should create their own fixture data in a test database.

## Commit Rules

- Use atomic commits grouped by logical intent.
- Conventional format: `feat(scope)`, `fix(scope)`, `refactor(scope)`, `docs`, `test(scope)`, `chore(scope)`.

## Minimum Pre-Push Checks

```bash
make check    # Runs all checks: typecheck, unit tests, Go tests
```

Run verification only when the user explicitly asks for it.

For targeted checks when requested:
```bash
pnpm typecheck        # TypeScript type errors only
pnpm test             # TS unit tests only (Vitest, all packages)
make test             # Go tests only
```

## AI Agent Verification Loop

After writing or modifying code, always run the full verification pipeline:

```bash
make check
```

**Workflow:**
- Write code to satisfy the requirement
- Run `make check`
- If any step fails, read the error output, fix the code, and re-run
- Repeat until all checks pass
- Only then consider the task complete

**Quick iteration:** If you know only TypeScript or Go is affected, run individual checks first for faster feedback, then finish with a full `make check` before marking work complete.

## Multi-tenancy

All queries filter by `workspace_id`. A singleton implicit user owns all data; workspaces exist as organizational folders. The `X-Workspace-ID` header routes requests to the correct workspace.

## Agent Assignees

Assignees are polymorphic — can be a member or an agent. `assignee_type` + `assignee_id` on issues. Agents render with distinct styling (purple background, robot icon).

## Agent skills

### Issue tracker

Issues and PRDs live as markdown files under `.scratch/<feature-slug>/`. See `docs/agents/issue-tracker.md`.

### Triage labels

Five canonical triage roles (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`) used verbatim as `Status:` values in issue files. See `docs/agents/triage-labels.md`.

### Domain docs

Multi-context monorepo: `CONTEXT-MAP.md` at root points to per-area `CONTEXT.md` files (server, apps/*, packages/*). See `docs/agents/domain.md`.
