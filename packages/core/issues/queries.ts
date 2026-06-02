import { keepPreviousData, queryOptions } from "@tanstack/react-query";
import { api } from "../api";
import type {
  GroupedIssuesResponse,
  IssueStatus,
  ListGroupedIssuesParams,
  ListIssuesParams,
  ListIssuesCache,
} from "../types";
import { BOARD_STATUSES } from "./config";

export interface IssueSortParam {
  sort_by?: ListIssuesParams["sort_by"];
  sort_direction?: ListIssuesParams["sort_direction"];
}

export const issueKeys = {
  all: (wsId: string) => ["issues", wsId] as const,
  /** PREFIX for invalidation — no sort. */
  list: (wsId: string) => [...issueKeys.all(wsId), "list"] as const,
  /** FULL KEY for queryOptions — includes sort. */
  listSorted: (wsId: string, sort?: IssueSortParam) =>
    [...issueKeys.list(wsId), sort ?? {}] as const,
  assigneeGroupsAll: (wsId: string) =>
    [...issueKeys.all(wsId), "assignee-groups"] as const,
  assigneeGroups: (wsId: string, filter: AssigneeGroupedIssuesFilter) =>
    [...issueKeys.assigneeGroupsAll(wsId), filter] as const,
  /** All Feature Gantt queries — prefix-match key for cross-project invalidation. */
  featureGanttAll: (wsId: string) =>
    [...issueKeys.all(wsId), "feature-gantt"] as const,
  /**
   * Per-project Gantt issue list (scheduled-only). Uses its own cache key
   * rather than reusing the bucketed `myList` cache so WS handlers and
   * cache helpers don't have to special-case a non-bucketed shape under
   * the `my` prefix.
   */
  featureGantt: (wsId: string, featureId: string) =>
    [...issueKeys.featureGanttAll(wsId), featureId] as const,
  detail: (wsId: string, id: string) =>
    [...issueKeys.all(wsId), "detail", id] as const,
  children: (wsId: string, id: string) =>
    [...issueKeys.all(wsId), "children", id] as const,
  childProgress: (wsId: string) =>
    [...issueKeys.all(wsId), "child-progress"] as const,
  /** Full-issue timeline (single TanStack Query, no cursor). */
  timeline: (issueId: string) =>
    ["issues", "timeline", issueId] as const,
  usage: (issueId: string) => ["issues", "usage", issueId] as const,
  /** Issue-level attachments — used by the description editor so its
   *  inline file-card / image NodeViews can re-sign download URLs at
   *  click time. */
  attachments: (issueId: string) => ["issues", "attachments", issueId] as const,
  /** Per-issue task list (issue-detail Execution log section). */
  tasks: (issueId: string) => ["issues", "tasks", issueId] as const,
  /** Prefix-match key for invalidating tasks across all issues — used by
   *  the global WS task: prefix path so any task lifecycle event refreshes
   *  every per-issue list, regardless of which issue is currently mounted. */
  tasksAll: () => ["issues", "tasks"] as const,
};

export type AssigneeGroupedIssuesFilter = Omit<
  ListGroupedIssuesParams,
  "group_by" | "limit" | "offset" | "group_assignee_type" | "group_assignee_id"
>;

/** Page size per status column. */
export const ISSUE_PAGE_SIZE = 50;

/** Statuses the issues/my-issues pages paginate. Cancelled is intentionally excluded — it has never been surfaced in the list/board views. */
export const PAGINATED_STATUSES: readonly IssueStatus[] = BOARD_STATUSES;

/** Flatten a bucketed response to a single Issue[] for consumers that want the whole list. */
export function flattenIssueBuckets(data: ListIssuesCache) {
  const out = [];
  for (const status of PAGINATED_STATUSES) {
    const bucket = data.byStatus[status];
    if (bucket) out.push(...bucket.issues);
  }
  return out;
}

async function fetchFirstPages(filter: Pick<ListIssuesParams, "assignee_id" | "assignee_ids" | "creator_id" | "feature_id"> = {}, sort?: IssueSortParam): Promise<ListIssuesCache> {
  const responses = await Promise.all(
    PAGINATED_STATUSES.map((status) =>
      api.listIssues({ status, limit: ISSUE_PAGE_SIZE, offset: 0, ...sort, ...filter }),
    ),
  );
  const byStatus: ListIssuesCache["byStatus"] = {};
  PAGINATED_STATUSES.forEach((status, i) => {
    const res = responses[i]!;
    byStatus[status] = { issues: res.issues, total: res.total };
  });
  return { byStatus };
}

/**
 * CACHE SHAPE NOTE: The raw cache stores {@link ListIssuesCache} (buckets keyed
 * by status, each with `{ issues, total }`), and `select` flattens it to
 * `Issue[]` for consumers. Mutations and ws-updaters must use
 * `setQueryData<ListIssuesCache>(...)` and preserve the byStatus shape.
 *
 * Fetches the first page of each paginated status in parallel. Use
 * {@link useLoadMoreByStatus} to paginate a specific status into the cache.
 */
export type ActorIssueFilter = {
  assignee_id?: string;
  creator_id?: string;
};

export function actorIssueListOptions(
  wsId: string,
  scope: string,
  filter: ActorIssueFilter,
  sort?: IssueSortParam,
) {
  return queryOptions({
    queryKey: [...issueKeys.list(wsId), "actor", scope, filter, sort ?? {}] as const,
    queryFn: () => fetchFirstPages(filter, sort),
    select: flattenIssueBuckets,
    placeholderData: keepPreviousData,
  });
}

