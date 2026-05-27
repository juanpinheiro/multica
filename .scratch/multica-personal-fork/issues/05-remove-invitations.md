# 05 — Remove invitations feature

**Status:** `done`

## Parent

`.scratch/multica-personal-fork/PRD.md`

## What to build

Remove the workspace invitation system end-to-end. The fork has a single user — there's no one to invite. Delete the invitation handlers, the related API routes, the inviter and invitee frontend flows, and the related queries/mutations in `@multica/core`.

The `invitation` database table is dropped as part of issue 17 (consolidate-migrations); this issue only removes the code that reads/writes it.

## Acceptance criteria

- [x] `handler/invitation.go` and `handler/invitation_test.go` deleted
- [x] `CreateInvitation`, `RevokeInvitation`, `ListWorkspaceInvitations`, `ListMyInvitations`, `GetMyInvitation`, `AcceptInvitation`, `DeclineInvitation` handlers removed
- [x] Corresponding route declarations removed from `cmd/server/router.go`: `POST /api/workspaces/{id}/members`, `DELETE /api/workspaces/{id}/invitations/{invitationId}`, `GET /api/workspaces/{id}/invitations`, `/api/invitations` group (list/get/accept/decline)
- [x] `packages/views/invitations/` and `packages/views/invite/` deleted
- [x] `packages/core/` invitation queries / mutations deleted
- [x] `apps/web/app/(auth)/invitations/` and `apps/web/app/(auth)/invite/` route directories deleted *(routes lived under `(auth)/`, not `(landing)/` — AC path was stale)*
- [x] Any references to invitations in `views/settings/` or `views/layout/` cleaned (e.g. "pending invitations" badges in the sidebar)
- [x] `pnpm typecheck`, `pnpm test`, and `go test ./internal/handler/...` pass *(broader `go test ./...` shows pre-existing daemon/repocache failures unrelated to invitations — environmental on Windows, present before this issue)*

## Blocked by

None - can start immediately

## Comments

### Key decisions

- **Deleted the dead `(h *Handler) CreateMember` Go handler and its `CreateMemberRequest` type.** The route `POST /api/workspaces/{id}/members` was wired to `h.CreateInvitation`, not `h.CreateMember`. With the route gone and no other callers, the `CreateMember` handler was unreachable dead code. Issue 06's AC doesn't name it but it falls naturally into the members-cleanup scope — preserving dead code through one more issue felt worse than the small scope expansion. Same reasoning for dropping `CreateMemberRequest` from the TS types.
- **Reserved-slugs trim deferred to issue 19.** Both `invite` and `invitations` entries stay in `server/internal/handler/reserved_slugs.json` for now. Issue 19 (docs-rewrite) is the canonical pass that reduces the reserved list to only what corresponds to real routes; touching it here would conflict with that work. CI still verifies the generated `packages/core/paths/reserved-slugs.ts` matches the JSON, so the two stay in sync.
- **Migration files left untouched.** `server/migrations/041_workspace_invitation.{up,down}.sql` stays. Issue 17 (consolidate-migrations) collapses every historical migration into a single `001_init.sql` after all feature deletions land; deleting individual `up`/`down` pairs now would just create churn for that pass.
- **Email service kept; only invitation methods removed.** `internal/service/email.go` retains `SendVerificationCode` (used by magic-link auth, which lives until issue 09). Dropped `SendInvitationEmail`, `buildInvitationParams`, `sanitizeSubjectField`, `maxSubjectFieldRunes` constant, and the now-orphaned `html` / `unicode` / `unicode/utf8` imports. The whole `email_test.go` only tested `sanitizeSubjectField` / `buildInvitationParams` — deleted with the implementation.
- **Auth-callback test rewrite, not patch.** The old test (`apps/web/app/auth/callback/page.test.tsx`) had 7 test cases, 5 of which exercised the invitations lookup path. Rewriting it as a smaller, focused suite (nextUrl honored, unsafe nextUrl ignored, default destination resolution) is cleaner than mutating each test individually. The remaining behaviors (nextUrl honoring, unsafe-target rejection, default routing) are the bits that matter without invitations.
- **`resolveLoggedInDestination` collapsed in the web login page.** The old `async` helper that branched on invitations was the only async caller — turned it into a thin sync wrapper over `resolvePostAuthDestination`. Then dropped the unused `QueryClient` type import and `paths` (no longer needed). `api` import preserved — still used by `api.issueCliToken()` in the desktop-handoff branch.
- **WS event listeners (`invitation:created/accepted/declined/revoked`) and the `member:added` invitation-cache invalidation removed together.** They were the only consumers of `workspaceKeys.myInvitations()` / `workspaceKeys.invitations(...)`; removing the listeners and the keys in a single pass keeps the realtime sync surface coherent and the build green.
- **Stale workspace-creation hint (`new_page.invite_hint`) and its UI render dropped.** "You can invite teammates once your workspace is ready" no longer applies to a single-user fork. EN + zh-Hans locale entries dropped; the `<p>` that rendered the hint in `new-workspace-page.tsx` removed.
- **Sidebar `getShareableUrl` / `useNavigation().push` dropped.** Once the invitation accept mutation was gone, `push` was unused — removed from the destructure. `paths` import stays (used to navigate to the active workspace in the workspace switcher).
- **`packages/views/package.json` exports trimmed.** `./invite` and `./invitations` exports removed alongside the deleted directories.

