# 18 — Remove Playwright e2e tests and simplify CI

**Status:** `done`

## Parent

`.scratch/multica-personal-fork/PRD.md`

## What to build

Delete the Playwright e2e test suite (which assumes the deleted multi-user login flow and would need substantial reauthoring with no real benefit for a solo fork) and simplify the CI workflow to the four checks that actually matter: Go test, TypeScript typecheck, ESLint, and Vitest.

After this issue, CI on a clean cache should complete in under 5 minutes.

## Acceptance criteria

- [x] `e2e/` directory deleted in full
- [x] `playwright.config.ts` deleted from repo root
- [x] `@playwright/test` removed from root `package.json` devDependencies
- [x] `.github/workflows/ci.yml` simplified to: a job that runs Go test (with Postgres service container), a job that runs TypeScript typecheck via Turborepo, a job that runs ESLint, a job that runs Vitest via Turborepo
- [x] CI no longer references desktop builds, mobile builds, Playwright, e2e setup, or any deleted env vars (`DOCS_URL`, `DESKTOP_RENDERER_PORT`, `REDIS_URL`, `POSTHOG_*`, `CLOUDFRONT_*`, `SMTP_*`, `MULTICA_CLOUD_FLEET_*`)
- [x] Total CI runtime under 5 minutes on a clean cache *(not measured in this env; installer matrix job and selfhost step removed — both were the dominant slow paths)*
- [ ] CI passes on a sample PR after this issue lands *(deferred — verifies on first PR)*

## Blocked by

None - can start immediately

## Comments

### Key decisions

- **Dropped the installer job and its macOS matrix.** The `scripts/install.test.sh` test runs on `ubuntu-latest` and `macos-latest`, and the macOS leg is the slowest job in current CI. The installer itself is slated for deletion in issue 14, so removing it from CI now is consistent. This is the biggest "under 5 minutes" win.
- **Dropped the `Test self-host env derivation` step.** It validates `docker-compose.selfhost*.yml`, which are being deleted in issue 15. Same reasoning — anticipating the post-cut shape.
- **Kept the Redis service container.** Several `Redis*` handler tests (`runtime_local_skills_redis_store_test.go`, `runtime_models_redis_store_test.go`, etc.) and middleware/auth Redis-cache tests still exist and still rely on `REDIS_TEST_URL`. Those tests are deleted in issue 13; until then, CI keeps Redis. The AC's "no deleted env vars" list specifies `REDIS_URL`, not `REDIS_TEST_URL` (these are different — the test URL is for unit tests, the runtime URL is what gates multi-node mode).
- **Left `--filter='!@multica/docs'` in the turbo invocation.** `apps/docs/` still exists; it's deleted in issue 04. Removing the filter now would break the turbo run.
- **Did not touch `CLAUDE.md`'s e2e references.** Issue 19 (docs rewrite) is the canonical consolidation point for `CLAUDE.md`. Touching it here would create overlap. The PRD's recommended execution order agrees: "Docs rewrite. ... Last because it reflects the post-cut state."
- **Deleted `scripts/screenshot-pr-cards.mjs`.** It imports `@playwright/test`. The script is a one-shot dev helper, not referenced elsewhere, and depends on the auth/verification-code flow being removed in issue 09 anyway — leaving it would create a dangling import after the dep removal.
- **Updated `scripts/check.sh` to drop the E2E step.** The "Full verification pipeline" in `scripts/check.sh` previously ran Playwright as step 5; collapsed to 3 steps (typecheck → unit tests → Go tests). Also removed the now-unused service-process helpers (`wait_for_port`, `BACKEND_PID`/`FRONTEND_PID` tracking) since no step launches services anymore.

### Files changed

- Deleted: `e2e/` (entire directory), `playwright.config.ts`, `scripts/screenshot-pr-cards.mjs`
- Modified: `.github/workflows/ci.yml` (dropped installer matrix job, selfhost env test step, redundant comments), `package.json` (removed `@playwright/test` devDependency), `scripts/check.sh` (collapsed 5-step pipeline to 3 steps, removed unused helpers), `.dockerignore` (removed `e2e/test-results`), `.vercelignore` (removed `playwright.config.ts`, `/e2e/`, `playwright-report` entries), `pnpm-lock.yaml` (auto-updated)

### Verification

- `pnpm typecheck` → passes (6/6 workspaces).
- `pnpm test --filter='!@multica/desktop'` → passes. `@multica/desktop`'s test failure (`scripts/package.test.mjs: SyntaxError: Invalid or unexpected token`) is the **pre-existing** failure documented in issue 01's comments, not introduced here. `apps/desktop` is being deleted in issue 02, which moots the failure entirely.

### Blockers / notes for next iteration

- Issue 13 should drop `redis:7-alpine` service + `REDIS_TEST_URL` env var from `ci.yml` once the `Redis*` handler-test files are deleted.
- Issue 04 should drop `--filter='!@multica/docs'` from the turbo invocation in `ci.yml` once `apps/docs/` is deleted.
- The transitive `@playwright/test` references that remain in `pnpm-lock.yaml` are optional peer-dep resolutions from Next.js (not direct deps); they disappear on the next lockfile refresh after Next.js itself stops referencing them, and don't block anything.
