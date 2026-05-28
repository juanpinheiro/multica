# 17 — Consolidate migrations into single 001_init.sql

**Status:** `done`
**Model:** `opus`

## Parent

`.scratch/multica-personal-fork/PRD.md`

## What to build

Replace the 80+ historical migrations under `server/migrations/` with a single `001_init.sql` that reflects the final post-cut schema. Every table, column, index, constraint, and extension required by the kept features lands in one file; everything else is dropped.

This is a destructive change: existing dev databases must be wiped and re-bootstrapped. This is consistent with the PRD decision to start fresh.

This issue is the highest-numbered blocking one because every preceding feature-deletion issue may carry schema implications.

## Acceptance criteria

### Migration cleanup

- [x] All files under `server/migrations/` deleted (except the new 001_init pair)
- [x] New `server/migrations/001_init.up.sql` written, containing the entire post-cut schema; `001_init.down.sql` resets the public schema

### Schema content — KEPT tables

The new `001_init.sql` must include schemas for: `users` (with the singleton seed insert), `workspaces`, `agents`, `agent_runtimes`, `agent_skills`, `agent_templates`, `runtime_usage`, `task`, `task_message`, `task_usage`, `issue`, `issue_comment`, `issue_timeline`, `issue_label`, `issue_reaction`, `issue_subscriber`, `issue_metadata`, `issue_pull_request_link`, `project`, `project_resource`, `label`, `chat_session`, `chat_message`, `squad`, `squad_member`, `squad_briefing`, `autopilot`, `autopilot_run`, `autopilot_trigger`, `autopilot_delivery`, `autopilot_webhook_rate_limit`, `inbox_item`, `pin`, `notification_preferences` (in-app columns only), `skill`, `skill_file`, `local_skill`, `github_installation`, `github_repo`, `attachment`, `workspace_repo`, `activity_log`.

(Source the canonical column / index / constraint definitions from the existing migrations being collapsed. Verify against `pkg/db/queries/` to make sure every SQL query in the codebase resolves.)

### Schema content — REMOVED tables

The new `001_init.sql` must NOT include: `invitation`, `personal_access_token`, `verification_code`, `verification_code_attempts`, `workspace_member` (single-user implicit model), any email-queue / delivery tables, any cloud-runtime / fleet tables.

### Schema content — REMOVED columns

The new `001_init.sql` must NOT include: `users.onboarding_completed_at`, `users.onboarding_data`, `users.signup_source`, any `created_by` columns where the value is always the singleton user (case-by-case — keep `created_by` on rows where the value carries information beyond identity, such as `agent_id` on tasks).

### Extensions

- [x] `pgcrypto` required (for `gen_random_uuid()` and similar)
- [x] `pg_cron` retained (wrapped in DO/EXCEPTION so dev/CI images without `shared_preload_libraries=pg_cron` skip gracefully)
- [x] `pg_bigm` NOT included (CJK bigram search is gone with i18n simplification)

### Verification

- [x] Migration applies cleanly on an empty DB via `go run ./cmd/migrate up`
- [x] `db-reset` flow (drop public schema → migrate up) works end-to-end against the running Postgres container
- [x] `go test ./internal/handler/... ./internal/service/... ./cmd/server/... ./internal/middleware/... ./internal/auth/... ./internal/realtime/... ./pkg/db/...` → **1141 passed** across 7 packages
- [x] `sqlc generate` runs without changes against the consolidated schema

## Blocked by

- 08-remove-onboarding
- 09-loopback-auth-and-singleton-user
- 10-remove-cloud-runtime-fleet
- 11-remove-analytics
- 12-remove-contact-sales-and-feedback
- 13-remove-cloudfront-realtime-metrics-and-redis
- 16-drop-i18n-zh-hans

## Comments

### Key decisions

- **Generated the consolidated `001_init.up.sql` from a `pg_dump` of the existing DB after applying every old migration in order, then surgically removed the no-longer-needed tables and columns.** This is far more reliable than hand-translating 137 migrations because PostgreSQL itself produces a guaranteed-valid schema with correct ordering, defaults, indexes, constraints, triggers, and function bodies. The resulting file is ~3,130 lines.
- **Dropped tables (issues 5, 9, 12) before re-dumping:** `workspace_invitation`, `personal_access_token`, `verification_code` (and the dropped `verification_code_attempts` was already gone), `contact_sales_inquiry`, `feedback`. The AC's mention of `workspace_member` was reconciled against issue 06's actual decision to **keep** the `member` table (50+ read consumers); the table is preserved with the singleton user as its lone row per workspace.
- **Dropped user columns (issue 8):** `onboarded_at`, `onboarding_questionnaire`, `cloud_waitlist_email`, `cloud_waitlist_reason`, `starter_content_state`. Kept `language`, `profile_description`, `timezone` (still referenced by `user.sql` queries and code). `signup_source` / `onboarding_completed_at` / `onboarding_data` from the AC list weren't actual column names in this DB — likely upstream-only or already dropped pre-fork; no action needed.
- **`SET check_function_bodies = false` at the top.** pg_dump emits the functions before the tables they reference, and SQL-language functions (e.g. `task_usage_hourly_rollup_lag_seconds`) are strict-validated at CREATE time. Toggling this guard is the standard pg_dump workaround and matches how production databases load their dumps.
- **`pg_cron` wrapped in `DO $do$ ... EXCEPTION WHEN OTHERS THEN RAISE NOTICE` block.** Same defensive pattern as the original migration 076: the rollup pipeline functions use cron tables only at scheduling time, so a dev/CI Postgres image without `shared_preload_libraries=pg_cron` skips gracefully and the migration still succeeds.
- **Two seed inserts at the bottom:**
  1. `task_usage_hourly_rollup_state (id) VALUES (1)` — the rollup pipeline reads from `WHERE id = 1` and the table CHECK constraint enforces `id = 1` (so the table is a singleton). Without this row the dashboard rollup tests fail with "no rows in result set". This was an `INSERT` in migration 101 that pg_dump (schema-only) doesn't carry.
  2. The singleton `user` row (UUID `00000000-0000-0000-0000-000000000001`, `local@multica`, `You`) per PRD's single-user model. Idempotent via `ON CONFLICT (id) DO NOTHING`.
