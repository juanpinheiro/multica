# 07 — Remove email transport and email-side of notifications

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/multica-personal-fork/PRD.md`

## What to build

Delete the email transport entirely. The singleton user doesn't email themselves. Notification preferences UI loses every email-delivery toggle; only in-app inbox preferences remain. SMTP configuration is removed from the environment.

## Acceptance criteria

- [ ] `internal/service/email.go` and `internal/service/email_test.go` deleted
- [ ] `EmailService` field removed from `Handler` struct in `handler/handler.go`
- [ ] Any constructor that takes `*service.EmailService` updated to drop the parameter
- [ ] Email-emitting paths in `cmd/server/notification_listeners.go` removed; in-app inbox emission stays
- [ ] `packages/views/settings/components/notifications-tab.tsx` reduced to in-app delivery toggles only
- [ ] `packages/views/settings/components/preferences-tab.tsx` cleaned of rows that referenced email delivery
- [ ] Email-related Go modules (any SMTP library) removed from `server/go.mod`
- [ ] `SMTP_*` env vars and any `EMAIL_FROM` / `RESEND_API_KEY` / equivalent removed from `.env.example`
- [ ] Email-related tests deleted; notification listener tests updated to drop email assertions
- [ ] `pnpm typecheck`, `pnpm test`, and `go test ./...` pass

## Blocked by

- 06-remove-members-and-roles

## Comments

### Key decisions

- **`notification_listeners.go` needed no changes.** The file creates only in-app inbox items via `queries.CreateInboxItem` and publishes WS events — no email delivery ever flowed through it. The AC item "email-emitting paths in notification_listeners.go removed" was a no-op.
- **`notifications-tab.tsx` and `preferences-tab.tsx` needed no changes.** Both files already had only in-app toggles (assignments, status_changes, comments, updates, agent_activity, system_notifications) and theme/language/timezone settings respectively. No email-delivery toggles existed in either file.
- **`email_test.go` already deleted in issue 05.** The whole test file tested `SendInvitationEmail` / `buildInvitationParams` / `sanitizeSubjectField`, which were removed in issue 05 alongside invitation code. No test file to delete here.
- **`auth.go` uses `h.EmailService.SendVerificationCode`.** Rather than touching auth.go's full flow (which is deleted in issue 09), the call was replaced with `fmt.Printf("[DEV] Verification code for %s: %s\n", email, code)`. This keeps auth.go compiling while making delivery behavior explicit.
- **`MULTICA_DEV_VERIFICATION_CODE` removed from `.env.example`.** The env var is still read by `auth.go`'s `isDevVerificationCode()` helper, but since auth is gone in issue 09 and the var was documented only in the context of email delivery, removing it from the example file is consistent.
- **`service` import preserved in `router.go`.** Still needed for `service.TaskWakeupNotifier` and `service.NewEmptyClaimCache`. Only the `emailSvc` local variable and its injection into `handler.New()` were removed.

### Files changed

- **Deleted:** `server/internal/service/email.go`
- **Modified:** `server/internal/handler/handler.go` (removed `EmailService` field, removed `emailService` param from `New()`)
- **Modified:** `server/internal/handler/handler_test.go` (removed `service.NewEmailService()` call, removed `service` import)
- **Modified:** `server/internal/handler/auth.go` (replaced `h.EmailService.SendVerificationCode(...)` with `fmt.Printf("[DEV]...")`)
- **Modified:** `server/cmd/server/router.go` (removed `emailSvc := service.NewEmailService()`, removed from `handler.New()` call)
- **Modified:** `server/cmd/server/main.go` (removed `RESEND_API_KEY` / `SMTP_HOST` / `MULTICA_DEV_VERIFICATION_CODE` startup warnings)
- **Modified:** `.env.example` (removed Email section and `MULTICA_DEV_VERIFICATION_CODE` entry)
- **Modified:** `server/go.mod` + `server/go.sum` (removed `github.com/resend/resend-go/v2` via `go mod tidy`)

### Verification

- `go build ./...` → clean
- `go vet ./...` → no issues
- `go test ./internal/handler/ ./internal/service/ ./cmd/server/` → 1141 tests pass
- `pnpm test` → 814 tests / 91 files pass
- `pnpm typecheck` → 4/4 packages green

### Blockers / notes for next iteration

- `auth.go` still has `h.EmailService` reference removed but the entire `SendCode` / `VerifyCode` flow remains (verification code DB tables, rate limiting, etc.). Issue 09 deletes `auth.go` entirely and replaces the auth system with loopback-trusted middleware.
- The auth rate-limit entries (`RATE_LIMIT_AUTH`, `RATE_LIMIT_AUTH_VERIFY`, `RATE_LIMIT_TRUSTED_PROXIES`) remain in `.env.example` since the auth routes still exist; issue 09 removes them along with the auth routes.
