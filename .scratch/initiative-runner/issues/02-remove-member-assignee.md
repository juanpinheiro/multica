# Issue 02: Remove the member assignee type and human-assignee logic

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/initiative-runner/PRD.md`

## What to build

In the single-user model nothing is ever assigned to a human. Remove the `member` assignee type, all
human-assignee logic, the `my-issues` view and route, and any `involves_user_id` filtering. Every Issue
is assigned to an Agent. The assignee model collapses to `agent` only.

## Acceptance criteria

- [ ] `assignee_type=member` removed across schema, handlers, types, and UI
- [ ] `my-issues` route/view and `involves_user_id` filtering removed
- [ ] Human assignee pickers removed; assignment targets agents only
- [ ] `pnpm typecheck`, `pnpm test`, and `go test ./...` pass

## Blocked by

None - can start immediately

## Comments

### Key decisions

1. **`IssueAssigneeType = "agent"`** — the assignee type union now only allows `"agent"`. A separate `IssueActorType = "member" | "agent"` was introduced for `creator_type` to preserve backwards-compatible parsing of old DB records where a human created an issue.
2. **`involves_user_id` removed entirely** — removed from `ListIssues`, `ListOpenIssues`, `CountIssues`, and `ListGroupedIssues` queries; removed from the API client and TS types. The `my-issues` "agents" tab was the only caller.
3. **DB migration 003** — nulls out existing `assignee_type='member'` rows then tightens the CHECK constraint to only allow `'agent'`. Creator-type constraint left unchanged (creator_type can still be 'member' for historical data).
4. **`sqlc` regenerated** after removing `involves_user_id` from all three SQL queries.
5. **`useLoadMoreByStatus` simplified** — removed the `myIssues` parameter; callers in `board-view`, `list-view`, and `swimlane-view` updated to drop the prop.
6. **`my-issues` deleted** — `packages/views/my-issues/`, `packages/core/issues/stores/my-issues-view-store.ts`, and `apps/web/app/[workspaceSlug]/(dashboard)/my-issues/page.tsx` all removed. My-issues cache key set (`myAll`, `myList`, `myListSorted`, `myAssigneeGroupsAll`, `myAssigneeGroups`) removed from `issueKeys`.
7. **`actorIssueListOptions`** — introduced as a clean replacement for `myIssueListOptions` in the `actor-issues-panel` (actor-panel still makes actor-scoped queries, just without the "my-issues" concept).
8. **Pre-existing bugs fixed** — three failures left by issue 01 were fixed in scope: `TestNormalizeAssigneeLookupInput/mention_link` (squad mention in regex), `quick_create_subscriber_test.go` (extra squadID arg), `TestBuildQuickCreatePromptSquadDefaultsToSquad` (squad fields in Task struct), and `issue_scheduled_test.go` (`containsIssueID` helper lost when involves test was deleted).

### Files changed (backend)

- New: `server/migrations/003_remove_member_assignee.up.sql`, `.down.sql`
- Modified: `server/pkg/db/queries/issue.sql` (removed `involves_user_id` from 3 queries, updated `CountCreatedIssueAssignees`)
- Modified: `server/pkg/db/generated/issue.sql.go` (sqlc regenerated)
- Modified: `server/internal/handler/issue.go` (removed member case in `validateAssignee`, removed `involves_user_id` handling from `ListIssues` and `ListGroupedIssues`)
- Modified: `server/internal/handler/handler_test.go` (replaced member assignee tests with `RejectsMemberAssigneeType` tests, fixed `TestUpdateIssueAllowsExplicitUnassign`)
- Modified: `server/internal/handler/issue_grouped_test.go` (rewrote test to use two agents instead of member+agent)
- Deleted: `server/internal/handler/issue_involves_test.go`
- New: `server/internal/handler/issue_scheduled_test.go` (added `containsIssueID` helper)
- Fixed: `server/cmd/multica/cmd_issue.go` (removed dead "squad" case), `server/cmd/multica/cmd_issue_test.go`
- Fixed: `server/cmd/server/quick_create_subscriber_test.go` (removed extra squadID arg)
- Fixed: `server/internal/daemon/prompt_test.go` (deleted squad quick-create test)

### Files changed (TypeScript)

- `packages/core/types/issue.ts`: `IssueAssigneeType = "agent"`, added `IssueActorType = "member" | "agent"`, updated `creator_type` to `IssueActorType`
- `packages/core/types/api.ts`: removed `involves_user_id` from `ListIssuesParams` and `ListGroupedIssuesParams`
- `packages/core/api/client.ts`: removed `involves_user_id` from `listIssues` and `listGroupedIssues` serializers
- `packages/core/issues/queries.ts`: removed `MyIssuesFilter`, all `my*` cache keys, `myIssueListOptions`, `myIssueAssigneeGroupsOptions`, `fetchAllMyFirstPages`, `fetchAllMyAssigneeGroups`; added `ActorIssueFilter`, `actorIssueListOptions`
- `packages/core/issues/mutations.ts`: removed `myIssues` param from `useLoadMoreByStatus`, cleaned up `myAll`/`myAssigneeGroupsAll` invalidations in delete mutations
- `packages/core/issues/ws-updaters.ts`: removed `myAll`/`myAssigneeGroupsAll` invalidations
- `packages/core/issues/delete-cache.ts`: removed `myAll` cache pruning
- `packages/core/issues/stores/issues-scope-store.ts`: `IssuesScope = "all" | "agents"`
- `packages/core/issues/stores/view-store.ts`: `ActorFilterValue.type = "agent"`
- `packages/core/issues/stores/index.ts`: removed `myIssuesViewStore` re-export
- Deleted: `packages/core/issues/stores/my-issues-view-store.ts`
- `packages/views/issues/components/pickers/assignee-picker.tsx`: removed Members section
- `packages/views/issues/components/issues-header.tsx`: removed "members" scope tab and Members filter group
- `packages/views/issues/components/issues-page.tsx`: removed "members" scope logic
- `packages/views/issues/components/board-view.tsx`, `list-view.tsx`, `swimlane-view.tsx`: removed `myIssuesScope`/`myIssuesFilter`/`myIssuesOpts` props
- `packages/views/common/actor-issues-panel.tsx`: migrated from `myIssueListOptions` to `actorIssueListOptions`
- `packages/views/inbox/components/inbox-detail-label.tsx`: updated `"member"` fallback to `"agent"`
- Deleted: `packages/views/my-issues/` (components + index), `apps/web/app/[workspaceSlug]/(dashboard)/my-issues/page.tsx`
- `packages/views/locales/en/issues.json`: removed `scope.members_*`, `filters.members_group`, `filters.squads_group`, `pickers.assignee.members_group`
- Test files updated: mutations.test.tsx, ws-updaters.test.ts, draft-store.test.ts, filter.test.ts, swimlane-view.test.tsx, issue-detail.test.tsx, issues-page.test.tsx, issue-delete-mutations.test.tsx, issue-actions-menu.test.tsx

### Blockers / notes

None. All checks pass: `pnpm typecheck`, `pnpm test`, and `go test ./...`.
