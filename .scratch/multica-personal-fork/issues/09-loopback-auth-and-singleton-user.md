# 09 — Loopback auth + singleton user

**Status:** `done`
**Model:** `opus`

## Parent

`.scratch/multica-personal-fork/PRD.md`

## What to build

Replace the entire authentication surface with a loopback-trusted middleware and bootstrap a singleton implicit user on server startup.

**Auth model:**
- Loopback requests (`r.RemoteAddr` in `127.0.0.0/8` or `::1`) pass through with the singleton user attached to the request context. No `Authorization` header required.
- Non-loopback requests: if `MULTICA_TOKEN` is unset → 401 with the message "server is not configured for non-loopback access". If set → require `Authorization: Bearer <MULTICA_TOKEN>`; any mismatch → 401.
- Trusted-proxy `X-Forwarded-For` handling: when the immediate peer is in `RATE_LIMIT_TRUSTED_PROXIES`, the original client IP from `X-Forwarded-For` is used; if that resolves to loopback, the request is loopback.

**Singleton user:**
- On server startup, ensure a row exists in `users` with UUID `00000000-0000-0000-0000-000000000001`, email `local@multica`, name `You`. Idempotent — runs every startup, no-ops if the row exists.
- Every authenticated request is attributed to this user. `author_id` foreign keys on comments, chat sessions, audit log entries, etc. continue to work.

This is the only issue in the deletion effort that introduces new code. Test coverage is required (see below).

## Acceptance criteria

- [ ] New middleware in `internal/middleware/` that implements loopback + optional token gating per the rules above
- [ ] New bootstrap function in `internal/auth/` or `internal/service/` that ensures the singleton user exists; called from server startup in `cmd/server/main.go`
- [ ] `handler/auth.go`, `handler/auth_signup_test.go`, `handler/personal_access_token.go` deleted
- [ ] `internal/auth/PATCache`, `DaemonTokenCache`, `MembershipCache` deleted
- [ ] All `/auth/*` routes (`/auth/send-code`, `/auth/verify-code`, `/auth/google`, `/auth/logout`) and `/api/tokens/*` routes deleted from `cmd/server/router.go`
- [ ] All `/api/me/onboarding/*` routes already deleted in 08; verify
- [ ] `middleware/Auth` and `middleware/DaemonAuth` replaced by the new loopback middleware (or kept as thin wrappers that delegate to it)
- [ ] `apps/web/app/(auth)/` route group deleted (no login pages exist)
- [ ] `packages/views/auth/` deleted
- [ ] `packages/core/auth/` reduced to the bare minimum: `useCurrentUser()` returns the singleton; the auth store either disappears or holds a constant
- [ ] CLI: `multica login` and `multica login --token` updated to write `MULTICA_TOKEN` into the profile config (used when the CLI talks to a non-loopback server), or deleted if the loopback-only common case doesn't need them
- [ ] Tests for the loopback middleware cover: loopback pass-through (with and without `Authorization` header), non-loopback with `MULTICA_TOKEN` unset → 401, non-loopback with `MULTICA_TOKEN` set + no header → 401, non-loopback with `MULTICA_TOKEN` set + wrong header → 401, non-loopback with correct header → pass, trusted-proxy `X-Forwarded-For` resolving to loopback → pass, trusted-proxy `X-Forwarded-For` resolving to non-loopback → 401
- [ ] Tests for the singleton bootstrap cover: empty DB creates the singleton, DB with the singleton present is a no-op (no duplicate insert)
- [ ] `pnpm typecheck`, `pnpm test`, and `go test ./...` pass

## Blocked by

- 08-remove-onboarding
