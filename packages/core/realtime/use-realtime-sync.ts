"use client";

import { useEffect, useRef } from "react";
import { useQueryClient, type QueryClient } from "@tanstack/react-query";
import type { WSClient } from "../api/ws-client";
import type { StoreApi, UseBoundStore } from "zustand";
import type { AuthState } from "../auth/store";
import { createLogger } from "../logger";
import { clearWorkspaceStorage } from "../platform/storage-cleanup";
import { defaultStorage } from "../platform/storage";
import { getCurrentWsId, getCurrentSlug } from "../platform/workspace-storage";
import { issueKeys } from "../issues/queries";
import { featureKeys } from "../features/queries";
import { pinKeys } from "../pins/queries";
import { autopilotKeys } from "../autopilots/queries";
import { runtimeKeys } from "../runtimes/queries";
import {
  agentTaskSnapshotKeys,
  agentActivityKeys,
  agentRunCountsKeys,
  agentTasksKeys,
} from "../agents/queries";
import { githubKeys } from "../github/queries";
import {
  onIssueCreated,
  onIssueUpdated,
  onIssueDeleted,
  onIssueLabelsChanged,
  onIssueMetadataChanged,
} from "../issues/ws-updaters";
import { onInboxNew, onInboxInvalidate, onInboxIssueStatusChanged, onInboxIssueDeleted } from "../inbox/ws-updaters";
import { inboxKeys } from "../inbox/queries";
import { workspaceKeys, workspaceListOptions } from "../workspace/queries";
import type { Workspace } from "../types/workspace";
import { resolvePostAuthDestination } from "../paths";
import type {
  WorkspaceDeletedPayload,
  WorkspaceUpdatedPayload,
  IssueUpdatedPayload,
  IssueCreatedPayload,
  IssueDeletedPayload,
  IssueLabelsChangedPayload,
  IssueMetadataChangedPayload,
  InboxNewPayload,
  CommentCreatedPayload,
  CommentUpdatedPayload,
  CommentDeletedPayload,
  CommentResolvedPayload,
  CommentUnresolvedPayload,
  ActivityCreatedPayload,
  TaskMessagePayload,
} from "../types";

const logger = createLogger("realtime-sync");

/**
 * Apply a workspace:updated event to the cache. Always refreshes the
 * workspace list. If the incoming `issue_prefix` differs from what's
 * currently cached, also invalidates issueKeys.all for that workspace,
 * since every issue's rendered identifier (`MUL-123`) is recomputed from
 * the workspace prefix at read time. Without this, the UI keeps showing
 * the old `OLD-N` keys until the next hard refresh.
 *
 * If the workspace isn't in the cached list (first observation), we
 * conservatively invalidate — the prefix is effectively "new" relative to
 * what's cached, so any issues already loaded under the old prefix would
 * be stale anyway.
 */
export function applyWorkspaceUpdatedToCache(
  qc: QueryClient,
  payload: WorkspaceUpdatedPayload,
): void {
  const next = payload.workspace;
  if (next?.id) {
    const cached =
      qc
        .getQueryData<Workspace[]>(workspaceKeys.list())
        ?.find((w) => w.id === next.id) ?? null;
    if (!cached || cached.issue_prefix !== next.issue_prefix) {
      qc.invalidateQueries({ queryKey: issueKeys.all(next.id) });
    }
  }
  qc.invalidateQueries({ queryKey: workspaceKeys.list() });
}

/**
 * Invalidates all workspace-scoped queries. Used after reconnect and when a
 * new WSClient instance is detected (workspace switch) to recover events
 * missed while disconnected.
 */
function invalidateWorkspaceScopedQueries(qc: QueryClient): void {
  const wsId = getCurrentWsId();
  if (wsId) {
    qc.invalidateQueries({ queryKey: issueKeys.all(wsId) });
    qc.invalidateQueries({ queryKey: inboxKeys.all(wsId) });
    qc.invalidateQueries({ queryKey: workspaceKeys.agents(wsId) });
    qc.invalidateQueries({ queryKey: workspaceKeys.members(wsId) });
    qc.invalidateQueries({ queryKey: workspaceKeys.skills(wsId) });
    qc.invalidateQueries({ queryKey: featureKeys.all(wsId) });
    qc.invalidateQueries({ queryKey: runtimeKeys.all(wsId) });
    qc.invalidateQueries({ queryKey: autopilotKeys.all(wsId) });
    qc.invalidateQueries({ queryKey: agentTaskSnapshotKeys.all(wsId) });
    qc.invalidateQueries({ queryKey: agentActivityKeys.all(wsId) });
    qc.invalidateQueries({ queryKey: agentRunCountsKeys.all(wsId) });
  }
  qc.invalidateQueries({ queryKey: workspaceKeys.list() });
}