### Files changed

**Deleted**
- Backend: `server/internal/handler/invitation.go`, `server/internal/handler/invitation_test.go`, `server/pkg/db/queries/invitation.sql`, `server/pkg/db/generated/invitation.sql.go`, `server/internal/service/email_test.go`
- Frontend: `packages/views/invitations/` (3 files), `packages/views/invite/` (2 files), `apps/web/app/(auth)/invitations/`, `apps/web/app/(auth)/invite/`, `packages/views/locales/en/invite.json`, `packages/views/locales/zh-Hans/invite.json`

**Modified — backend**
- `server/cmd/server/router.go`: dropped workspace-scoped invitation routes (`/invitations`, `POST /members`, `DELETE /invitations/{invitationId}`) and the user-scoped `/api/invitations*` group.
- `server/cmd/server/listeners.go`: dropped `EventInvitationCreated` / `EventInvitationRevoked` from `personalEvents` map and the two `bus.Subscribe(...)` blocks for them. Doc comment updated.
- `server/internal/handler/workspace.go`: deleted dead `(h *Handler) CreateMember` handler and `CreateMemberRequest` type; trimmed stale `AcceptInvitation` mention from `CreateWorkspace` doc comment.
- `server/internal/handler/workspace_test.go`: trimmed stale `AcceptInvitation` mention from `TestCreateWorkspace_DoesNotMarkOnboarded` doc comment.
- `server/internal/handler/handler_test.go`: deleted `TestRevokeInvitationRejectsMalformedInvitationID` and `TestGetMyInvitationRejectsMalformedID`.
- `server/internal/handler/onboarding.go`: dropped `OnboardingPathInviteAccept` entry from `validCompletionPaths`.
- `server/internal/service/email.go`: dropped `SendInvitationEmail`, `buildInvitationParams`, `sanitizeSubjectField`, `maxSubjectFieldRunes`, and the now-unused `html` / `unicode` / `unicode/utf8` imports.
- `server/internal/analytics/events.go`: dropped `EventTeamInviteSent`, `EventTeamInviteAccepted`, `TeamInviteSent()`, `TeamInviteAccepted()`, `OnboardingPathInviteAccept`.
- `server/pkg/protocol/events.go`: dropped the four `EventInvitation*` constants.
- `server/pkg/db/generated/models.go`: dropped `WorkspaceInvitation` struct.

