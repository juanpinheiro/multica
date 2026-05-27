# 08 — Remove onboarding wizard

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/multica-personal-fork/PRD.md`

## What to build

Delete the onboarding wizard and all its backend support. The user is reconstructing their workflow and doesn't need a guided setup — first launch shows an empty dashboard, the user creates a workspace and first agent via normal UI flows.

This is the last multi-user-related issue. After it lands, the codebase has no concept of "new user signup" or "guided setup" — every kept feature (issues, chat, agents, etc.) is reachable without any onboarding step.

## Acceptance criteria

- [x] `handler/onboarding.go`, `handler/onboarding_shim.go`, `handler/onboarding_test.go` deleted
- [x] `packages/views/onboarding/` deleted in full
- [x] `packages/core/onboarding/` deleted in full
- [x] `apps/web/app/(auth)/onboarding/` route deleted *(AC had wrong path `(landing)/onboarding/` — actual path was `(auth)/onboarding/`)*
- [x] `/api/me/onboarding/*` routes removed from `cmd/server/router.go`: `PatchOnboarding`, `CompleteOnboarding`, `JoinCloudWaitlist`, `BootstrapOnboardingRuntime`, `BootstrapOnboardingNoRuntime`
- [x] References to `users.onboarded_at`, `users.onboarding_questionnaire`, `users.starter_content_state` removed from handler/service code (column drops happen in issue 17; generated DB code left intact)
- [x] References to onboarding in `CLAUDE.md` removed *(no references existed)*
- [x] Frontend: `useHasOnboarded` hook deleted, `paths.onboarding()` path removed, onboarding gate in workspace layout removed, `WelcomeAfterOnboarding` component deleted
- [x] `pnpm typecheck`, `pnpm test`, and `go test ./...` pass

## Blocked by

- 07-remove-email-and-notification-email-side

## Comments

### Key decisions

- **`resolvePostAuthDestination` simplified.** The function previously took `(workspaces, hasOnboarded)` and routed unonboarded users to `/onboarding`. Without onboarding, the logic is simply: first workspace → `/<slug>/issues`; no workspaces → `/workspaces/new`. The signature now takes only `workspaces`. All four call sites updated: `use-dashboard-guard.ts`, `login/page.tsx`, `auth/callback/page.tsx`, `workspace-tab.tsx`, and `use-realtime-sync.ts`.
- **`useHasOnboarded` deleted entirely.** It was a store selector on `user.onboarded_at`. Removed from `packages/core/paths/resolve.ts` and all callers.
- **`paths.onboarding()` deleted.** Removed the path helper and `/onboarding` from `GLOBAL_PREFIXES`. The slug `"onboarding"` is retained in `reserved_slugs.json` per the existing "historical, kept reserved post-removal" note.
- **Workspace layout hard gate removed.** The `useEffect` that redirected to `/onboarding` when `user.onboarded_at == null` is gone. The layout still redirects unauthenticated users to `/login`.
- **`WelcomeAfterOnboarding` deleted.** The post-onboarding welcome modal (which fired the Helper agent setup and starter issues) is gone. `apps/web/app/[workspaceSlug]/layout.tsx` no longer imports or renders it. `web-providers.tsx` no longer imports `useWelcomeStore` or calls `reset()` on logout.
- **`UserResponse` simplified.** Removed `OnboardedAt`, `OnboardingQuestionnaire`, `StarterContentState` fields from the `UserResponse` struct and `userToResponse` function in `auth.go`. The DB columns remain until issue 17.
- **Analytics events removed.** `EventOnboardingStarted`, `EventOnboardingQuestionnaireSubmit`, `EventOnboardingCompleted`, `EventCloudWaitlistJoined` constants and `OnboardingQuestionnaireSubmitted()`, `OnboardingCompleted()`, `CloudWaitlistJoined()` functions deleted from `server/internal/analytics/events.go`. `SourceOnboarding` and `OnboardingPath*` constants deleted.
- **`TestCreateWorkspace_DoesNotMarkOnboarded` deleted.** This test guarded the invariant that `CreateWorkspace` must leave `onboarded_at = NULL` so the workspace layout gate could redirect to `/onboarding`. With the gate and the gate's contract both gone, the test is meaningless.
- **`onboarding` locale namespace removed.** `packages/views/locales/en/onboarding.json` and `zh-Hans/onboarding.json` deleted. Removed from `locales/index.ts` resource bundle and from `i18n/resources-types.ts` type augmentation.
- **Stale comments cleaned up.** Removed onboarding-centric comments in `agent.go`, `analytics/events.go`, `analytics/index.ts`, `types/agent.ts`, and workspace layout.
- **AC path correction.** The onboarding route lived at `apps/web/app/(auth)/onboarding/` (auth route group), not `(landing)/onboarding/` as stated in the AC. The actual deletion was correct.

### Files changed

**Deleted — backend:**
- `server/internal/handler/onboarding.go`
- `server/internal/handler/onboarding_shim.go`
- `server/internal/handler/onboarding_test.go`

**Deleted — frontend:**
- `packages/core/onboarding/` (entire directory: store, types, step-order, welcome-store, recommend-template, tests)
- `packages/views/onboarding/` (entire directory: flow, steps, components, templates)
- `apps/web/app/(auth)/onboarding/page.tsx`
- `packages/views/workspace/welcome-after-onboarding.tsx`
- `packages/views/workspace/welcome-after-onboarding.test.tsx`
- `packages/views/locales/en/onboarding.json`
- `packages/views/locales/zh-Hans/onboarding.json`

**Modified — backend:**
- `server/cmd/server/router.go` — removed 5 onboarding routes
- `server/internal/handler/auth.go` — removed onboarding fields from `UserResponse` and `userToResponse`
- `server/internal/handler/workspace.go` — removed stale onboarding comment
- `server/internal/handler/workspace_test.go` — deleted `TestCreateWorkspace_DoesNotMarkOnboarded`
- `server/internal/handler/agent.go` — updated stale comment
- `server/internal/analytics/events.go` — removed onboarding event constants and functions

**Modified — frontend:**
- `packages/core/paths/resolve.ts` — simplified `resolvePostAuthDestination` to drop `hasOnboarded` arg
- `packages/core/paths/resolve.test.ts` — updated tests for new signature
- `packages/core/paths/paths.ts` — removed `paths.onboarding()` and `/onboarding` from `GLOBAL_PREFIXES`
- `packages/core/paths/index.ts` — removed `useHasOnboarded` export
- `packages/core/package.json` — removed `./onboarding` export
- `packages/core/api/client.ts` — removed `markOnboardingComplete`, `joinCloudWaitlist`, `patchOnboarding` methods, removed `OnboardingCompletionPath` import
- `packages/core/api/schemas.ts` — removed `onboarded_at`, `onboarding_questionnaire`, `starter_content_state` from `UserSchema` and `EMPTY_USER`
- `packages/core/types/workspace.ts` — removed `onboarded_at`, `onboarding_questionnaire`, `starter_content_state` from `User` interface
- `packages/core/types/agent.ts` — updated stale comment
- `packages/core/realtime/use-realtime-sync.ts` — removed `useHasOnboarded` import and `hasOnboardedRef`, updated `resolvePostAuthDestination` call
- `packages/core/realtime/use-realtime-sync-ws-instance.test.tsx` — removed `useHasOnboarded` from mock
- `packages/core/analytics/index.ts` — cleaned up stale onboarding comment
- `packages/views/i18n/resources-types.ts` — removed `onboarding` namespace from `I18nResources`
- `packages/views/locales/index.ts` — removed onboarding imports and entries from `RESOURCES`
- `packages/views/layout/use-dashboard-guard.ts` — removed `useHasOnboarded` and `resolvePostAuthDestination` with `hasOnboarded` arg
- `packages/views/package.json` — removed `./onboarding` and `./workspace/welcome-after-onboarding` exports
- `packages/views/settings/components/workspace-tab.tsx` — removed `useHasOnboarded`, updated `resolvePostAuthDestination` call
- `packages/views/settings/components/workspace-tab.test.tsx` — removed `useHasOnboarded` from mock
- `apps/web/app/[workspaceSlug]/layout.tsx` — removed `WelcomeAfterOnboarding` import/render, removed onboarding gate
- `apps/web/app/(auth)/login/page.tsx` — removed `useHasOnboarded`, simplified auth redirect logic
- `apps/web/app/auth/callback/page.tsx` — removed `hasOnboarded` logic, simplified `resolvePostAuthDestination` call
- `apps/web/app/auth/callback/page.test.tsx` — rewrote tests for post-onboarding behavior (no onboarding path)
- `apps/web/components/web-providers.tsx` — removed `useWelcomeStore` import and `reset()` call
- `apps/web/test/helpers.tsx` — removed `onboarded_at`, `onboarding_questionnaire`, `starter_content_state` from `mockUser`

### Blockers / notes for next iteration

- **DB columns stay until issue 17.** `users.onboarded_at`, `users.onboarding_questionnaire`, `users.cloud_waitlist_email`, `users.cloud_waitlist_reason`, `users.starter_content_state` still exist in the DB schema and in `server/pkg/db/generated/models.go` and `user.sql.go`. The generated code still has `MarkUserOnboarded`, `PatchUserOnboarding`, `JoinCloudWaitlist`, `SetStarterContentState` queries but they are now unreachable — no handler calls them. These will be dropped in issue 17's migration consolidation, at which point `make sqlc` will regenerate the DB layer without these columns.
- Issue 09 (loopback auth + singleton user) will delete `auth.go` entirely and replace it with a loopback middleware. The simplified `userToResponse` and `UserResponse` in this issue will be replaced again there.
- The `apps/web/app/layout.tsx` comment "Editorial serif used for onboarding headlines" (line 39) is stale but harmless — the font variable itself is still consumed by the layout. Left for issue 19's docs rewrite.