export interface RealtimeSyncStores {
  authStore: UseBoundStore<StoreApi<AuthState>>;
}

/**
 * Centralized WS -> store sync. Called once from WSProvider.
 *
 * Uses the "WS as invalidation signal + refetch" pattern:
 * - onAny handler extracts event prefix and calls the matching store refresh
 * - Debounce per-prefix prevents rapid-fire refetches (e.g. bulk issue updates)
 * - Precise handlers only for side effects (toast, navigation, self-check)
 *
 * Per-issue events (comments, activity, reactions, subscribers) are handled
 * both here (invalidation fallback) and by per-page useWSEvent hooks (granular
 * updates). Daemon register events invalidate runtimes globally; heartbeats
 * are skipped to avoid excessive refetches.
 *
 * @param ws - WebSocket client instance (null when not yet connected)
 * @param stores - Platform-created Zustand store instances for auth and workspace
 * @param onToast - Optional callback for showing toast messages (platform-specific)
 */
export function useRealtimeSync(
  ws: WSClient | null,
  stores: RealtimeSyncStores,
  onToast?: (message: string, type?: "info" | "error") => void,
) {
  const { authStore } = stores;
  const qc = useQueryClient();

  // Main sync: onAny -> refreshMap with debounce
  useEffect(() => {
    if (!ws) return;

    const refreshMap: Record<string, () => void> = {
      inbox: () => {
        const wsId = getCurrentWsId();
        if (wsId) onInboxInvalidate(qc, wsId);
      },
      agent: () => {
        const wsId = getCurrentWsId();
        if (wsId) {
          qc.invalidateQueries({ queryKey: workspaceKeys.agents(wsId) });
        }
      },
      member: () => {
        const wsId = getCurrentWsId();
        if (wsId) qc.invalidateQueries({ queryKey: workspaceKeys.members(wsId) });
      },
      // workspace:updated is handled by the specific handler below
      // (compares prefixes to decide whether to also invalidate issues).
      // This generic fallback still fires for workspace:deleted (paired
      // with the specific navigation handler) and any future workspace:*
      // events without dedicated handlers.
      workspace: () => {
        qc.invalidateQueries({ queryKey: workspaceKeys.list() });
      },
      skill: () => {
        const wsId = getCurrentWsId();
        if (wsId) qc.invalidateQueries({ queryKey: workspaceKeys.skills(wsId) });
      },
      feature: () => {
        const wsId = getCurrentWsId();
        if (wsId) qc.invalidateQueries({ queryKey: featureKeys.all(wsId) });
      },
      label: () => {
        // label:created/updated/deleted — also refresh issues, since each
        // issue carries a denormalized snapshot of its labels (rename/recolor
        // /delete on a label needs to flush the chips on every issue showing
        // it).
        const wsId = getCurrentWsId();
        if (wsId) {
          qc.invalidateQueries({ queryKey: ["labels", wsId] });
          qc.invalidateQueries({ queryKey: issueKeys.all(wsId) });
        }
      },
      pin: () => {
        const wsId = getCurrentWsId();
        const userId = authStore.getState().user?.id;
        if (wsId && userId) qc.invalidateQueries({ queryKey: pinKeys.all(wsId, userId) });
      },
      daemon: () => {
        const wsId = getCurrentWsId();
        if (wsId) {
          qc.invalidateQueries({ queryKey: runtimeKeys.all(wsId) });
        }
      },
      autopilot: () => {
        const wsId = getCurrentWsId();
        if (wsId) qc.invalidateQueries({ queryKey: autopilotKeys.all(wsId) });
      },
      github_installation: () => {
        const wsId = getCurrentWsId();
        if (wsId) qc.invalidateQueries({ queryKey: githubKeys.installations(wsId) });
      },
      pull_request: () => {
        // PR list is keyed by issue id, not workspace, so we invalidate all
        // PR queries — the open issue detail page will refetch its own list.
        qc.invalidateQueries({ queryKey: ["github", "pull-requests"] });
      },
      // Powers the agent presence cache: any task lifecycle change
      // (dispatch / completed / failed / cancelled) refreshes the
      // workspace-wide agent-task-snapshot query so per-agent presence
      // reflects the change. task:message is NOT in this prefix path — it
      // stays in specificEvents to avoid an invalidate storm during long runs.
      task: () => {
        const wsId = getCurrentWsId();
        if (!wsId) return;
        qc.invalidateQueries({ queryKey: agentTaskSnapshotKeys.list(wsId) });
        // 30d activity series shares the same lifecycle signal — any task
        // completion / failure shifts the histogram. (Dispatch alone
        // doesn't change a completed_at-anchored series, but invalidating
        // here keeps the WS-handler shape uniform; the resulting refetch
        // is cheap.) Both the list (trailing 7d slice) and the detail
        // panel read off this single cache.
        qc.invalidateQueries({ queryKey: agentActivityKeys.last30d(wsId) });
        // 30-day run count likewise increments per task lifecycle event.
        qc.invalidateQueries({ queryKey: agentRunCountsKeys.last30d(wsId) });
        // Per-agent task list (Activity tab "Recent work"). Prefix match
        // catches every agent's list — the per-agent detail key sits
        // under agentTasks/<wsId>/<agentId>.
        qc.invalidateQueries({ queryKey: agentTasksKeys.all(wsId) });
        // Per-issue task list (issue-detail Execution log). Prefix match
        // across all issues — keeps the contract "any task: event makes
        // every list-of-tasks query stale" so cache stays fresh even
        // when the relevant component isn't currently mounted.
        qc.invalidateQueries({ queryKey: ["issues", "tasks"] });
        // Per-issue token usage card (issue-detail right rail). Same
        // shape as the tasks invalidation above — any task lifecycle
        // event shifts the aggregated usage numbers.
        qc.invalidateQueries({ queryKey: ["issues", "usage"] });
      },
    };

    const timers = new Map<string, ReturnType<typeof setTimeout>>();
    const debouncedRefresh = (prefix: string, fn: () => void) => {
      const existing = timers.get(prefix);
      if (existing) clearTimeout(existing);
      timers.set(
        prefix,
        setTimeout(() => {
          timers.delete(prefix);
          fn();
        }, 100),
      );
    };

    // Event types handled by specific handlers below -- skip generic refresh
    const specificEvents = new Set([
      "workspace:updated",
      "issue:updated", "issue:created", "issue:deleted", "issue_labels:changed", "issue_metadata:changed", "inbox:new",
      "comment:created", "comment:updated", "comment:deleted",
      "comment:resolved", "comment:unresolved",
      "activity:created",
      "daemon:heartbeat",
      // task:message stays out of the prefix path because it fires per
      // streamed message during a long run — invalidating the snapshot on
      // every message would flood the network.
      "task:message",
    ]);

    const unsubAny = ws.onAny((msg) => {
      if (specificEvents.has(msg.type)) return;
      const prefix = msg.type.split(":")[0] ?? "";
      const refresh = refreshMap[prefix];
      if (refresh) debouncedRefresh(prefix, refresh);
    });

    // --- Specific event handlers (granular cache updates) ---
    // No self-event filtering: actor_id identifies the USER, not the TAB.
    // Filtering by actor_id would block other tabs of the same user.
    // Instead, both mutations and WS handlers use dedup checks to be idempotent.

    const unsubIssueUpdated = ws.on("issue:updated", (p) => {
      const { issue } = p as IssueUpdatedPayload;
      if (!issue?.id) return;
      const wsId = getCurrentWsId();
      if (wsId) {
        onIssueUpdated(qc, wsId, issue);
        if (issue.status) {
          onInboxIssueStatusChanged(qc, wsId, issue.id, issue.status);
        }
      }
    });

    const unsubIssueCreated = ws.on("issue:created", (p) => {
      const { issue } = p as IssueCreatedPayload;
      if (!issue) return;
      const wsId = getCurrentWsId();
      if (wsId) onIssueCreated(qc, wsId, issue);
    });

    const unsubIssueDeleted = ws.on("issue:deleted", (p) => {
      const { issue_id } = p as IssueDeletedPayload;
      if (!issue_id) return;
      const wsId = getCurrentWsId();
      if (wsId) {
        onIssueDeleted(qc, wsId, issue_id);
        onInboxIssueDeleted(qc, wsId, issue_id);
      }
    });

    const unsubIssueLabelsChanged = ws.on("issue_labels:changed", (p) => {
      const { issue_id, labels } = p as IssueLabelsChangedPayload;
      if (!issue_id) return;
      const wsId = getCurrentWsId();
      if (wsId) onIssueLabelsChanged(qc, wsId, issue_id, labels ?? []);
    });

    const unsubIssueMetadataChanged = ws.on("issue_metadata:changed", (p) => {
      const { issue_id, metadata } = p as IssueMetadataChangedPayload;
      if (!issue_id) return;
      const wsId = getCurrentWsId();
      if (wsId) onIssueMetadataChanged(qc, wsId, issue_id, metadata ?? {});
    });

    const unsubInboxNew = ws.on("inbox:new", async (p) => {
      const { item } = p as InboxNewPayload;
      if (!item) return;
      const wsId = getCurrentWsId();
      if (wsId) onInboxNew(qc, wsId, item);
      // Fire a native OS notification only when the app isn't focused.
      if (typeof document !== "undefined" && document.hasFocus()) return;
      // Capture the source workspace slug at emit time. The user may switch
      // workspaces before clicking the banner (macOS Notification Center
      // holds banners), so routing must not read "current slug" at click
      // time — otherwise notifications from workspace A click through to
      // workspace B's inbox and 404.
      const slug = getCurrentSlug();
      if (!slug) return;
      const desktopAPI = (
        window as unknown as {
          desktopAPI?: {
            showNotification?: (payload: {
              slug: string;
              itemId: string;
              issueKey: string;
              title: string;
              body: string;
            }) => void;
          };
        }
      ).desktopAPI;
      // `issueKey` matches the inbox page's URL selector (issue id when the
      // item is attached to an issue, otherwise the inbox item id). `itemId`
      // is the inbox row's own id, needed to fire markInboxRead on click.
      desktopAPI?.showNotification?.({
        slug,
        itemId: item.id,
        issueKey: item.issue_id ?? item.id,
        title: item.title,
        body: item.body ?? "",
      });
    });

    // --- Timeline event handlers (global fallback) ---
    // These events are also handled granularly by useIssueTimeline when
    // IssueDetail is mounted. This global handler exists to mark the
    // timeline cache stale for issues whose IssueDetail is *not* mounted,
    // so stale data isn't served on next mount (staleTime: Infinity, set on
    // the QueryClient default, relies on this).
    //
    // `refetchType: "none"` is the load-bearing detail: without it, an
    // active IssueDetail observer would refetch the entire timeline on
    // every comment / activity / reaction event. The refetch replaces
    // every entry's reference and busts React.memo on every CommentCard
    // subtree (visible during AI streaming as a flash across all sibling
    // threads, MUL-1941). Inactive observers don't refetch either way;
    // when IssueDetail mounts later, the stale flag triggers the refetch
    // through `refetchOnMount`. Active observers stay fresh via the
    // granular setQueryData handlers in `useIssueTimeline`.
    const invalidateTimeline = (issueId: string) => {
      qc.invalidateQueries({
        queryKey: issueKeys.timeline(issueId),
        refetchType: "none",
      });
    };

    const unsubCommentCreated = ws.on("comment:created", (p) => {
      const { comment } = p as CommentCreatedPayload;
      if (comment?.issue_id) invalidateTimeline(comment.issue_id);
    });

    const unsubCommentUpdated = ws.on("comment:updated", (p) => {
      const { comment } = p as CommentUpdatedPayload;
      if (comment?.issue_id) invalidateTimeline(comment.issue_id);
    });

    const unsubCommentDeleted = ws.on("comment:deleted", (p) => {
      const { issue_id } = p as CommentDeletedPayload;
      if (issue_id) invalidateTimeline(issue_id);
    });

    const unsubCommentResolved = ws.on("comment:resolved", (p) => {
      const { comment } = p as CommentResolvedPayload;
      if (comment?.issue_id) invalidateTimeline(comment.issue_id);
    });

    const unsubCommentUnresolved = ws.on("comment:unresolved", (p) => {
      const { comment } = p as CommentUnresolvedPayload;
      if (comment?.issue_id) invalidateTimeline(comment.issue_id);
    });

    const unsubActivityCreated = ws.on("activity:created", (p) => {
      const { issue_id } = p as ActivityCreatedPayload;
      if (issue_id) invalidateTimeline(issue_id);
    });

    // --- Side-effect handlers (toast, navigation) ---

    // After the current workspace disappears (deleted or we were kicked out),
    // navigate to another workspace the user still has access to, or to the
    // create-workspace page. We use a full-page navigation: this reliably
    // tears down any in-flight queries / subscriptions tied to the dead
    // workspace without relying on framework-specific routers from here in
    // core.
    const relocateAfterWorkspaceLoss = async (lostWsId: string) => {
      const wsList = await qc.fetchQuery({
        ...workspaceListOptions(),
        staleTime: 0,
      });
      const remaining = wsList.filter((w) => w.id !== lostWsId);
      const target = resolvePostAuthDestination(remaining);
      if (typeof window !== "undefined") {
        window.location.assign(target);
      }
    };

    const unsubWsUpdated = ws.on("workspace:updated", (p) => {
      applyWorkspaceUpdatedToCache(qc, p as WorkspaceUpdatedPayload);
    });

    const unsubWsDeleted = ws.on("workspace:deleted", (p) => {
      const { workspace_id } = p as WorkspaceDeletedPayload;
      // Event payload has UUID; look up slug from cached workspace list
      // since clearWorkspaceStorage keys are namespaced by slug.
      const wsList = qc.getQueryData<{ id: string; slug: string }[]>(workspaceKeys.list()) ?? [];
      const deletedSlug = wsList.find((w) => w.id === workspace_id)?.slug;
      if (deletedSlug) clearWorkspaceStorage(defaultStorage, deletedSlug);
      if (getCurrentWsId() === workspace_id) {
        logger.warn("current workspace deleted, switching");
        onToast?.("This workspace was deleted", "info");
        relocateAfterWorkspaceLoss(workspace_id);
      }
    });

    const unsubTaskMessage = ws.on("task:message", (p) => {
      const payload = p as TaskMessagePayload;
      qc.setQueryData<TaskMessagePayload[]>(
        ["task-messages", payload.task_id],
        (old = []) => {
          if (old.some((m) => m.seq === payload.seq)) return old;
          return [...old, payload].sort((a, b) => a.seq - b.seq);
        },
      );
    });

    return () => {
      unsubAny();
      unsubIssueUpdated();
      unsubIssueCreated();
      unsubIssueDeleted();
      unsubIssueLabelsChanged();
      unsubIssueMetadataChanged();
      unsubInboxNew();
      unsubCommentCreated();
      unsubCommentUpdated();
      unsubCommentDeleted();
      unsubCommentResolved();
      unsubCommentUnresolved();
      unsubActivityCreated();
      unsubWsUpdated();
      unsubWsDeleted();
      unsubTaskMessage();
      timers.forEach(clearTimeout);
      timers.clear();
    };
  }, [ws, qc, authStore, onToast]);

  // Reconnect -> refetch all data to recover missed events
  useEffect(() => {
    if (!ws) return;

    const unsub = ws.onReconnect(async () => {
      logger.info("reconnected, refetching all data");
      try {
        invalidateWorkspaceScopedQueries(qc);
      } catch (e) {
        logger.error("reconnect refetch failed", e);
      }
    });

    return unsub;
  }, [ws, qc]);

  // New WSClient instance (workspace switch) -> invalidate workspace-scoped
  // queries to recover events missed while the previous instance was torn down.
  // Skips the initial assignment to avoid a redundant refetch on first mount.
  const wsInstanceRef = useRef<WSClient | null>(null);
  useEffect(() => {
    if (!ws) return;
    if (wsInstanceRef.current === null) {
      // First non-null instance — store and skip invalidation.
      wsInstanceRef.current = ws;
      return;
    }
    if (wsInstanceRef.current === ws) return;
    wsInstanceRef.current = ws;

    logger.info("new WSClient instance detected, invalidating workspace queries");
    invalidateWorkspaceScopedQueries(qc);
  }, [ws, qc]);
}
