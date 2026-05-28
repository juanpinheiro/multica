# 20 — Fix root-path redirect + remove dead login-session machinery

**Status:** `done`
**Model:** `opus`

## Parent

`.scratch/multica-personal-fork/PRD.md`

## What to build

Issue 09 replaced the entire auth surface with loopback-trusted middleware and a singleton implicit user. The login route and its associated views were deleted, but several callers and dead supporting machinery were left behind. The most visible symptom: opening `http://localhost:3000/` on a fresh install renders a 404 ("Page not found"), because the root page still redirects to a route that no longer exists.

Behavior required after this slice:

- A fresh install visiting `/` lands on `/workspaces/new` (no workspaces yet → directly into the creation flow).
- After at least one workspace exists, visiting `/` lands on `/{lastSlug}/issues`, using the `last_workspace_slug` cookie that the workspace layout already writes.
- The web app no longer has any code path that references a login screen, a session cookie, or a "log in" gate. Every loopback request is implicitly authenticated; no client-side gating remains.

Concretely, the work bundles all of the following because they're a single conceptual change ("remove the last vestiges of the login flow"):

- Replace the root-page redirect target so it resolves to the correct post-auth destination directly, without bouncing through `/login`.
- Drop the `multica_logged_in` cookie machinery: the `auth-cookie.ts` helper, the `onLogin` / `onLogout` wiring on `WebProviders`, and the `hasSession` branches in `proxy.ts`. Without a login flow, the cookie never gets set legitimately, so every branch keyed on it is dead.
- Drop the legacy `/login` redirect inside `proxy.ts`'s `LEGACY_ROUTE_SEGMENTS` handling — bookmark recovery for the slug migration is meaningless once the destination route is gone. Decide on a sensible fallback for legacy unscoped URLs (`/issues/abc`) when no workspace cookie is present (the safe answer is `/workspaces/new`).
- Remove the orphan empty route groups `apps/web/app/(auth)/` and `apps/web/app/auth/` (subdirectories `login/`, `workspaces/`, `callback/` are now empty).
- Update the `not-found.tsx` "Back to Multica" link so it doesn't dead-end back into the same root-redirect bug.

## Acceptance criteria

- [ ] Visiting `/` on a fresh install renders `/workspaces/new` (no 404, no redirect chain that ends at a dead route)
- [ ] After creating a workspace, visiting `/` renders `/{slug}/issues`
- [ ] No code under `apps/web/` references `multica_logged_in`, `setLoggedInCookie`, `clearLoggedInCookie`, `hasSession`, or `/login` (verified by grep)
- [ ] `apps/web/features/auth/` directory removed
- [ ] `apps/web/app/(auth)/` and `apps/web/app/auth/` directories removed
- [ ] `apps/web/app/page.tsx` reads the `last_workspace_slug` cookie server-side (or delegates to a small client component) and redirects accordingly; do not reintroduce a dependency on the login page
- [ ] `WebProviders` no longer passes `onLogin` / `onLogout`; `CoreProvider`'s callbacks become optional (already are) or are removed entirely if no other caller uses them
- [ ] `pnpm typecheck`, `pnpm test`, and `pnpm lint` pass
- [ ] `make check` passes (TypeScript + Vitest + Go test)

## Blocked by

None - can start immediately

## Comments

### Key decisions

- **Root page redirects server-side using `next/headers` cookies.** `app/page.tsx` now reads `last_workspace_slug` server-side via `cookies()` from `next/headers` and redirects to either `paths.workspace(slug).issues()` or `paths.newWorkspace()`. No client component, no bouncing through `/login`. Slug is URL-encoded by `paths.workspace()` so a tampered cookie can't break out of the path.
- **Legacy URL handler in `proxy.ts` collapsed to two branches.** Without auth, the previous three-branch flow (no session → /login, session+slug → /{slug}/..., session without slug → /) collapses to a single ternary: `lastSlug ? /${lastSlug}${pathname} : /workspaces/new`. No 401-style detour exists anymore.
- **Root-path proxy redirect deleted.** The proxy's `pathname === "/" && hasSession && lastSlug` branch is gone. The root page now handles redirection itself, which is simpler and means `/` works the same way for fresh installs as for returning users.
- **`onLogin` / `onLogout` removed from `CoreProvider` and `AuthInitializer`.** The callbacks only ever existed to write the `multica_logged_in` cookie. With no login flow they have no caller, so the props were dropped from `CoreProviderProps`, `CoreProvider`, `AuthInitializer`. WebProviders no longer imports anything from `features/auth/`.
- **`not-found.tsx` left untouched.** Its `Back to Multica` link still points at `/`, but `/` is no longer a dead-end after the root-page fix — server-side redirect now resolves correctly. No edit needed.

### Files changed

**Modified:**
- `apps/web/app/page.tsx` — server-side redirect using `next/headers` cookies and `paths` helpers
- `apps/web/proxy.ts` — removed `hasSession` checks, `/login` fallback, and root-path redirect branch; legacy-segment fallback now goes to `/workspaces/new` when no cookie
- `apps/web/components/web-providers.tsx` — dropped `onLogin` / `onLogout` props and `features/auth` import
- `packages/core/platform/types.ts` — removed `onLogin` / `onLogout` from `CoreProviderProps`
- `packages/core/platform/core-provider.tsx` — dropped `onLogin` / `onLogout` parameters and forwarding to `AuthInitializer`
- `packages/core/platform/auth-initializer.tsx` — dropped `onLogin` / `onLogout` props and their call sites

**Deleted:**
- `apps/web/features/auth/auth-cookie.ts` (and the now-empty `apps/web/features/auth/` directory and `apps/web/features/` parent)
- `apps/web/app/(auth)/` (empty `login/` and `workspaces/new/` directories)
- `apps/web/app/auth/` (empty `callback/` directory)

### Verification

- `pnpm typecheck` → 4/4 packages green.
- `pnpm test` → 666 tests / 81 files pass across `@multica/core`, `@multica/ui`, `@multica/views`, `@multica/web`.
- `pnpm lint` → only pre-existing warnings (no new errors). The `qc` missing-dep warning on `auth-initializer.tsx` was pre-existing (`useEffect` already had `[]` and `qc` already in scope before this edit).
- Grep verified: no references to `multica_logged_in`, `setLoggedInCookie`, `clearLoggedInCookie`, `hasSession`, or `/login` remain under `apps/web/`.
- Server-side Go test packages relevant to my changes pass (`internal/handler`, `internal/middleware`, `internal/service`, `internal/realtime`). The other failing Go packages (`daemon/execenv`, `daemon/repocache`, `pkg/agent`, `pkg/redact`) are pre-existing Windows-environment failures documented in earlier issues' comments (missing claude/codex/hermes/kimi/opencode binaries on PATH, Windows path-length / symlink-permission issues) — unrelated to this slice.

### Blockers / notes for next iteration

None — all acceptance criteria met. Issues 21 (remove API Tokens tab) and 22 (clean stale auth strings and locales) remain independent and can proceed.
