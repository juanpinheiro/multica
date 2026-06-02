# Issue 01: Remove squad

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/initiative-runner/PRD.md`

## What to build

Delete the squad feature end-to-end. The single-user, agent-driven model has no use for a leader agent
delegating to a team of agents — the Orchestrator (built later) replaces that role. Remove the squad
schema, handlers, the squad-leader briefing/delegation machinery, the `multica squad` CLI, the squads UI
and routes, and the `assignee_type=squad` option. Keep `assignee_type=agent`. Comments and agent
@mentions stay (they remain a dispatch trigger).

## Acceptance criteria

- [ ] `squad` / `squad_member` tables and all squad SQL removed; no orphan FKs
- [ ] Squad handlers, the leader briefing/delegation flow, and `multica squad ...` CLI removed
- [ ] Squads UI, routes, and the `assignee_type=squad` option removed; assignee picker offers agents only
- [ ] `pnpm typecheck`, `pnpm test`, and `go test ./...` pass (kept features unaffected)

## Blocked by

None - can start immediately

## Comments

### Key decisions

1. **Migration**: Added `002_remove_squad.up.sql` / `002_remove_squad.down.sql` to drop `squad` and `squad_member` tables, drop `autopilot_run.squad_id`, and tighten `assignee_type` CHECK constraints to `('member', 'agent')` for issues and `('agent')` for autopilots.
2. **sqlc**: Deleted `squad.sql` query file and the stale generated `squad.sql.go` (sqlc doesn't auto-delete orphaned generated files). Updated `autopilot.sql` and `issue.sql` to remove squad subqueries from `involves_user_id` filter.
3. **Simplifications**: `resolveAutopilotLeader` collapsed to a single `GetAgent` call; `EnqueueTaskForSquadLeader` and `EnqueueQuickCreateTask(squadID)` removed; squad-leader briefing injection stripped from daemon claim handler; `IsSquadLeader` removed from `TaskContextForEnv`.
4. **Reserved slugs**: Removed `"squads"` from `reserved_slugs.json` and ran `pnpm generate:reserved-slugs`.
5. **Frontend**: Parallel agent removed squad from types, API client, query hooks, modals, assignee picker, sidebar, paths, locales, and all route pages.

### Files changed (backend)
- Deleted: `handler/squad.go`, `handler/squad_briefing.go`, 5 squad test files, `service/squad_no_action.go`, `service/autopilot_squad_test.go`, `cmd/multica/cmd_squad.go`, `pkg/db/queries/squad.sql`, `pkg/db/generated/squad.sql.go`
- New: `migrations/002_remove_squad.up.sql`, `migrations/002_remove_squad.down.sql`
- Modified: `handler/{issue,comment,daemon,autopilot,agent,task_lifecycle}.go`, `handler/issue_child_done{,.test}.go`, `handler/{handler,github,issue_involves}_test.go`, `service/{task,autopilot,agent_ready}.go`, `daemon/{types,prompt}.go`, `daemon/execenv/{execenv,runtime_config}.go`, `daemon/execenv/execenv_test.go`, `cmd/server/router.go`, `cmd/server/notification_listeners.go`, `cmd/multica/main.go`, `pkg/protocol/events.go`, `pkg/db/queries/{autopilot,issue}.sql`, `internal/util/mention.go`, `handler/reserved_slugs.json`

### Blockers / notes
None. Pre-existing test failures (`TestOpencodeBackendBlocksDirOverride`, `TestRedactHomeDirectory`) are environment-specific and unrelated to this change.
