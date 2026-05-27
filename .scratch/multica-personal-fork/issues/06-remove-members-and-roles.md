# 06 — Remove workspace members and roles

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/multica-personal-fork/PRD.md`

## What to build

Remove workspace members management and the role system (`owner` / `admin` / `member`). With a singleton implicit user, every workspace has exactly one implicit member who is implicitly the owner. The role-check middleware that exists today is reduced to a no-op pass-through so that subsequent issues don't need to re-route every endpoint.

The `workspace_member` table is dropped as part of issue 17 (consolidate-migrations); this issue only removes the code that reads/writes it.

## Acceptance criteria

- [x] Member-management handlers in `handler/workspace.go` removed: `UpdateMember`, `DeleteMember`, `LeaveWorkspace`; `ListMembersWithUser` kept (read-only, 50+ consumers)
- [x] `handler/workspace_revoke.go` deleted
- [x] `packages/views/members/` deleted
- [x] `packages/views/settings/components/members-tab.tsx` deleted; the tab is removed from the settings page navigation
- [x] `apps/web/app/[workspaceSlug]/(dashboard)/members/` route deleted
- [x] `middleware/RequireWorkspaceRoleFromURL` reduced to a no-op pass-through that delegates to `RequireWorkspaceMemberFromURL`
- [x] `middleware/RequireWorkspaceMember` simplified: resolves and attaches workspace context; synthesizes `db.Member{Role: "owner"}` without any DB membership lookup
- [x] `GetAssigneeFrequency` endpoint reviewed: kept (dashboard uses it)
- [x] `pnpm typecheck`, `pnpm test`, and `go test ./internal/handler/... ./cmd/server/...` pass

## Notes

- `packages/core/paths/paths.ts`: `memberDetail` path removed (route deleted)
- `packages/core/types/events.ts`: `member:added`, `member:updated`, `member:removed` WS event types removed
- `packages/core/realtime/use-realtime-sync.ts`: member event handlers removed
- `packages/core/api/client.ts`: `updateMember`, `deleteMember`, `leaveWorkspace` API methods removed
- `packages/core/workspace/mutations.ts`: `useLeaveWorkspace` mutation removed
- `packages/views/settings/components/workspace-tab.tsx`: leave-workspace UI removed; canManageWorkspace/isOwner hardcoded to true
- `packages/views/common/actor-avatar.tsx`: member hover card removed (no profile page); member profile link removed
- Tests removed: revocation tests, membership cache invalidation tests for delete/update/leave, GitHub role-gating router test, integration test for non-owner delete protection (all tested multi-user concepts no longer applicable)

## Blocked by

- 05-remove-invitations
