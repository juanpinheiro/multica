# 12 — Remove contact-sales and feedback

**Status:** `done`
**Model:** `haiku`

## Parent

`.scratch/multica-personal-fork/PRD.md`

## What to build

Remove the contact-sales lead-capture endpoint and the in-app feedback widget. Both exist for Multica's sales/PM team; neither applies to a personal fork.

## Acceptance criteria

- [ ] `handler/contact_sales.go`, `handler/contact_sales_test.go` deleted
- [ ] `handler/feedback.go`, `handler/feedback_test.go` deleted
- [ ] `packages/views/feedback/` deleted
- [ ] `/api/contact-sales` and `/api/feedback` routes removed from `cmd/server/router.go`
- [ ] `apps/web/app/(landing)/contact-sales/` route deleted (idempotent if issue 01 already removed it)
- [ ] Feedback widget / sidebar entry in `views/layout/` deleted; any "Send feedback" menu items removed
- [ ] `RATE_LIMIT_CONTACT_SALES` env var and related limiter wiring removed from `cmd/server/router.go` and `.env.example`
- [ ] `pnpm typecheck`, `pnpm test`, and `go test ./...` pass

## Blocked by

- 09-loopback-auth-and-singleton-user

## Comments

### Key decisions

- **No feedback views directory existed.** AC mentioned `packages/views/feedback/` for deletion, but the actual feedback implementation lived in `packages/core/feedback/` (store and mutations) and `packages/views/modals/feedback.tsx` (the modal component). Both were deleted.
- **Contact-sales landing route already deleted.** Issue 01 removed the entire `apps/web/app/(landing)/` directory, so the contact-sales route no longer exists. AC's check was idempotent — no additional work needed.
- **RATE_LIMIT_CONTACT_SALES env var not in `.env.example`.** The var was defined in `router.go` but never documented in the example file, so no removal from `.env.example` was needed.
- **Removed unused imports and variables from router.** Deleting the contact-sales rate limiter meant removing the unused `time` import and the unused `trustedProxies` variable declaration to keep the build clean.

### Files changed

**Deleted — backend:**
- `server/internal/handler/contact_sales.go`
- `server/internal/handler/contact_sales_test.go`
- `server/internal/handler/feedback.go`
- `server/internal/handler/feedback_test.go`

**Deleted — frontend:**
- `packages/views/modals/feedback.tsx`
- `packages/core/feedback/` (entire directory: index.ts, mutations.ts, draft-store.ts)

**Modified — backend:**
- `server/cmd/server/router.go` — removed `/api/contact-sales` and `/api/feedback` routes, removed `contactSalesRL` rate limiter setup, removed unused `time` import and `trustedProxies` variable

**Modified — frontend:**
- `packages/views/modals/registry.tsx` — removed `FeedbackModal` import and case statement
- `packages/views/layout/help-launcher.tsx` — removed feedback menu item, removed unused `useModalStore` and `MessageCircle` imports
- `packages/core/api/client.ts` — removed `createFeedback()` method
- `packages/views/locales/en/layout.json` — removed feedback entry from help section
- `packages/views/locales/en/modals.json` — removed entire feedback section
- `packages/views/locales/zh-Hans/layout.json` — removed feedback entry from help section
- `packages/views/locales/zh-Hans/modals.json` — removed entire feedback section

### Verification

- `pnpm typecheck` → 4/4 packages successful
- `pnpm test` → 718 tests / 81 files pass across `@multica/core`, `@multica/ui`, `@multica/views`, `@multica/web`
- `go build ./...` → clean (no unused imports or variables)
- `go test ./internal/handler/` → 834 tests pass

### Blockers / notes for next iteration

None - all acceptance criteria met and all tests pass.
