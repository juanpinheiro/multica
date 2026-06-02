# Issue 08: Subtract SaaS login / identity chrome

**Status:** `ready-for-agent`
**Model:** `opus`

## Parent

`.scratch/standalone-install/PRD.md`

## What to build

Remove the multi-user SaaS login/identity machinery that is dead weight in the personal/standalone build (ADR-0008 consequence). The personal build already trusts loopback connections as the singleton user (`middleware.LoopbackAuth` + `service.BootstrapSingletonUser`), so the email-code (Resend), Google OAuth, and JWT login-page paths never run — they are pure surface area to delete.

Subtract the dead login/identity chrome (email login codes / Resend integration, Google OAuth flow, JWT login pages and their wiring) while preserving: loopback singleton auth on the server, and the existing `MULTICA_TOKEN` bearer path for non-loopback access (untouched). This touches auth and request boundaries — proceed carefully and lean on tests.

## Acceptance criteria

- [ ] The personal build presents no login wall: a loopback request is served as the singleton user with no auth step.
- [ ] Email-code/Resend, Google OAuth, and JWT login-page code paths are removed (not just disabled) from the build.
- [ ] Loopback singleton auth still works, and the `MULTICA_TOKEN` non-loopback bearer path is unchanged.
- [ ] No dead references remain (routes, handlers, env vars, config) to the removed login machinery; the server and web build clean without them.
- [ ] Existing auth/middleware tests (e.g. loopback + token paths) still pass; removed-path tests are deleted with their code.
- [ ] `make check` passes.

## Blocked by

- None - can start immediately
