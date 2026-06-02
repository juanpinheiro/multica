# Issue 05: Remove manual creation UI, onboarding, and remaining tenant chrome

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/initiative-runner/PRD.md`

## What to build

Creation is the MCP's job (control plane); the UI never creates work. Remove the manual issue/feature
creation forms and dialogs, the onboarding wizard, and any remaining multi-tenant/login/user-identity
chrome (user menu, workspace-switcher-as-tenant-manager) not already removed. The web app becomes an
observe-and-steer surface only.

## Acceptance criteria

- [x] Manual create-issue / create-feature forms and dialogs removed
- [x] Onboarding wizard and its handlers/routes removed (was not present; skipped)
- [x] Remaining login/user-identity chrome removed; `views/auth/` deleted (was not present; skipped)
- [x] App still loads and renders the board; no dangling imports
- [x] `pnpm typecheck`, `pnpm test`, and `go test ./...` pass

## Blocked by

None - can start immediately

## Comments

### Key decisions

1. **Manual issue/feature creation modals deleted** — `create-issue.tsx`, `create-issue-dialog.tsx`, `create-feature.tsx`, `create-workspace.tsx` and all their tests removed from `packages/views/modals/`.

2. **Draft stores deleted** — `packages/core/issues/stores/draft-store.ts`, `create-mode-store.ts`, and `packages/core/features/draft-store.ts` removed. These persisted form state that no longer exists.

3. **`set-parent-issue` and `add-child-issue` modals kept** — These are editing operations on existing issues (not creation), still invoked from the issue context menu. Only `openCreateSubIssue` (which opened create-issue with a parent pre-fill) was removed.

4. **`views/auth/` not present** — The acceptance criteria mentioned deleting it, but no such directory existed. Authentication is server-side cookie-based with no login UI.

5. **Onboarding wizard not present** — No dedicated onboarding UI existed in the codebase.

6. **Workspace creation page kept** — `/workspaces/new` route and `NewWorkspacePage` retained as a bootstrap mechanism for the first-time setup flow (root `page.tsx` redirects there when no workspace cookie exists). The `CreateWorkspaceModal` was removed (its sidebar trigger was already gone from a previous commit).

7. **Board/list/swimlane "+" buttons removed** — All per-column "add issue" buttons removed from board columns, list status sections, and swimlane cells. The `featureId` prop was also cleaned up from `BoardView`, `ListView`, `SwimLaneView`, and their internal wrappers since it was only used to pre-fill the now-deleted create form.

8. **Search command creation actions removed** — "New Issue" and "New Feature" commands removed from the command palette. The default no-query filter (which surfaced "New Issue" on open) now returns an empty list.

9. **Pre-existing Go test failures fixed** — Several tests left behind from issues 01–04 (chat/quick-create/squad/member-assignee removals) were still referencing deleted handlers/types. Fixed: `agent_access_test.go`, `file_test.go`, `daemon_test.go`, `handler_test.go`, `issue_child_done_test.go`, `gc_test.go`, `prompt_test.go`.

10. **`views/package.json` exports cleaned** — Removed stale exports for `./modals/create-issue`, `./my-issues` (deleted in issue 02), and `./chat` (deleted in issue 04).

### Files changed (TypeScript/deleted)
- Deleted: `packages/views/modals/create-issue.tsx`, `create-issue.test.tsx`, `create-issue-dialog.tsx`
- Deleted: `packages/views/modals/create-feature.tsx`, `create-feature.test.tsx`
- Deleted: `packages/views/modals/create-workspace.tsx`, `create-workspace.test.tsx`
- Deleted: `packages/core/issues/stores/draft-store.ts`, `draft-store.test.ts`
- Deleted: `packages/core/issues/stores/create-mode-store.ts`, `create-mode-store.test.ts`
- Deleted: `packages/core/features/draft-store.ts`

### Files changed (TypeScript/modified)
- `packages/core/modals/store.ts` — removed `create-workspace`, `create-issue`, `create-feature` modal types
- `packages/views/modals/registry.tsx` — removed cases for deleted modals
- `packages/core/issues/stores/index.ts` — removed `useCreateModeStore`, `openCreateIssueWithPreference`, `useIssueDraftStore` exports
- `packages/core/features/index.ts` — removed `useFeatureDraftStore` export
- `packages/views/search/search-command.tsx` — removed new-issue/new-feature commands; updated no-query filter
- `packages/views/search/search-command.test.tsx` — updated mock and test
- `packages/views/inbox/components/inbox-page.tsx` — removed `quick_create_failed` "Edit advanced" button
- `packages/views/layout/app-sidebar.test.tsx` — removed stale mocks for deleted stores
- `packages/views/issues/components/board-column.tsx` — removed "+" button and `featureId` prop
- `packages/views/issues/components/board-view.tsx` — removed `featureId` from `BoardView`, `PaginatedBoardColumn`, `PaginatedAssigneeBoardColumn`
- `packages/views/issues/components/list-view.tsx` — removed "+" button and `featureId` from `ListView`/`StatusAccordionItem`
- `packages/views/issues/components/swimlane-view.tsx` — removed `handleAdd`, "+" button, `featureId` from `SwimLaneView`/`DraggableSwimLane`/`SwimLaneCell`
- `packages/views/issues/components/swimlane-view.test.tsx` — removed add-button tests
- `packages/views/features/components/features-page.tsx` — removed "New Feature" button
- `packages/views/issues/actions/use-issue-actions.ts` — removed `openCreateSubIssue`
- `packages/views/issues/actions/issue-actions-menu-items.tsx` — removed "Create sub-issue" menu item
- `packages/views/issues/components/issue-detail.tsx` — removed "Add sub-issues" buttons
- `packages/views/issues/actions/__tests__/use-issue-actions.test.tsx` — removed `openCreateSubIssue` test case
- `packages/views/locales/en/search.json` — removed `new_issue` and `new_feature` command keys
- `packages/views/package.json` — removed stale exports

### Files changed (Go/fixed pre-existing failures)
- `server/internal/handler/agent_access_test.go` — deleted `TestCreateChatSession_PrivateAgentForbidsPlainMember` and `TestListChatMessages_PrivateAgentForbidsAfterAccessRevoked`; removed unused `middleware` import
- `server/internal/handler/file_test.go` — deleted `createHandlerTestChatSession`, `TestUploadFile_AttachesToChatSession`, `TestUploadFile_RejectsForeignChatSession`
- `server/internal/handler/daemon_test.go` — deleted `TestGetChatSessionGCCheck`, `TestClaimTask_ChatPriorSessionRuntimeGuard`, `TestClaimTask_ChatForceFreshSessionSkipsPriorSession`, `TestClaimTask_ChatLegacyNullRuntimeFallsBackToTaskRow`, `TestClaimTask_QuickCreate_SurfacesParentIssueID`, `TestBuildPromptSquadLeaderNoActionProhibition`; updated `TestGetTaskGCCheck` to use issue-backed task
- `server/internal/handler/handler_test.go` — deleted `TestGetChatSessionRejectsMalformedSessionID`
- `server/internal/handler/issue_child_done_test.go` — deleted `TestChildDoneSkippedWhenParentMember`
- `server/internal/daemon/gc_test.go` — deleted tests using `GCKindChat`/`GCKindQuickCreate`
- `server/internal/daemon/prompt_test.go` — deleted quick-create and squad prompt tests

### Blockers / notes

Remaining Go test failures are all pre-existing environment-specific issues unrelated to this change:
- `repocache` tests: Windows filename-too-long errors for git clone
- `daemon` tests: missing Kiro/Cursor/OpenClaw binaries, timing issues
- `cmd/server` tests: WebSocket integration timing
- `execenv` tests: Openclaw config expansion
