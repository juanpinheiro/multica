"use client";

import { useState, useCallback, useMemo } from "react";
import {
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import type {
  Comment,
  TimelineEntry,
} from "@multica/core/types";
import type {
  CommentCreatedPayload,
  CommentUpdatedPayload,
  CommentDeletedPayload,
  CommentResolvedPayload,
  CommentUnresolvedPayload,
  ActivityCreatedPayload,
} from "@multica/core/types";
import {
  issueTimelineOptions,
  issueKeys,
} from "@multica/core/issues/queries";
import {
  useCreateComment,
  useUpdateComment,
  useDeleteComment,
  useResolveComment,
} from "@multica/core/issues/mutations";
import { sortTimelineEntriesAsc } from "@multica/core/issues/timeline-sort";
import { useWSEvent, useWSReconnect } from "@multica/core/realtime";
import { toast } from "sonner";
import { useT } from "../../i18n";

type TLCache = TimelineEntry[];

function commentToTimelineEntry(c: Comment): TimelineEntry {
  return {
    type: "comment",
    id: c.id,
    actor_type: c.author_type,
    actor_id: c.author_id,
    content: c.content,
    parent_id: c.parent_id,
    created_at: c.created_at,
    updated_at: c.updated_at,
    comment_type: c.type,
    attachments: c.attachments ?? [],
    resolved_at: c.resolved_at,
    resolved_by_type: c.resolved_by_type,
    resolved_by_id: c.resolved_by_id,
  };
}

export function useIssueTimeline(issueId: string, userId?: string) {
  const { t } = useT("issues");
  const qc = useQueryClient();

  const query = useQuery(issueTimelineOptions(issueId));
  const { data, isLoading: loading } = query;

  const timeline = useMemo<TimelineEntry[]>(() => data ?? [], [data]);

  const [submitting, setSubmitting] = useState(false);

  // Stable mutation handles. TanStack v5 returns a fresh result wrapper from
  // useMutation per render, but the inner mutateAsync / mutate functions are
  // stable. Pull just those so the useCallback identities downstream don't
  // flip on every parent re-render — listing the whole mutation object would
  // defeat React.memo on CommentCard.
  const { mutateAsync: createComment } = useCreateComment(issueId);
  const { mutateAsync: updateComment } = useUpdateComment(issueId);
  const { mutateAsync: deleteCommentAsync } = useDeleteComment(issueId);
  const { mutateAsync: resolveCommentAsync } = useResolveComment(issueId);

  // Reconnect recovery: invalidate so the next render refetches the full
  // timeline. Cheaper than diffing across a possibly-long disconnect.
  useWSReconnect(
    useCallback(() => {
      qc.invalidateQueries({ queryKey: issueKeys.timeline(issueId) });
    }, [qc, issueId]),
  );

  // --- WS event handlers ---

  useWSEvent(
    "comment:created",
    useCallback(
      (payload: unknown) => {
        const { comment } = payload as CommentCreatedPayload;
        if (comment.issue_id !== issueId) return;
        qc.setQueryData<TLCache>(issueKeys.timeline(issueId), (old) => {
          const entry = commentToTimelineEntry(comment);
          if (!old) return [entry];
          if (old.some((e) => e.id === comment.id)) return old;
          return sortTimelineEntriesAsc([...old, entry]);
        });
      },
      [qc, issueId],
    ),
  );

  useWSEvent(
    "comment:updated",
    useCallback(
      (payload: unknown) => {
        const { comment } = payload as CommentUpdatedPayload;
        if (comment.issue_id !== issueId) return;
        qc.setQueryData<TLCache>(issueKeys.timeline(issueId), (old) =>
          old?.map((e) =>
            e.id === comment.id ? commentToTimelineEntry(comment) : e,
          ),
        );
      },
      [qc, issueId],
    ),
  );

  // Granular handlers for comment:resolved / comment:unresolved. The payload
  // carries the full Comment with the new resolved_at/resolved_by_* fields,
  // which `commentToTimelineEntry` already preserves, so the existing
  // entry can simply be replaced in place. Without these handlers the only
  // path that updated the cache was `useRealtimeSync`'s global invalidate,
  // which forces a full timeline refetch and busts every CommentCard memo.
  useWSEvent(
    "comment:resolved",
    useCallback(
      (payload: unknown) => {
        const { comment } = payload as CommentResolvedPayload;
        if (comment.issue_id !== issueId) return;
        qc.setQueryData<TLCache>(issueKeys.timeline(issueId), (old) =>
          old?.map((e) =>
            e.id === comment.id ? commentToTimelineEntry(comment) : e,
          ),
        );
      },
      [qc, issueId],
    ),
  );

  useWSEvent(
    "comment:unresolved",
    useCallback(
      (payload: unknown) => {
        const { comment } = payload as CommentUnresolvedPayload;
        if (comment.issue_id !== issueId) return;
        qc.setQueryData<TLCache>(issueKeys.timeline(issueId), (old) =>
          old?.map((e) =>
            e.id === comment.id ? commentToTimelineEntry(comment) : e,
          ),
        );
      },
      [qc, issueId],
    ),
  );

  useWSEvent(
    "comment:deleted",
    useCallback(
      (payload: unknown) => {
        const { comment_id, issue_id } = payload as CommentDeletedPayload;
        if (issue_id !== issueId) return;
        qc.setQueryData<TLCache>(issueKeys.timeline(issueId), (old) => {
          if (!old) return old;
          // Cascade through replies (full timeline now lives in this single
          // cache, so a flat sweep is sufficient).
          const idsToRemove = new Set<string>([comment_id]);
          let changed = true;
          while (changed) {
            changed = false;
            for (const e of old) {
              if (
                e.parent_id &&
                idsToRemove.has(e.parent_id) &&
                !idsToRemove.has(e.id)
              ) {
                idsToRemove.add(e.id);
                changed = true;
              }
            }
          }
          return old.filter((e) => !idsToRemove.has(e.id));
        });
      },
      [qc, issueId],
    ),
  );

  useWSEvent(
    "activity:created",
    useCallback(
      (payload: unknown) => {
        const p = payload as ActivityCreatedPayload;
        if (p.issue_id !== issueId) return;
        const entry = p.entry;
        if (!entry || !entry.id) return;
        qc.setQueryData<TLCache>(issueKeys.timeline(issueId), (old) => {
          if (!old) return [entry];
          if (old.some((e) => e.id === entry.id)) return old;
          return sortTimelineEntriesAsc([...old, entry]);
        });
      },
      [qc, issueId],
    ),
  );

  // --- Mutation functions ---

  const submitComment = useCallback(
    async (content: string, attachmentIds?: string[]) => {
      if (!content.trim() || submitting || !userId) return;
      setSubmitting(true);
      try {
        await createComment({ content, attachmentIds });
      } catch (err) {
        toast.error(
          err instanceof Error && err.message
            ? err.message
            : t(($) => $.comment.send_failed),
        );
      } finally {
        setSubmitting(false);
      }
    },
    [userId, submitting, createComment, t],
  );

  const submitReply = useCallback(
    async (parentId: string, content: string, attachmentIds?: string[]) => {
      if (!content.trim() || !userId) return;
      try {
        await createComment({
          content,
          type: "comment",
          parentId,
          attachmentIds,
        });
      } catch (err) {
        toast.error(
          err instanceof Error && err.message
            ? err.message
            : t(($) => $.comment.send_reply_failed),
        );
      }
    },
    [userId, createComment, t],
  );

  const editComment = useCallback(
    async (commentId: string, content: string, attachmentIds?: string[], removeAttachmentIds?: string[]) => {
      try {
        await updateComment({ commentId, content, attachmentIds, removeAttachmentIds });
      } catch (err) {
        toast.error(
          err instanceof Error && err.message
            ? err.message
            : t(($) => $.comment.update_failed),
        );
      }
    },
    [updateComment, t],
  );

  const deleteComment = useCallback(
    async (commentId: string) => {
      try {
        await deleteCommentAsync(commentId);
      } catch (err) {
        toast.error(
          err instanceof Error && err.message
            ? err.message
            : t(($) => $.comment.delete_failed),
        );
      }
    },
    [deleteCommentAsync, t],
  );

  const toggleResolveComment = useCallback(
    async (commentId: string, resolved: boolean) => {
      try {
        await resolveCommentAsync({ commentId, resolved });
      } catch (err) {
        toast.error(
          err instanceof Error && err.message
            ? err.message
            : resolved
              ? t(($) => $.comment.resolve.resolve_failed)
              : t(($) => $.comment.resolve.unresolve_failed),
        );
      }
    },
    [resolveCommentAsync, t],
  );

  return {
    timeline,
    loading,
    submitting,
    submitComment,
    submitReply,
    editComment,
    deleteComment,
    toggleResolveComment,
  };
}
