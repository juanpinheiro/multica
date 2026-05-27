# PRD: Multica Personal Fork

**Status:** `ready-for-agent`
**Owner:** Juan Pinheiro
**Created:** 2026-05-26

## Problem Statement

The user forked Multica to use as a personal AI agent management platform — somewhere to assign work to local Claude Code / Codex / Gemini / Cursor Agent / etc. agents, watch them run via the local daemon, and track progress in a dashboard.

The current codebase is built as a multi-tenant SaaS product for 2-10 person teams (this is the explicit framing in `CLAUDE.md`'s "Project Context" section). As a result, it carries a lot of weight that has nothing to do with the user's intended workflow:

- **Three alternate frontends** beyond the web app (`apps/desktop`, `apps/mobile`, `apps/docs`) that the user will never use, but whose existence drives ~40% of the architectural rules in `CLAUDE.md` (zero `next/*` imports in shared packages, `NavigationAdapter`, `WindowOverlay`, `DragStrip`, tab-store, the entire `packages/views/platform/` indirection layer).
- **Marketing surface** inside `apps/web` (`(landing)/{about,changelog,homepage,usecases,download,contact-sales}`) that requires `fumadocs-mdx`, which is incompatible with Next 16's Turbopack and forces the dev/build scripts to fall back to `--webpack`. This single fact makes dev rebuilds 3-5× slower than they should be and is documented as a known issue in `apps/web/next.config.ts:74-77`. The user already deleted `apps/docs/` contents (except `docs/agents/`) but the dependency on fumadocs still bleeds into `apps/web`.
- **Multi-user authentication machinery** (magic-link email + Google OAuth + Personal Access Tokens + multiple Redis-backed token caches + membership cache + workspace roles + invitations + email service) that exists to support team workflows the user does not have.
- **SaaS-only operational infrastructure** (`internal/cloudruntime/` proxies a remote Fleet service that only exists on Multica.ai; `internal/analytics/posthog.go` ships product telemetry to Multica's own PostHog; `handler/contact_sales*` and `handler/feedback*` are lead-capture for Multica's sales team; `auth/CloudFrontSigner` signs cookies for Multica's CloudFront distribution; `realtimeMetricsHandler` exposes ops scrape endpoints meant for Multica's SRE).
- **Release pipeline for a public product** (`.goreleaser.yml`, `release.yml` GitHub Action, Homebrew tap publish, install scripts at `scripts/install.{sh,ps1}`, `SELF_HOSTING*.md` guides, `docker-compose.selfhost*.yml`, Helm chart in `deploy/helm/`, `Dockerfile.web` separate from `Dockerfile`) — all assume someone other than the author is installing Multica.
- **Chinese i18n** (`packages/views/locales/zh-Hans/`, `README.zh-CN.md`, `conventions.zh.mdx`, "Chinese voice guide" rules in `CLAUDE.md`) that the user does not read or write.
- **Multi-node infrastructure** (Redis-backed `UpdateStore`, `ModelListStore`, `LocalSkillListStore`, `LocalSkillImportStore`, `LivenessStore`, `WebhookRateLimiter`, multi-node WebSocket pub/sub) that exists to let Multica.ai scale horizontally. The user runs a single node.
- **Onboarding wizard** (`packages/views/onboarding/` + `handler/onboarding*.go` + `handler/onboarding_shim.go` for pre-v3 desktop clients) that walks new SaaS signups through agent setup. The user is reconstructing their own workflow and does not need a guided tour.
- **Playwright e2e tests** that assume the full multi-user login → invite → assign workflow. Most will break or become irrelevant after the multi-user removal.

The combined effect: the dev server is sluggish, the architecture rules are bloated, the schema has ~80 historical migrations covering features that are being removed, and the mental overhead of working in the codebase is dominated by infrastructure the user will never touch.

## Solution

Strip Multica down to a personal-use fork that preserves every actual product feature (dashboard surface) and removes everything else. The result is a web + Go server + Postgres + local CLI daemon stack with:

- **No alternate platforms.** Desktop and mobile deleted entirely. Web is the only frontend. The architectural restrictions that existed to support cross-platform sharing (`zero next/* in packages/views`, `NavigationAdapter`, `WindowOverlay`, etc.) lift, simplifying the package structure.
- **No marketing surface.** Landing routes deleted, `apps/docs/` (except `docs/agents/`, which the user already preserved) deleted, fumadocs-mdx dependency removed from `apps/web`. This unblocks Next 16 Turbopack and gives back the 3-5× dev speed.
- **No real authentication.** The server binds to `127.0.0.1` by default and treats loopback requests as authenticated. If `MULTICA_TOKEN` is set, non-loopback requests must present it via `Authorization: Bearer <token>`. A singleton implicit user (UUID `00000000-0000-0000-0000-000000000001`) owns all comments, chat sessions, and other rows that previously had `author_id`.
- **No SaaS-specific infrastructure.** Cloud Runtime fleet, PostHog analytics, contact sales, feedback, CloudFront signer, realtime metrics endpoint, multi-node Redis stores, webhook rate limiter (kept for autopilots), all deleted. Redis as a runtime dependency is removed.
- **No release pipeline.** No GoReleaser, no Homebrew tap, no release GitHub Action, no install scripts, no self-hosting docs.
- **Single locale.** Only English. `zh-Hans` translations, Chinese voice guide, and the Chinese conventions doc are deleted.
- **Fresh DB.** All historical migrations are consolidated into a single `001_init.sql` representing the post-cut schema. No migration scripts for the deletion — the user wipes the DB and re-bootstraps.
- **All product features preserved.** Issues + board + comments + timeline + reactions + labels + projects + attachments + mentions + subscribers + batch ops, chat sessions, agents + templates + env + skills, runtimes + daemon + heartbeat + update + local-skills + model-list, squads, autopilots (cron + webhook + delivery log + replay), inbox, pins, GitHub integration (webhook + PR sync + repos + integrations tab), dashboard of token/cost/runtime usage, multi-workspace switcher, the full `multica` CLI (`issue`, `squad`, `autopilot`, `agent`, `runtime`, `project`, `skill`, `workspace`, etc.).

## User Stories

1. As a solo developer running a personal agent platform, I want the web dashboard to start in under 5 seconds and HMR in under 200ms, so that iteration on the codebase doesn't feel sluggish.
2. As a solo developer, I want to open the local web app and land directly in my workspace without a login screen, so that the auth ceremony doesn't get in the way of every session.
3. As a solo developer, I want the option to set `MULTICA_TOKEN` and bind to a non-loopback address, so that I can safely access my instance from another device on the same Tailnet without rewriting the auth layer.
4. As a solo developer, I want to start the agent daemon with `multica daemon start` from my terminal, so that I don't need a desktop wrapper to manage the daemon process.
5. As a solo developer, I want to create multiple workspaces (e.g. `personal`, `work`, `oss`) and switch between them, so that I can keep distinct agent rosters, runtimes, and GitHub integrations per context.
6. As a solo developer, I want to assign an issue to an agent and have the daemon execute it locally with `claude` / `codex` / `gemini` / etc., so that I can offload work the same way the original Multica supports.
7. As a solo developer, I want a chat sidebar where I can ad-hoc invoke an agent without creating an issue first, so that lightweight "do this for me" requests don't require ticket overhead.
8. As a solo developer, I want squads (groups of agents with a leader that delegates), so that I can build a routing layer like `@FrontendTeam` even in a solo context where the layer represents specialization rather than team structure.
9. As a solo developer, I want autopilots (cron + webhook triggers that create issues automatically), so that I can schedule recurring agent work (daily standups, weekly audits) and react to external webhooks.
10. As a solo developer, I want the dashboard to show token usage, cost, run-time, and task counts aggregated by agent/project/day, so that I can monitor what my agents are spending.
11. As a solo developer, I want pins (sidebar shortcuts to issues, agents, or projects), so that frequently-used items stay accessible.
12. As a solo developer, I want the inbox notification feed (mentions, issue updates, task completions), so that I can catch up on what happened while I was away without checking every page.
13. As a solo developer, I want the full GitHub integration (webhook for PR sync, repos tab, integrations tab in settings, PR status surfaced on issues), so that I can eventually use the fork to list PRs and connect agent work to real GitHub activity.
14. As a solo developer, I want the skills system (reusable instruction sets that agents can attach), so that I can compound agent capabilities over time.
15. As a solo developer, I want comments and timeline on issues (so agents can report progress), reactions, mentions (so I can `@agent` to trigger work), and subscribers, so that the issue-driven workflow remains as expressive as the original.
16. As a solo developer, I do NOT want email notifications, so that I don't have to configure SMTP for a tool I use alone — in-app inbox is enough.
17. As a solo developer, I do NOT want a login screen, signup form, magic-link email, Google OAuth, or personal access token UI, so that there's no auth ceremony for a tool that runs on my own machine.
18. As a solo developer, I do NOT want a desktop app, a mobile app, or a docs site, so that the monorepo only contains the surface I actually use.
19. As a solo developer, I do NOT want telemetry being sent to PostHog, so that my usage data doesn't leave my machine.
20. As a solo developer, I do NOT want Chinese translations, since I work in English and Portuguese — and the existing Portuguese translation doesn't exist anyway.
21. As a solo developer, I do NOT want the Cloud Runtime fleet proxy, since I only use the local daemon.
22. As a solo developer, I do NOT want contact-sales or feedback endpoints, since there's no sales team and no PM to receive feedback.
23. As a solo developer, I do NOT want a release pipeline (GoReleaser, Homebrew tap, install scripts, self-hosting docs), since I build and run the code from my own checkout.
24. As a solo developer, I do NOT want an onboarding wizard, since I'm rebuilding my workflow and know exactly what to set up.
25. As a solo developer, I do NOT want Playwright e2e tests, since the flows they cover (multi-user login, invitations) are being deleted, and I can verify behavior by using the app.
26. As a solo developer, I want to wipe the database and start fresh, so that the schema reflects only the features I'm keeping, with no orphan tables from cut features.
27. As a solo developer, I want migrations consolidated into a single `001_init.sql`, so that the schema is auto-documented in one file instead of spread across 80+ historical migrations.
28. As a solo developer, I want the project name "multica" preserved everywhere (binary, env vars, packages, DB name, module path), so that there is no rename refactor and I can still cherry-pick bug fixes from upstream if I want.
29. As a solo developer, I want the CI workflow (`.github/workflows/ci.yml`) simplified to Go test + TypeScript typecheck + lint + Vitest (no Playwright, no desktop, no mobile), so that I get fast feedback on PRs without paying for irrelevant build steps.
30. As a solo developer, I want the `vitest` and `go test` suites to keep covering everything they cover today (minus tests of deleted features), so that I have a safety net while I'm doing massive refactors.
31. As a solo developer, I want a new auth middleware that allows all loopback requests and gates non-loopback requests behind `MULTICA_TOKEN` when set, so that local use is frictionless and remote exposure is opt-in.
32. As a solo developer, I want the backend to bootstrap a singleton implicit user on first start, so that the existing schema (with `author_id` foreign keys) continues to work without my having to log in.
33. As a solo developer, I want the dashboard's usage metrics (tokens, cost, run-time, task count) to continue working after analytics is cut, because those metrics come from the `task_usage` Postgres table reported by the daemon — they have no dependency on PostHog.
34. As a solo developer, I want the notification preferences settings tab to keep only the in-app toggles (drop everything email-related), so that the UI doesn't expose configuration for a delivery channel that doesn't exist anymore.
35. As a solo developer, I want `README.md` rewritten as a short personal-fork orientation (architecture overview, how to run, how to add features), so that the repo's front page reflects what the project actually is.
36. As a solo developer, I want the `CLAUDE.md` cleaned of any rule that exists only to support deleted platforms (desktop / mobile rules, NavigationAdapter, WindowOverlay, sharing principles between web/desktop, mobile React-version policy), so that the agent instructions reflect the actual codebase.
37. As a solo developer, I want the naming conventions from `apps/docs/content/docs/developers/conventions.mdx` inlined into `CLAUDE.md` (the parts about code/file/DB-column naming), so that the convention reference survives the deletion of `apps/docs/`.
38. As a solo developer, I want `reserved_slugs.json` reduced to only the technical slugs that actually exist as routes (`api`, `auth`, `ws`, `uploads`, `health`, `healthz`, `readyz`, plus the workspace-level reserved routes used by `apps/web`), so that the list isn't dominated by marketing slugs that don't apply.

## Implementation Decisions

### Scope shape

- **Web-only frontend.** `apps/desktop/` and `apps/mobile/` deleted in full. Web (`apps/web/`) is the only client. The architectural rules in `CLAUDE.md` that exist to support cross-platform sharing (zero `next/*` imports in `packages/views`, the `NavigationAdapter` indirection, the desktop-specific route categories, the `WindowOverlay` / `DragStrip` / tab-store / `WorkspaceRouteLayout` machinery, the mobile React-version policy) become obsolete and are removed.
- **`packages/views/` may import `next/*` directly** after desktop is gone, eliminating a major source of indirection. Existing platform abstractions in `packages/views/platform/` and `packages/core/platform/` are reduced to whatever Next still legitimately needs.
- **No alternate package consumers.** Shared packages (`@multica/core`, `@multica/ui`, `@multica/views`) only need to compile under one bundler (Next.js / Turbopack). The "Internal Packages pattern" stays — packages still export raw `.ts`/`.tsx`.

### Authentication model

- **Loopback-trusted auth.** The auth middleware allows any request whose `r.RemoteAddr` resolves to the loopback range (`127.0.0.0/8` or `::1`). For non-loopback requests, if `MULTICA_TOKEN` is set in the environment, the middleware requires `Authorization: Bearer <MULTICA_TOKEN>`. If `MULTICA_TOKEN` is unset and the request is non-loopback, the middleware returns 401 — exposing the server without setting a token is a configuration error.
- **Singleton implicit user.** On server startup, the backend ensures a user row exists with UUID `00000000-0000-0000-0000-000000000001`, email `local@multica`, name `You`. Every authenticated request is attributed to this user. Comments, chat sessions, audit log entries, etc. continue to use `author_id` foreign keys against this row.
- **Daemon authentication unchanged in principle.** The daemon still registers with its `MULTICA_DAEMON_ID` (UUID) to identify which runtime row it corresponds to. Daemon → server traffic over loopback is automatically trusted; if the user runs the daemon on a remote machine pointing at a remote server, they set the same `MULTICA_TOKEN` on both sides.
- **Deletions:** `handler/auth.go`, `handler/personal_access_token.go`, `handler/auth_signup_test.go`, the magic-link verification code tables, Google OAuth handler, `auth.PATCache`, `auth.DaemonTokenCache`, `auth.MembershipCache`, `middleware/DaemonAuth` (collapses into the new middleware), `internal/auth/CloudFrontSigner`, the `RefreshCloudFrontCookies` middleware.
- **What stays:** `middleware/RequireWorkspaceMember` and friends become no-ops (single user is a member of every workspace by definition) but the API of `r.Use(...)` calls in `cmd/server/router.go` stays the same to minimize churn.

### Multi-user removal

- All routes, handlers, services, and schema related to multi-user team workflows are deleted: invitations (`handler/invitation.go`, `handler/invitation_test.go`, `views/invitations/`, `views/invite/`, the `invitations` API surface in `cmd/server/router.go`), members (`handler/workspace.go` member endpoints stripped, `handler/workspace_revoke.go`, `views/members/`, `views/settings/members-tab.tsx`), roles (the `owner` / `admin` / `member` enum and all `RequireWorkspaceRoleFromURL` guards), email transport (`internal/service/email.go`, related notification listeners in `cmd/server/notification_listeners.go` that send email), onboarding (`handler/onboarding.go`, `handler/onboarding_shim.go`, `views/onboarding/`, `core/onboarding/`, cloud-waitlist endpoints).
- **Workspace creation is preserved** because multi-workspace stays (per Decision 11). Without other users, "create workspace" creates a workspace owned by the singleton implicit user. The workspace switcher in the sidebar remains.
- **Notification preferences settings tab** keeps only the in-app delivery toggles. Email delivery toggles are removed.

### SaaS-only infrastructure removal

- `internal/cloudruntime/` deleted in full. `handler/cloud_runtime.go`, `handler/cloud_runtime_test.go`, `views/runtimes/components/cloud-runtime-dialog.tsx`, `views/runtimes/components/custom-pricing-dialog.tsx`, `core/runtimes/cloud-runtime.ts`, `core/runtimes/custom-pricing-store.ts` deleted. The `/api/cloud-runtime/*` route subtree removed from `cmd/server/router.go`. The `CloudRuntimeFleetURL` / `CloudRuntimeFleetTimeout` config fields removed from `handler.Config`.
- `internal/analytics/` deleted in full. All `h.analytics.Capture(...)` call sites in handlers replaced with no-ops (or removed where the call was the only meaningful work in the function). `core/analytics/` deleted. PostHog env vars (`POSTHOG_API_KEY`, `POSTHOG_HOST`, `POSTHOG_ENVIRONMENT`) removed from `.env.example` and `turbo.json` globalEnv. The `analytics.Client` interface and all its implementations (PostHog, no-op) deleted; handler constructor signature simplified.
- `handler/contact_sales.go`, `handler/contact_sales_test.go`, `handler/feedback.go`, `handler/feedback_test.go`, `views/feedback/` deleted. The `/api/contact-sales` and `/api/feedback` routes removed.
- `internal/auth/CloudFrontSigner` deleted. The `/api/me` response no longer includes signed cookie URLs. Local file serving via `storage.LocalStorage` is the only attachment delivery path; S3 direct GET is still supported but without CloudFront signing.
- `realtimeMetricsHandler` and the `/health/realtime` route deleted. `REALTIME_METRICS_TOKEN` env var removed.
- The Redis-backed `UpdateStore`, `ModelListStore`, `LocalSkillListStore`, `LocalSkillImportStore`, `LivenessStore` deleted; only the in-memory implementations remain. `WebhookRateLimiter` and `WebhookIPRateLimiter` Redis variants deleted; the in-memory variants remain (autopilots stay, so webhook rate limiting stays). The `rdb` parameter to `NewRouter` is removed entirely. Redis as a dependency is removed from `docker-compose.yml`, `.env.example`, the Makefile, and CI.
- The realtime relay's multi-node pub/sub mode is removed; the `realtime.Hub` keeps its single-node in-process delivery only.

### Platform / marketing surface removal

- `apps/desktop/` deleted in full.
- `apps/mobile/` deleted in full. The `apps/mobile/CLAUDE.md`, the mobile-specific rules and tech-stack baseline, and the mobile-related scripts in the root `package.json` (`dev:mobile*`, `ios:mobile*`) are deleted. Mobile-specific deps in `pnpm-workspace.yaml` catalog (Expo, React Native) are removed. The mobile-locked `react@19.2.0` / `react-native@0.83.6` direct dependencies in the root `package.json` are removed.
- `apps/docs/` already deleted by the user except `docs/agents/`. The remaining fumadocs config (`apps/web/source.config.ts`, `apps/web/.source/`, the `(landing)/*` routes, the `createMDX` wrapper in `next.config.ts`, the `fumadocs-mdx` / `fumadocs-core` dependencies in `apps/web/package.json`, the `postinstall: fumadocs-mdx` script, the `fumadocs-mdx &&` prefix on the `typecheck` and `build` scripts, the `--webpack` flag on `dev` and `build`, the `/docs` rewrite in `next.config.ts`, the `DOCS_URL` env var in `turbo.json` and `.env.example`) all deleted. With fumadocs gone, Next 16 Turbopack becomes available on `dev` and `build`.
- `apps/web/app/(landing)/{about,changelog,homepage,usecases,download,contact-sales}` deleted.

### Release pipeline removal

- `.goreleaser.yml` deleted.
- `.github/workflows/release.yml` deleted.
- `.github/workflows/desktop-smoke.yml` deleted.
- `scripts/install.sh` and `scripts/install.ps1` deleted.
- `SELF_HOSTING.md`, `SELF_HOSTING_ADVANCED.md`, `SELF_HOSTING_AI.md` deleted.
- `docker-compose.selfhost.yml` and `docker-compose.selfhost.build.yml` deleted.
- `deploy/helm/` deleted in full.
- `Dockerfile.web` deleted (only `Dockerfile` for the server remains).
- The "CLI Release" section of `CLAUDE.md` deleted.

### i18n simplification

- `packages/views/locales/zh-Hans/` deleted (24 JSON namespace files).
- `packages/views/locales/parity.test.ts` simplified to verify only that the EN bundle is internally consistent (no second locale to compare against).
- `packages/views/locales/index.ts` reduced to only the `en` entry in `RESOURCES`.
- `packages/views/locales/glossary.md` deleted.
- `README.zh-CN.md` deleted.
- `apps/docs/content/docs/developers/conventions.zh.mdx` was already deleted with `apps/docs/`. The naming convention portion of `conventions.mdx` is inlined into `CLAUDE.md` before `apps/docs/` is fully removed; the Chinese voice guide is dropped entirely. The "Conventions reference" pointer at the top of `CLAUDE.md` is replaced with the inlined naming conventions.
- The `SupportedLocale` type in `@multica/core/i18n` keeps `'en'` as the only union member. The i18n machinery (`useT()`, namespaces, `i18next` config) is kept intact for forward compatibility — only the `zh-Hans` resources and references are removed.

### Schema consolidation

- All migrations under `server/migrations/` are deleted.
- A single new `001_init.sql` is written that reflects the post-cut schema: every kept feature's tables, columns, indexes, and constraints, with no traces of deleted features.
- Tables removed: `invitation`, `personal_access_token`, `verification_code`, `verification_code_attempts`, `workspace_member` (replaced by implicit single-user model — workspaces still need an owner FK pointing at the singleton user), all email queue / delivery tables if any exist, autopilot deletion is NOT happening (autopilots stay).
- Columns removed: `users.onboarding_completed_at`, `users.onboarding_data`, `users.signup_source`, any `created_by` columns where the value will always be the singleton user (kept as denormalization where it's already there to minimize churn — assess case by case).
- Extensions: only `pgcrypto` required. `pg_bigm` and `pg_cron` (used by autopilots scheduling) — keep `pg_cron` because autopilots stay, drop `pg_bigm` because CJK search is gone with i18n simplification. If `pg_cron` is unavailable in dev environments, autopilot scheduling falls back to in-process Go ticker (mirroring the existing `service/autopilot_scheduler.go` fallback).
- Docker compose Postgres image stays `pgvector/pgvector:pg17` for now — switching to vanilla `postgres:17` is possible since pgvector isn't used, but it's not blocking. Track as a follow-up.

### CI / build

- `.github/workflows/ci.yml` simplified to: Go test, TypeScript typecheck (via Turborepo), ESLint, Vitest (via Turborepo). No Playwright, no desktop build, no mobile build. Postgres service container kept for Go integration tests.
- `.github/workflows/desktop-smoke.yml` deleted.
- `.github/workflows/release.yml` deleted.
- `turbo.json` cleaned: remove `DOCS_URL`, `DESKTOP_RENDERER_PORT` from `globalEnv`. Add `inputs` declarations to `typecheck`, `test`, `lint` tasks so cache invalidation is tighter. Consider adding `globalDependencies: ["pnpm-lock.yaml", "tsconfig.json"]`.
- The webpack → Turbopack switch (dropping `--webpack` from `apps/web/package.json` dev/build scripts) lands in the web-build-cleanup issue (the first issue derived from this PRD), giving the user fast feedback before the larger cuts begin.

### Test deletions

- `e2e/` directory deleted in full.
- `playwright.config.ts` deleted.
- `@playwright/test` removed from root `package.json` devDependencies.
- Unit tests (Vitest + Go test) for deleted features (invitations, members, magic-link auth, Google OAuth, PATs, onboarding, cloud runtime, analytics, contact sales, feedback, email service, CloudFront signer, desktop, mobile, zh-Hans parity) are deleted alongside their implementations.

### Documentation cleanup

- `README.md` rewritten to a short personal-fork README: what this is, how to run, where to look (`CLAUDE.md` for everything detailed), no marketing language, no badges, no screenshots.
- `docs/assets/` (marketing banners, screenshots, logos) deleted. Any logo asset still needed by the web app login/header is moved to `apps/web/public/`.
- `CLAUDE.md` significantly trimmed: delete the "Mobile-specific Rules" section, delete the "Desktop-specific Rules" section, delete the "Sharing Principles" section (only one share zone now), delete the "Cross-Platform Development Rules" section, delete the "CSS Architecture" section's reference to desktop, delete the "Apps/desktop" line from the architecture list, delete the "CLI Release" section, delete the "Multi-tenancy" section's multi-user phrasing, simplify the "Project Context" section to reflect single-user reality. Insert the naming portion of `conventions.mdx` near the top as a replacement for the "Conventions reference" pointer at lines 5-18.
- `CONTRIBUTING.md` and `AGENTS.md` reviewed for stale references and trimmed similarly.
- `reserved_slugs.json` reduced to only the technical slugs that exist as routes today: `api`, `auth`, `ws`, `uploads`, `health`, `healthz`, `readyz`, plus the workspace-level slugs that exist under `apps/web/app/[workspaceSlug]/(dashboard)/`: `agents`, `autopilots`, `inbox`, `issues`, `members`, `my-issues`, `projects`, `runtimes`, `settings`, `skills`, `squads`, `usage`. The `members` reservation stays even though the members tab is removed, because the slug should still be reserved from being chosen as a workspace name. The `packages/core/paths/reserved-slugs.ts` regeneration via `pnpm generate:reserved-slugs` is run after the JSON edit.

### Naming and module path

- "multica" name preserved throughout: binary name, env var prefixes (`MULTICA_*`), package names (`@multica/core`, `@multica/ui`, `@multica/views`, `@multica/web`), DB name (`POSTGRES_DB=multica`), Go module path (`github.com/multica-ai/multica/...`), realtime channel/event prefixes (`multica:chat:*`).
- Issue identifier prefix (the `MUL-123` format) derives from the workspace slug; that mechanism is unchanged.

### Deployment shape

- Stays as today: Postgres + Go server + Next.js Node process + local CLI daemon, orchestrated via Makefile. No Redis. No GoReleaser. No production Docker images published; the user builds from their checkout. `docker-compose.yml` reduced to just the Postgres service.

### Fresh start

- No data migration. The user wipes the DB before running the consolidated `001_init.sql`. No backwards-compatibility shims. No "migration down" scripts for deleted tables (the deletions live in code only; the DB they're wiping never had them).

### Out-of-scope items (explicitly listed in the "Out of Scope" section below)

- Rebrand / rename.
- Switching the frontend bundler away from Next.js.
- Switching DB to SQLite or embedding Postgres.
- Adding Portuguese translations.
- Adding new features.
- Reintroducing tests that didn't exist before.

## Testing Decisions

### What makes a good test in this project

- Test external behavior, not internal structure. For backend handlers, that means: hit the HTTP endpoint, assert on response status / body / DB state — don't reach into the handler's private helpers.
- For frontend, that means: render the component, simulate user interaction, assert on rendered output — don't assert on internal hook state or unexposed callbacks.
- Tests for kept features continue using the patterns already established in the repo (Vitest + `@testing-library/react` for frontend, standard `go test` + `testdata/` fixtures + DB-backed setup for backend handlers).

### Modules that get new tests

The only **new code** in this entire effort is the auth simplification. Tests should cover:

- **Loopback auth middleware.** Given the middleware that allows loopback and gates non-loopback behind `MULTICA_TOKEN`, the test cases are:
  - Loopback request (`r.RemoteAddr` is `127.0.0.1:...` or `[::1]:...`) → request passes through with the singleton user attached to context, regardless of `Authorization` header presence.
  - Non-loopback request with `MULTICA_TOKEN` unset → 401 with a clear error message ("server is not configured for non-loopback access").
  - Non-loopback request with `MULTICA_TOKEN` set, no `Authorization` header → 401.
  - Non-loopback request with `MULTICA_TOKEN` set, wrong token in `Authorization` → 401.
  - Non-loopback request with `MULTICA_TOKEN` set, correct `Authorization: Bearer <token>` → request passes through with singleton user attached.
  - Edge case: `X-Forwarded-For` from a trusted proxy. The trusted-proxy logic already exists in `middleware.ParseTrustedProxies` — if the proxy is trusted and the original `X-Forwarded-For` is loopback, the request is loopback. Otherwise it's non-loopback.

- **Singleton user bootstrap.** Given the function that ensures the singleton user exists, the test cases are:
  - DB with no `users` row → after bootstrap, the singleton user exists with UUID `00000000-0000-0000-0000-000000000001`.
  - DB with the singleton user already present → bootstrap is a no-op, no DB mutation.
  - DB with a different user (rare, but possible if the user hand-edits) → singleton user is still created idempotently.

### Modules that lose tests (alongside their code)

All the existing tests for deleted features (auth, invitations, members, onboarding, contact sales, feedback, cloud runtime, analytics, email, CloudFront signer, etc.) are deleted with the code. No effort is made to "salvage" tests — if the feature is gone, the test is gone.

### Modules whose tests stay unchanged

All tests for kept features (issues, chat, agents, runtimes, squads, autopilots, projects, labels, skills, inbox, comments, GitHub, dashboard usage, pins, workspace, the CLI commands) stay. These tests are the safety net during the deletion effort — they catch unintended regressions where a deletion accidentally breaks a kept feature.

### What is NOT tested

- The deletions themselves don't need tests — "delete this file" doesn't have observable behavior to assert.
- The `CLAUDE.md` / `README.md` rewrites don't need tests.
- The migration consolidation is tested implicitly by every Go test that creates a test DB; if the consolidated `001_init.sql` is wrong, every backend test fails.
- The schema-level cleanups don't get explicit migration tests — fresh start means there's only one starting point.

### Prior art

- The existing handler tests in `server/internal/handler/*_test.go` show the established pattern: spin up a test DB, register the handler, fire `httptest.NewRequest`, assert. The new auth middleware tests follow the same shape, using `httptest.ResponseRecorder` and varying `r.RemoteAddr`.
- The existing Vitest tests in `packages/views/auth/login-page.test.tsx` and similar are deleted (login is gone), but the patterns in `packages/views/issues/*.test.tsx` are unchanged.

## Out of Scope

- **Rebranding or renaming.** The project keeps the name "multica" everywhere (binary, env vars, packages, DB, module path). Renaming is a separate effort with no technical benefit for a personal fork.
- **Bundler / framework changes.** Next.js stays as the web framework. There is no migration to Vite + React Router, no `output: 'export'` static export, no rewrite to a SPA. The dev-speed problem is fixed by removing fumadocs and re-enabling Turbopack, not by changing the bundler.
- **Database changes beyond consolidation.** Postgres stays. No switch to SQLite, no embedded Postgres via `embedded-postgres-binaries-go`, no migration to a single Go binary with embedded DB. pgvector image is kept for now (vanilla `postgres:17` is a possible follow-up but doesn't block).
- **Adding Portuguese translations.** Only English is kept. PT is not added; that's a feature, not a cut.
- **Adding new product features.** The fork preserves every existing dashboard feature exactly as-is. New features (custom dashboard widgets, new chart types, new agent providers, new integrations) are explicit non-goals of this effort.
- **Reintroducing test coverage that didn't exist before.** Tests for kept features stay as they are; tests for cut features are deleted; new tests are only written for the new auth middleware and singleton bootstrap.
- **Exposing the server to the public internet.** The fork is designed for local use. The `MULTICA_TOKEN` mechanism supports trusted-network exposure (Tailscale, LAN), but nothing in this PRD addresses public-internet hardening (HTTPS termination, request validation beyond what exists, CSRF tokens, etc.). Those would be follow-ups if the user ever wanted public exposure.
- **Backwards compatibility with prior Multica versions.** Fresh DB. No migration scripts. No support for connecting an old desktop client to the new server. If the user has any state in a prior Multica install they want to preserve, they handle it manually before running the new code.
- **Anything in `docs/agents/`.** Those three files (`issue-tracker.md`, `triage-labels.md`, `domain.md`) are part of the user's working environment for this effort and are intentionally preserved.

## Further Notes

### Order of execution (recommended)

The deletions interact in ways that make some orderings dangerous. Recommended sequence:

1. **Web build cleanup first.** Remove fumadocs, drop `--webpack`, kill `(landing)/*` routes. This is the only change that gives the user immediate, perceptible speed-up, and it's fully self-contained (the rest of the codebase doesn't depend on fumadocs). Doing it first means every subsequent change benefits from the faster dev loop.
2. **Delete alternate platforms** (`apps/desktop`, `apps/mobile`, remainder of `apps/docs`). These are independent of the backend cuts and can land in any order relative to each other.
3. **Remove multi-user** (invitations, members, roles, email, onboarding, notification email side). This must precede the auth simplification because the auth simplification assumes there's only one user.
4. **Loopback auth + singleton user.** Replace the entire auth surface (magic-link, Google, PAT, caches, etc.) with the loopback middleware + singleton bootstrap.
5. **Remove SaaS infrastructure** (cloud runtime, analytics, contact sales, feedback, CloudFront, realtime metrics, Redis stores). Order within this group is flexible — they're independent.
6. **Remove release pipeline + self-hosting docs.** Pure deletions, no code dependencies.
7. **i18n simplification.** Drop `zh-Hans`. Mostly independent but worth doing before the migration consolidation so the consolidated init doesn't carry CJK index definitions.
8. **Migration consolidation.** Once all schema changes are decided, write the single `001_init.sql`. This is the last code change because everything before it might have schema implications.
9. **Tests + CI cleanup.** Delete e2e, simplify `ci.yml`. Can land any time after the corresponding feature deletions.
10. **Docs rewrite.** README, `CLAUDE.md` trimming, `reserved_slugs.json` reduction, inlining of conventions. Last because it reflects the post-cut state.

### Risk and verification

The largest risks during this effort:

- **Accidentally breaking a kept feature when deleting a related one.** Example: cutting `handler/invitation.go` while accidentally leaving a `ListInvitations` call in a kept handler will surface as a typecheck or test failure — both of which are part of CI. The unit + integration test coverage on kept features is the primary defense.
- **Schema drift between the consolidated `001_init.sql` and what the Go code expects.** Mitigation: after writing the consolidated migration, run the full Go test suite — every handler test exercises real SQL against the schema, so missing columns or constraints surface immediately.
- **Frontend bundle breaking on a stray `@multica/views/...` import that points at a deleted file.** Mitigation: TypeScript typecheck via Turborepo catches every dangling import.
- **CLAUDE.md drifting from reality.** Mitigation: after each batch of deletions, re-read the relevant `CLAUDE.md` section and prune. This is part of the docs-rewrite issue.

### Dependencies on user environment

The result assumes the user has these on their machine:

- A working `claude` (or other supported agent CLI: `codex`, `copilot`, `openclaw`, `opencode`, `hermes`, `gemini`, `pi`, `cursor-agent`, `kimi`, `kiro-cli`) on `PATH`, authenticated independently (via that CLI's own auth flow — Multica does not handle Anthropic / OpenAI auth).
- Postgres reachable via `DATABASE_URL` (Docker container per `docker-compose.yml`, or a separately-managed Postgres).
- Go 1.26+ and Node 22+ for building from source.

The Multica fork does not bundle the agent CLIs, does not bundle Postgres, does not bundle Node — it expects the user to manage those dependencies as they already do for upstream Multica.

### Future follow-ups (explicitly not in this PRD)

- Switching from `pgvector/pgvector:pg17` to `postgres:17` (vanilla) since no pgvector features are used.
- Pruning the pnpm catalog of versions that were only used by deleted apps (mobile-specific Expo / React Native versions, desktop-specific Electron versions).
- Removing `electron`, `expo`, `react-native` from root `package.json` dependencies and `onlyBuiltDependencies`.
- Adding a one-command bootstrap (`make personal-init`) that creates the workspace + first agent automatically after a fresh DB, since the onboarding wizard is gone.
- Optional: switching from in-process Go scheduler to `pg_cron` for autopilots in production, or vice versa, depending on which is less operationally annoying.
- Optional: Tailscale Funnel docs / examples for the user, if they want remote access later.

## Comments