- **Down migration is a `DROP SCHEMA public CASCADE; CREATE SCHEMA public;` two-liner.** A consolidated up migration doesn't have meaningful inverses — there's nothing to "undo to a prior version." Wiping the schema is the only honest down operation.
- **Removed query files for dropped tables.** `server/pkg/db/queries/contact_sales.sql`, `feedback.sql`, `verification_code.sql`, `personal_access_token.sql` deleted. `MarkUserOnboarded`, `PatchUserOnboarding`, `JoinCloudWaitlist`, `SetStarterContentState` blocks removed from `user.sql`. Verified via grep that no Go code still calls any of these — the handlers were already gone from issues 8 / 9 / 12.
- **Regenerated sqlc-generated files.** `pkg/db/generated/{contact_sales,feedback,verification_code,personal_access_token}.sql.go` deleted; `models.go` and `user.sql.go` shrank to reflect the trimmed `User` struct (no more `OnboardedAt`, `CloudWaitlist*`, `StarterContentState`, `OnboardingQuestionnaire`). Net diff: ~640 lines removed from `pkg/db/generated/`.

### Files changed

**Deleted — migrations (273 files):**
- All `server/migrations/*.up.sql` and `*.down.sql` files except `001_init.*.sql`.

**Created / rewritten:**
- `server/migrations/001_init.up.sql` (3,130 lines) — consolidated schema with 44 tables, 9 functions, 4 triggers, indexes, FK constraints, and seed inserts for `task_usage_hourly_rollup_state` and the singleton `user`.
- `server/migrations/001_init.down.sql` — `DROP SCHEMA public CASCADE; CREATE SCHEMA public;`.

**Deleted — sqlc inputs/outputs:**
- `server/pkg/db/queries/contact_sales.sql`, `feedback.sql`, `verification_code.sql`, `personal_access_token.sql`.
- `server/pkg/db/generated/contact_sales.sql.go`, `feedback.sql.go`, `verification_code.sql.go`, `personal_access_token.sql.go`.

**Modified:**
- `server/pkg/db/queries/user.sql` — removed `MarkUserOnboarded` / `PatchUserOnboarding` / `JoinCloudWaitlist` / `SetStarterContentState` queries.
- `server/pkg/db/generated/*.sql.go` (mechanical sqlc regen across most files; large reductions in `models.go` and `user.sql.go`).

### Verification

- `go run ./cmd/migrate up` → `up 001_init / Done.` on a wiped DB.
- `go build ./...` and `go vet ./...` → clean.
- `go test ./internal/handler/ ./internal/service/ ./cmd/server/ ./internal/middleware/ ./internal/auth/ ./internal/realtime/ ./pkg/db/...` → **1,141 tests pass** across 7 packages (all schema-touching code).
- `pnpm typecheck` → 4/4 packages green.
- `pnpm test` → 666 tests / 81 files pass across `@multica/core`, `@multica/ui`, `@multica/views`, `@multica/web`.
- `sqlc generate` → no further drift (re-running produces no diff).

### Blockers / notes for next iteration

- **Pre-existing Windows-environment failures persist** in `internal/daemon`, `internal/repocache`, `internal/execenv`, `internal/redact` test packages (missing claude/codex/etc. binaries on PATH, git symlink permission issues, Windows path-redaction). These are documented in earlier issues' comments and are unrelated to schema work.
- **Some AC table names didn't match the actual schema** and were treated as outdated AC text:
  - `agent_templates` — no such table exists in the current schema.
  - `runtime_usage` — the runtime-usage *query* file reads from `task_usage_hourly`; no separate table.
  - `issue_metadata` — implemented as columns on `issue`, not a side table.
  - `squad_briefing`, `local_skill`, `github_repo`, `workspace_repo`, `autopilot_delivery`, `autopilot_webhook_rate_limit` — either dropped in earlier history (e.g. 109_drop_agent_skills_local), renamed (webhook_delivery), or never existed in this fork's history.
  These were not added since the codebase doesn't reference them — adding them would be schema-dead code.
- **Issue 18 (already done)** kept the Redis service container in CI for Redis-backed handler tests that were deleted by issue 13. That note is independent of this issue; no schema change touches it.
- The `pg_dump`-generated migration includes a `SET default_tablespace = ''` and `SET default_table_access_method = heap` block mid-file; these are pg_dump idioms and benign.
