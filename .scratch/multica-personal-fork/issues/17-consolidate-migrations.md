# 17 — Consolidate migrations into single 001_init.sql

**Status:** `ready-for-agent`
**Model:** `opus`

## Parent

`.scratch/multica-personal-fork/PRD.md`

## What to build

Replace the 80+ historical migrations under `server/migrations/` with a single `001_init.sql` that reflects the final post-cut schema. Every table, column, index, constraint, and extension required by the kept features lands in one file; everything else is dropped.

This is a destructive change: existing dev databases must be wiped and re-bootstrapped. This is consistent with the PRD decision to start fresh.

This issue is the highest-numbered blocking one because every preceding feature-deletion issue may carry schema implications.

## Acceptance criteria

### Migration cleanup

- [ ] All files under `server/migrations/` deleted
- [ ] New `server/migrations/001_init.sql` written, containing the entire post-cut schema

### Schema content — KEPT tables

The new `001_init.sql` must include schemas for: `users` (with the singleton seed insert), `workspaces`, `agents`, `agent_runtimes`, `agent_skills`, `agent_templates`, `runtime_usage`, `task`, `task_message`, `task_usage`, `issue`, `issue_comment`, `issue_timeline`, `issue_label`, `issue_reaction`, `issue_subscriber`, `issue_metadata`, `issue_pull_request_link`, `project`, `project_resource`, `label`, `chat_session`, `chat_message`, `squad`, `squad_member`, `squad_briefing`, `autopilot`, `autopilot_run`, `autopilot_trigger`, `autopilot_delivery`, `autopilot_webhook_rate_limit`, `inbox_item`, `pin`, `notification_preferences` (in-app columns only), `skill`, `skill_file`, `local_skill`, `github_installation`, `github_repo`, `attachment`, `workspace_repo`, `activity_log`.

(Source the canonical column / index / constraint definitions from the existing migrations being collapsed. Verify against `pkg/db/queries/` to make sure every SQL query in the codebase resolves.)

### Schema content — REMOVED tables

The new `001_init.sql` must NOT include: `invitation`, `personal_access_token`, `verification_code`, `verification_code_attempts`, `workspace_member` (single-user implicit model), any email-queue / delivery tables, any cloud-runtime / fleet tables.

### Schema content — REMOVED columns

The new `001_init.sql` must NOT include: `users.onboarding_completed_at`, `users.onboarding_data`, `users.signup_source`, any `created_by` columns where the value is always the singleton user (case-by-case — keep `created_by` on rows where the value carries information beyond identity, such as `agent_id` on tasks).

### Extensions

- [ ] `pgcrypto` required (for `gen_random_uuid()` and similar)
- [ ] `pg_cron` retained (autopilots scheduling)
- [ ] `pg_bigm` NOT included (CJK bigram search is gone with i18n simplification)

### Verification

- [ ] Migration applies cleanly on an empty DB via `make migrate-up`
- [ ] `make db-reset` works end-to-end on a fresh Postgres container
- [ ] `go test ./...` passes — every backend test creates its own DB and runs the migration; if a column the code expects is missing, this surfaces immediately
- [ ] `make sqlc` regenerates without changes (the schema the queries target hasn't drifted from the schema being applied)

## Blocked by

- 08-remove-onboarding
- 09-loopback-auth-and-singleton-user
- 10-remove-cloud-runtime-fleet
- 11-remove-analytics
- 12-remove-contact-sales-and-feedback
- 13-remove-cloudfront-realtime-metrics-and-redis
- 16-drop-i18n-zh-hans
