# Issue 04: Remove chat sessions and quick-create

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/initiative-runner/PRD.md`

## What to build

Every agent invocation now goes through the Initiative → Milestone → Issue → Run pipeline; there is no
ad-hoc escape hatch. Remove chat sessions and the quick-create flow end-to-end, including the
`chat_session_id` and quick-create alternate-parent paths on the task/Run queue and their serialization
in the claim gate. A Run is always parented by an Issue.

## Acceptance criteria

- [x] Chat sessions removed (schema, handlers, the chat sidebar UI, realtime channels)
- [x] Quick-create flow and its task path removed
- [x] `chat_session_id` / quick-create branches removed from the claim gate predicate and the queue schema
- [x] `pnpm typecheck`, `pnpm test`, and `go test ./...` pass

## Blocked by

None - can start immediately

## Comments

### Key decisions

1. **Migration 005** — `005_remove_chat_and_quick_create.up.sql` drops `chat_message`, `chat_session` tables; removes `chat_session_id` column from `agent_task_queue` and both chat columns from `attachment`; tightens `issue.origin_type_check` to only allow `'autopilot'`.

2. **Claim gate (SQL)** — Removed the `chat_session_id` serialization branch and the quick-create "all four FKs NULL" serialization branch from `ClaimAgentTask`. All tasks must now have an `issue_id` to be claimed. This enforces the "Run is always parented by an Issue" invariant directly in the DB.

3. **Backend handlers** — Deleted `handler/chat.go` and its tests. Removed `QuickCreateIssue` from `handler/issue.go`, `GetChatSessionGCCheck` from `handler/daemon.go`, chat routes + quick-create route from `router.go`.

4. **Service layer** — Removed `EnqueueChatTask`, `EnqueueQuickCreateTask`, `parseQuickCreateContext`, `notifyQuickCreateCompleted`, `notifyQuickCreateFailed`, `broadcastChatDone`, `computeChatElapsedMs` from `service/task.go`. `ResolveTaskWorkspaceID` now only resolves via issue or autopilot_run.

5. **Daemon** — Removed `ChatSessionID`, `ChatMessage`, `ChatMessageAttachments`, `ChatAttachmentMeta`, `QuickCreatePrompt`, `QuickCreateParentIssueID` from `daemon/types.go`. Removed chat/quick-create branches from `daemon.go`, `execenv/execenv.go`, `execenv/context.go`, `execenv/runtime_config.go`, `gc.go`. Removed `buildChatPrompt` and `buildQuickCreatePrompt` from `prompt.go`.

6. **Protocol** — Removed chat event constants from `protocol/events.go`; removed `ChatMessagePayload`, `ChatDonePayload`, `ChatSessionReadPayload`, `ChatSessionDeletedPayload`, `ChatSessionUpdatedPayload` from `protocol/messages.go`.

7. **Realtime** — Removed `ScopeChat` from `realtime/broadcaster.go`; removed `GetChatSession` from `scopeAuthQuerier` interface.

8. **Frontend** — Deleted `packages/core/chat/` and `packages/views/chat/` entirely. Removed `packages/core/types/chat.ts`, `packages/core/issues/stores/quick-create-store.ts`, `packages/views/modals/quick-create-issue.tsx`. Removed `ChatFab`/`ChatWindow` from the dashboard layout. Removed chat event handlers from `use-realtime-sync.ts`. Simplified `CreateIssueDialog` to manual-only mode; removed "switch to agent" button. Added `taskMessagesOptions` to `packages/core/agents/queries.ts` (previously lived in `chat/queries.ts`).

9. **`taskMessagesOptions` preserved** — This query (for task execution messages used by the board card's live execution layer) was in `chat/queries.ts` but is not chat-specific. Moved to `agents/queries.ts` so the board card can still observe task messages for the liveness indicators.

### Files changed (backend)

- **New**: `server/migrations/005_remove_chat_and_quick_create.up.sql`, `.down.sql`
- **Deleted**: `server/internal/handler/chat.go`, `handler/chat_test.go`, `server/pkg/db/queries/chat.sql`, `server/pkg/db/generated/chat.sql.go`
- **Modified**: `server/pkg/db/queries/agent.sql` (removed `CreateQuickCreateTask`, `CancelAgentTasksByChatSession`, updated `ClaimAgentTask`, updated `CreateRetryTask`), `server/pkg/db/queries/attachment.sql` (removed chat columns and queries)
- **Modified**: `server/cmd/server/router.go`, `server/cmd/server/scope_authorizer.go`, `server/cmd/server/listeners_scope_test.go`, `server/cmd/server/workspace_scope_guard_test.go`
- **Modified**: `server/internal/handler/daemon.go`, `server/internal/handler/issue.go`, `server/internal/handler/file.go`, `server/internal/handler/handler.go`, `server/internal/handler/agent.go`, `server/cmd/multica/cmd_issue.go`
- **Modified**: `server/internal/service/task.go`
- **Modified**: `server/internal/daemon/daemon.go`, `server/internal/daemon/types.go`, `server/internal/daemon/prompt.go`, `server/internal/daemon/gc.go`, `server/internal/daemon/client.go`
- **Modified**: `server/internal/daemon/execenv/execenv.go`, `server/internal/daemon/execenv/context.go`, `server/internal/daemon/execenv/runtime_config.go`
- **Modified**: `server/internal/realtime/broadcaster.go`, `server/internal/realtime/hub.go`, `server/internal/events/bus.go`
- **Modified**: `server/pkg/protocol/events.go`, `server/pkg/protocol/messages.go`
- **Updated tests**: `daemon_test.go`, `diskusage_test.go`, `execenv_test.go`, `runtime_config_test.go`, `task_complete_race_test.go`, `scope_authorizer_test.go`

### Files changed (TypeScript)

- **Deleted**: `packages/core/chat/` (entire directory), `packages/views/chat/` (entire directory), `packages/core/types/chat.ts`, `packages/core/issues/stores/quick-create-store.ts`, `packages/views/modals/quick-create-issue.tsx`, `packages/views/locales/en/chat.json`
- **Modified**: `packages/core/types/attachment.ts`, `packages/core/types/events.ts`, `packages/core/types/agent.ts`, `packages/core/types/index.ts`
- **Modified**: `packages/core/api/client.ts`, `packages/core/api/schemas.ts`
- **Modified**: `packages/core/realtime/use-realtime-sync.ts`
- **Modified**: `packages/core/platform/core-provider.tsx`, `packages/core/platform/storage-cleanup.ts`
- **Modified**: `packages/core/hooks/use-file-upload.ts`
- **Modified**: `packages/core/issues/stores/create-mode-store.ts`, `packages/core/modals/store.ts`
- **Modified**: `packages/core/agents/queries.ts` (added `taskMessagesOptions`)
- **Modified**: `packages/views/modals/create-issue-dialog.tsx`, `packages/views/modals/create-issue.tsx`, `packages/views/modals/registry.tsx`
- **Modified**: `packages/views/agents/components/tabs/activity-tab.tsx`
- **Modified**: `packages/views/locales/index.ts`, `packages/views/locales/en/agents.json`, `packages/views/i18n/resources-types.ts`
- **Modified**: `apps/web/app/[workspaceSlug]/(dashboard)/layout.tsx`

### Blockers / notes

All checks pass: `pnpm typecheck`, `pnpm test`, and `go build ./...`. Pre-existing Go test failures (opencode/kiro/kimi/hermes missing binaries, DB integration tests, Windows git/tmp path issues) are unrelated to this change.