export function issueListOptions(wsId: string, sort?: IssueSortParam) {
  return queryOptions({
    queryKey: issueKeys.listSorted(wsId, sort),
    queryFn: () => fetchFirstPages({}, sort),
    select: flattenIssueBuckets,
    placeholderData: keepPreviousData,
  });
}

export function featureIssueListOptions(wsId: string, featureId: string) {
  return queryOptions({
    queryKey: [...issueKeys.list(wsId), "feature", featureId] as const,
    queryFn: () => fetchFirstPages({ feature_id: featureId }),
    select: flattenIssueBuckets,
    placeholderData: keepPreviousData,
  });
}

export function issueAssigneeGroupsOptions(
  wsId: string,
  filter: AssigneeGroupedIssuesFilter,
  sort?: IssueSortParam,
) {
  return queryOptions<GroupedIssuesResponse>({
    queryKey: issueKeys.assigneeGroups(wsId, { ...filter, ...sort }),
    queryFn: () =>
      api.listGroupedIssues({
        group_by: "assignee",
        limit: ISSUE_PAGE_SIZE,
        offset: 0,
        ...sort,
        ...filter,
      }),
    placeholderData: keepPreviousData,
  });
}

/**
 * Page size for the scheduled-issue fetch. The Gantt view always pulls every
 * scheduled issue (no client pagination), so this is just the chunk size we
 * use to walk the server's `(limit, offset)` window until we hit `total`.
 */
export const FEATURE_GANTT_PAGE_LIMIT = 500;

/**
 * Paranoia cap on the loop in {@link fetchFeatureGanttIssues}. Real projects
 * shouldn't come close to this — a single project carrying 50k scheduled
 * issues is already a product problem, not a Gantt-rendering one — but the
 * guard prevents a buggy server `total` from spinning the loop forever.
 */
export const FEATURE_GANTT_MAX_ISSUES = 10_000;

async function fetchFeatureGanttIssues(featureId: string) {
  const issues = [];
  let offset = 0;
  while (offset < FEATURE_GANTT_MAX_ISSUES) {
    const res = await api.listIssues({
      feature_id: featureId,
      scheduled: true,
      limit: FEATURE_GANTT_PAGE_LIMIT,
      offset,
    });
    issues.push(...res.issues);
    if (res.issues.length < FEATURE_GANTT_PAGE_LIMIT) break;
    if (issues.length >= res.total) break;
    offset += FEATURE_GANTT_PAGE_LIMIT;
  }
  return issues;
}

/**
 * One-shot fetch of every scheduled issue (`start_date` or `due_date` set)
 * for a feature. The Feature Gantt view consumes this directly — no status
 * bucketing, no client-side pagination, no Load-all affordance — because
 * the scheduled subset is bounded enough to come back in a small handful of
 * requests.
 *
 * Backed by `GET /api/issues?scheduled=true&feature_id=…`; the SQL filter
 * mirrors the same `(start_date IS NOT NULL OR due_date IS NOT NULL)`
 * predicate the Gantt view applies on the client. Pages are walked until
 * `total` is reached so an oversized project can't silently lose bars past
 * the first page.
 */
export function featureGanttIssuesOptions(wsId: string, featureId: string) {
  return queryOptions({
    queryKey: issueKeys.featureGantt(wsId, featureId),
    queryFn: () => fetchFeatureGanttIssues(featureId),
  });
}

export function issueDetailOptions(wsId: string, id: string) {
  return queryOptions({
    queryKey: issueKeys.detail(wsId, id),
    queryFn: () => api.getIssue(id),
  });
}

export function childIssueProgressOptions(wsId: string) {
  return queryOptions({
    queryKey: issueKeys.childProgress(wsId),
    queryFn: () => api.getChildIssueProgress(),
    select: (data) => {
      const map = new Map<string, { done: number; total: number }>();
      for (const entry of data.progress) {
        map.set(entry.parent_issue_id, { done: entry.done, total: entry.total });
      }
      return map;
    },
  });
}

export function childIssuesOptions(wsId: string, id: string) {
  return queryOptions({
    queryKey: issueKeys.children(wsId, id),
    queryFn: () => api.listChildIssues(id).then((r) => r.issues),
  });
}

/**
 * Single-fetch timeline options. The endpoint returns the full ordered set of
 * comments + activities for an issue (server caps at 2000 as a safety net).
 * Cursor pagination was removed in #1929 — at observed data sizes (p99 ~30
 * entries per issue) it added complexity without a UX win and broke reply
 * threads at page boundaries.
 */
export function issueTimelineOptions(issueId: string) {
  return queryOptions({
    queryKey: issueKeys.timeline(issueId),
    queryFn: () => api.listTimeline(issueId),
  });
}

export function issueUsageOptions(issueId: string) {
  return queryOptions({
    queryKey: issueKeys.usage(issueId),
    queryFn: () => api.getIssueUsage(issueId),
  });
}

// Backs the description editor's fresh-sign download flow: NodeViews resolve
// an attachment id by matching the markdown URL against this list. The list
// is workspace-private metadata and lives on the same cache lifetime as the
// rest of the issue detail surface.
export function issueAttachmentsOptions(issueId: string) {
  return queryOptions({
    queryKey: issueKeys.attachments(issueId),
    queryFn: () => api.listAttachments(issueId),
  });
}