**Modified — frontend**
- `packages/core/types/workspace.ts`: dropped `Invitation` interface.
- `packages/core/types/index.ts`: dropped `Invitation` export.
- `packages/core/types/api.ts`: dropped `CreateMemberRequest` interface.
- `packages/core/types/events.ts`: dropped four `invitation:*` literals from `WSEventType`, four `Invitation*Payload` interfaces, the corresponding `WSEventPayloadMap` entries, and the unused `Invitation` type import.
- `packages/core/workspace/queries.ts`: dropped `invitations` / `myInvitations` keys and `invitationListOptions` / `myInvitationListOptions` query factories.
- `packages/core/api/client.ts`: dropped `createMember`, `listWorkspaceInvitations`, `revokeInvitation`, `listMyInvitations`, `getInvitation`, `acceptInvitation`, `declineInvitation`, plus the `Invitation` and `CreateMemberRequest` type imports.
- `packages/core/realtime/use-realtime-sync.ts`: dropped the four `invitation:*` listener subscriptions, their `unsub*()` cleanup calls, the `InvitationCreatedPayload` type import, and the `workspaceKeys.myInvitations()` invalidation inside `member:added`.
- `packages/core/paths/paths.ts`: dropped `paths.invite(id)`, `paths.invitations()`, and the `/invite/` / `/invitations` entries from `GLOBAL_PREFIXES`; updated module docblock.
- `packages/core/paths/paths.test.ts`: dropped assertions for the removed paths.
- `packages/core/paths/consistency.test.ts`: dropped `/invite/` from `globalPrefixes` test fixture.
- `packages/core/paths/resolve.ts`: trimmed stale invitation/desktop mentions from the docblock.
- `packages/views/i18n/resources-types.ts`: dropped `invite` namespace type augmentation.
- `packages/views/locales/index.ts`: dropped `enInvite` / `zhHansInvite` imports and `invite` resource entries.
- `packages/views/locales/en/layout.json`, `packages/views/locales/zh-Hans/layout.json`: dropped sidebar invitation labels.
- `packages/views/locales/en/settings.json`, `packages/views/locales/zh-Hans/settings.json`: dropped members-tab invitation strings (invite form labels, toasts, pending section).
- `packages/views/locales/en/workspace.json`, `packages/views/locales/zh-Hans/workspace.json`: dropped `new_page.invite_hint`.
- `packages/views/workspace/new-workspace-page.tsx`: dropped the `<p>` render of `invite_hint`.
- `packages/views/layout/app-sidebar.tsx`: dropped the workspace-switcher invitation badge ring, the pending-invitations section, both mutations, the unused `useMutation` / `useQueryClient` / `workspaceKeys` / `myInvitationListOptions` imports, the `EMPTY_INVITATIONS` constant, and the `push` destructure from `useNavigation`.
- `packages/views/layout/app-sidebar.test.tsx`: dropped invitation-related mocks from the `@multica/core/workspace/queries` mock.
- `packages/views/settings/components/members-tab.tsx`: dropped `InvitationRow` component, the invite form section, `invitationListOptions` query, `handleInviteMember` / `handleRevokeInvitation`, the `inviteEmail` / `inviteRole` / `inviteLoading` / `invitationActionId` state, the unused `roleConfig` local, and the now-unused `useState` imports (`Plus`, `X`, `Clock`, `Mail` icons; `Input`, `Card`, `CardContent`, `Select*` components; `CreateMemberRequest`, `Invitation` types).
- `packages/views/package.json`: dropped `./invite` and `./invitations` exports.
- `apps/web/app/(auth)/login/page.tsx`: collapsed async `resolveLoggedInDestination` into a thin sync wrapper over `resolvePostAuthDestination`; dropped `QueryClient` and `paths` imports.
- `apps/web/app/auth/callback/page.tsx`: dropped the invitations-lookup branch from the post-OAuth resolver; simplified the success path to nextUrl-or-default.
- `apps/web/app/auth/callback/page.test.tsx`: rewritten as a focused suite (nextUrl honored, unsafe nextUrl ignored, default destination resolution).

### Verification

- `pnpm typecheck` → 4/4 packages green.
- `pnpm test` → 91 files / 815 tests pass across `@multica/core`, `@multica/ui`, `@multica/views`, `@multica/web`.
- `cd server && go build ./...` → clean.
- `cd server && go vet ./...` → clean.
- `cd server && go test ./internal/handler/ ./internal/service/ ./internal/analytics/ ./cmd/server/ ./pkg/protocol/` → 1159 tests pass across 5 packages.

### Blockers / notes for next iteration

- `server/migrations/041_workspace_invitation.{up,down}.sql` still exist on disk — issue 17 deletes them as part of the migration consolidation.
- `server/internal/handler/reserved_slugs.json` still includes `invite` and `invitations` slugs — issue 19's docs/reserved-slugs trim removes them.
- Stale "邀请同事" / "leave_confirm_description" mentions linger in `zh-Hans/onboarding.json` and `zh-Hans/settings.json`. Issue 08 (remove onboarding) deletes the onboarding strings outright; issue 06 (remove members) rewrites the leave flow. Out of scope here.
- The `members-tab.tsx` file is now significantly trimmed but still contains role-management UI (UpdateMember, DeleteMember, LeaveWorkspace surfaces). Issue 06 deletes that surface entirely; this issue's cleanup left it functional in the meantime so workspace-admin role changes still work.
- `apps/web/app/(auth)/login/page.tsx` still references the deleted `/download` route via a `<Link>` ("Prefer desktop? Download"). Issue 01 noted this as a transient state; issue 09 (loopback auth + singleton user) rewrites the entire login page. Acceptable to leave dangling until then — typecheck/tests don't catch it.
- Daemon and repocache Go test packages have pre-existing environmental failures on Windows (missing claude/codex/kimi/etc. binaries on PATH, git symlink-rights issues). Not introduced by this issue and not invitation-related.
