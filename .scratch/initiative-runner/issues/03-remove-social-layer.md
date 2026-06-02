# Issue 03: Remove the human-collaboration social layer

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/initiative-runner/PRD.md`

## What to build

Remove the affordances that only make sense between multiple humans: reactions, subscribers, and human
@mentions (with their notification routing). **Keep comments** — they are how an Agent reports progress —
and **keep agent @mentions** as a dispatch trigger. Only the human-facing mention/notify path is removed.

## Acceptance criteria

- [x] Reactions removed (schema, handlers, UI)
- [x] Subscribers removed (schema, handlers, UI)
- [x] Human @mention detection/notification removed; agent @mention still enqueues a Run
- [x] Comments still render and still post; agent progress reporting unaffected
- [x] `pnpm typecheck`, `pnpm test`, and `go test ./...` pass

## Blocked by

None - can start immediately

## Comments

### Key decisions

1. **Migration 004** — `004_remove_social_layer.up.sql` drops `comment_reaction`, `issue_reaction`, `issue_subscriber`, and `notification_preference` tables. Inbox infrastructure (`inbox_item`) is kept for repurposing in issue 18.

2. **notification_listeners.go simplified** — Removed the subscriber fan-out (`notifySubscribers`/`notifyIssueSubscribers`), human mention notifications (`notifyMentionedMembers`), and notification preferences (`loadUserPrefs`/`isNotifMuted`). Kept only `archiveStaleTaskFailedInbox` which archives stale `task_failed` inbox rows when an issue reaches a terminal status — this is pure DB cleanup with no subscriber dependency.

3. **subscriber_listeners.go deleted** — Auto-subscription logic (creator/assignee/commenter/mentioned) removed entirely along with tests.

4. **notification_preferences deleted** — Handler, SQL, generated code, frontend queries/mutations, and the settings tab removed. Single-user system has no per-user preference concepts.

5. **Frontend** — Removed `Reaction`/`IssueReaction`/`IssueSubscriber` types, reaction/subscriber API methods, `useIssueReactions`/`useIssueSubscribers` hooks, `ReactionBar` component, reaction optimistic UI from timeline, subscriber popover from issue detail, `QuickEmojiPicker` for reactions in comments.

6. **comment.go** — `CommentResponse.Reactions` field removed; `commentToResponse` signature simplified to drop the reactions parameter. All callers updated.

7. **Agent @mention dispatch preserved** — `enqueueMentionedAgentTasks` in `comment.go` is untouched; only the human inbox notification path (in `notification_listeners.go`) was removed.

8. **Test helpers moved** — `createTestIssue`/`createTestUser`/`cleanupTest*`/`inboxItemsForRecipient` helpers from the deleted subscriber test file were re-added to `notification_listeners_test.go` as shared helpers for the cmd/server integration test package.

### Files changed (backend)

- New: `server/migrations/004_remove_social_layer.up.sql`, `.down.sql`
- Deleted: `server/pkg/db/queries/{reaction,issue_reaction,subscriber,notification_preference}.sql`
- Deleted: `server/pkg/db/generated/{reaction,issue_reaction,subscriber,notification_preference}.sql.go`
- Deleted: `server/internal/handler/{reaction,issue_reaction,subscriber,notification_preference}.go`
- Deleted: `server/internal/handler/subscriber_test.go`
- Deleted: `server/cmd/server/{subscriber_listeners,subscriber_listeners_test,quick_create_subscriber_test}.go`
- Modified: `server/cmd/server/{main,router,notification_listeners,notification_listeners_test}.go`
- Modified: `server/pkg/protocol/events.go` (removed reaction/subscriber event constants)
- Modified: `server/internal/handler/{comment,activity,issue,issue_child_done}.go`
- Modified: `server/internal/service/task.go` (removed AddIssueSubscriber call in quick-create path)

### Files changed (TypeScript)

- Deleted: `packages/core/types/subscriber.ts`, `packages/core/notification-preferences/` (entire dir)
- Deleted: `packages/views/issues/hooks/{use-issue-reactions,use-issue-subscribers}.ts`
- Deleted: `packages/ui/components/common/reaction-bar.tsx`
- Deleted: `packages/views/settings/components/notifications-tab.tsx`
- Modified: `packages/core/types/{activity,comment,issue,events,index}.ts`
- Modified: `packages/core/api/{client,schemas,schemas.test,schema.test}.ts`
- Modified: `packages/core/issues/{queries,mutations,delete-cache,ws-updaters.test}.ts`
- Modified: `packages/core/realtime/use-realtime-sync.ts`
- Modified: `packages/views/issues/{components/comment-card,components/issue-detail,components/issue-detail.test,hooks/index,hooks/use-issue-timeline,hooks/use-issue-timeline.test}.tsx`
- Modified: `packages/views/settings/components/settings-page.tsx`
- Modified: `packages/views/locales/en/issues.json` (removed dead subscriber keys)

### Blockers / notes

None. All checks pass: `pnpm typecheck`, `pnpm test`, and `go build ./...`. Pre-existing Go test failures (squad prompt tests, execenv opencode, redact home directory, timing-sensitive integration tests) are unrelated to this change.
