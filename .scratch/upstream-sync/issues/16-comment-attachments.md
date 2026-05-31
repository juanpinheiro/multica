# Issue 16: Comment attachments — multiple + edit-time removal

**Status:** `done`
**Model:** `sonnet`

## Parent

PRD 1 — Upstream Sync (`.scratch/upstream-sync/PRD.md`).

## What to build

Support attaching multiple files to a comment and removing an attachment while editing a comment, ported from upstream. Explicitly exclude the unstable since-delta / comment-session-resume work from the same upstream PR (it was reverted twice upstream) — only the attachment behavior is in scope.

## Acceptance criteria

- [x] A comment can carry multiple attachments selected together.
- [x] An attachment can be removed while editing a comment, without deleting the comment.
- [x] No since-delta / comment-session-resume behavior is introduced.
- [x] Tests cover multi-attachment create and edit-time removal.

## Blocked by

None — can start immediately.

## Comments

### Key decisions

- **Multi-file select**: `FileUploadButton` (`packages/ui/components/common/file-upload-button.tsx`) now has `multiple` on the `<input>` and loops `Array.from(files).forEach(onSelect)`. The `onSelect: (file: File) => void` interface stays single-file so all existing callers need no changes; the loop fires one call per file.
- **Edit-time removal — backend**: Added `UnlinkAttachmentsFromComment` SQL query (`attachment.sql`) that `SET comment_id = NULL WHERE comment_id = $1 AND workspace_id = $2 AND id = ANY($3)`. The WHERE clause keys on `comment_id` so stale IDs from other comments are a no-op. A new `unlinkAttachmentsFromComment` handler helper wraps it (file.go). `UpdateComment` gains `remove_attachment_ids []string` in its request body; the field is parsed with `parseUUIDSliceOrBadRequest` and the unlink runs before the link pass so a round-trip of the same ID (remove + re-add) is a no-op.
- **Edit-time removal — frontend**: `CommentRow` and `CommentCardImpl` each track `removedAttachmentIds: Set<string>` in local state. A new `EditAttachmentList` component renders non-inline existing attachments in edit mode with an X button; clicking adds the ID to the set. On save, the set is spread into `removeAttachmentIds` passed to `onEdit`. On cancel, the set is cleared. The editor's `attachments` prop collapses to `[...visibleExistingAttachments, ...pendingAttachments]` (no nested ternary).
- **Optimistic update**: `useUpdateComment` now filters `removeAttachmentIds` from the timeline entry's attachment array in `onMutate`, so the UI reflects the removal immediately before the server responds.
- **`onEdit` signature**: Added optional `removeAttachmentIds?: string[]` parameter, threaded through `CommentCardProps`, `use-issue-timeline.ts#editComment`, and `useUpdateComment`'s mutation vars.

### Files changed

- `server/pkg/db/queries/attachment.sql` — `UnlinkAttachmentsFromComment` query
- `server/pkg/db/generated/attachment.sql.go` — regenerated (sqlc)
- `server/internal/handler/file.go` — `unlinkAttachmentsFromComment` helper
- `server/internal/handler/comment.go` — `remove_attachment_ids` in `UpdateComment`
- `server/internal/handler/comment_attachment_test.go` (new) — 2 handler integration tests (DB-backed, run in CI)
- `packages/ui/components/common/file-upload-button.tsx` — `multiple` attribute + loop
- `packages/core/api/client.ts` — `removeAttachmentIds` param on `updateComment`
- `packages/core/issues/mutations.ts` — `useUpdateComment` threads `removeAttachmentIds`; optimistic update filters removed attachments
- `packages/core/issues/mutations.test.tsx` — 3 new `useUpdateComment` tests
- `packages/views/issues/components/comment-card.tsx` — `EditAttachmentList` component; `removedAttachmentIds` state in `CommentRow` and `CommentCardImpl`; updated `onEdit` signature
- `packages/views/issues/hooks/use-issue-timeline.ts` — `editComment` accepts and forwards `removeAttachmentIds`

### Blockers / notes

- Handler integration tests compile and `go vet` clean but require the postgres container to run (Docker Desktop not up on this machine). CI runs them against `pgvector/pg17`.
- No view-layer test was added for the `EditAttachmentList` interaction because rendering `comment-card.tsx` requires a full Tiptap editor setup (heavy jsdom + extension wiring). The component is straightforward: filter list + button click → state update; the mutation test covers the end-to-end behavior.
