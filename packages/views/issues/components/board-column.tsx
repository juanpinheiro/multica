"use client";

import { memo, useMemo, type ReactNode } from "react";
import { EyeOff, MoreHorizontal, UserMinus } from "lucide-react";
import { useDroppable } from "@dnd-kit/core";
import { SortableContext, verticalListSortingStrategy } from "@dnd-kit/sortable";
import type { Issue, IssueAssigneeType, IssueStatus } from "@multica/core/types";
import { Button } from "@multica/ui/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from "@multica/ui/components/ui/dropdown-menu";
import { STATUS_CONFIG } from "@multica/core/issues/config";
import { useViewStoreApi } from "@multica/core/issues/stores/view-store-context";
import { useWorkspaceId } from "@multica/core/hooks";
import { useQuery } from "@tanstack/react-query";
import { agentTaskSnapshotOptions } from "@multica/core/agents";
import { StatusHeading } from "./status-heading";
import { DraggableBoardCard } from "./board-card";
import type { ChildProgress } from "./list-row";
import { useT } from "../../i18n";
import { ActorAvatar } from "../../common/actor-avatar";

function useLiveIssueCount(wsId: string, issueIds: string[]): number {
  const { data: snapshot = [] } = useQuery(agentTaskSnapshotOptions(wsId));
  const issueIdSet = useMemo(() => new Set(issueIds), [issueIds]);
  return useMemo(() => {
    const live = new Set<string>();
    for (const t of snapshot) {
      if (issueIdSet.has(t.issue_id) && (t.status === "running" || t.status === "waiting_local_directory")) {
        live.add(t.issue_id);
      }
    }
    return live.size;
  }, [snapshot, issueIdSet]);
}

// Insertion-position prediction intentionally omitted. The server's
// ORDER BY uses PostgreSQL's en_US.utf8 collation (glibc), which
// cannot be faithfully replicated in JavaScript (ICU/V8). Showing an
// inaccurate indicator is worse than showing none.

export const BOARD_COL_WIDTH = 280;
export const BOARD_CARD_WIDTH = BOARD_COL_WIDTH - 16 - 8; // col(280) - col p-2(16) - droppable p-1(8)

export interface BoardColumnGroup {
  id: string;
  title: string;
  status?: IssueStatus;
  assigneeType?: IssueAssigneeType | null;
  assigneeId?: string | null;
  totalCount?: number;
  createData?: Record<string, unknown>;
}

export const BoardColumn = memo(function BoardColumn({
  group,
  issueIds,
  issueMap,
  childProgressMap,
  totalCount,
  footer,
  sortLabel,
}: {
  group: BoardColumnGroup;
  issueIds: string[];
  issueMap: Map<string, Issue>;
  childProgressMap?: Map<string, ChildProgress>;
  totalCount?: number;
  footer?: ReactNode;
  sortLabel?: string | null;
}) {
  const status = group.status;
  const cfg = status ? STATUS_CONFIG[status] : null;
  const { setNodeRef, isOver } = useDroppable({ id: group.id });
  const viewStoreApi = useViewStoreApi();
  const { t } = useT("issues");
  const wsId = useWorkspaceId();
  const liveCount = useLiveIssueCount(wsId, issueIds);

  // Resolve IDs to Issue objects, preserving parent-provided order
  const resolvedIssues = useMemo(
    () =>
      issueIds.flatMap((id) => {
        const issue = issueMap.get(id);
        return issue ? [issue] : [];
      }),
    [issueIds, issueMap],
  );

  return (
    <div style={{ width: BOARD_COL_WIDTH }} className={`flex shrink-0 flex-col rounded-xl ${cfg?.columnBg ?? "bg-muted/40"} p-2`}>
      <div className="mb-2 flex items-center justify-between px-1.5">
        <BoardGroupHeading group={group} count={totalCount ?? issueIds.length} liveCount={status === "in_progress" ? liveCount : undefined} />

        {status && (
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <Button variant="ghost" size="icon-sm" className="rounded-full text-muted-foreground">
                  <MoreHorizontal className="size-3.5" />
                </Button>
              }
            />
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => viewStoreApi.getState().hideStatus(status)}>
                <EyeOff className="size-3.5" />
                {t(($) => $.board.hide_column)}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>
      <div
        ref={setNodeRef}
        className={`relative min-h-[200px] flex-1 space-y-2 overflow-y-auto rounded-lg p-1 transition-colors ${
          isOver && sortLabel
            ? "ring-2 ring-brand/25 bg-accent/15"
            : isOver
              ? "bg-accent/60"
              : ""
        }`}
      >
        {isOver && sortLabel && (
          <div className="pointer-events-none absolute inset-0 z-10 flex items-center justify-center rounded-lg bg-background/40">
            <span className="rounded-md bg-popover px-2.5 py-1 text-xs font-medium text-popover-foreground shadow-sm border border-border">
              {sortLabel}
            </span>
          </div>
        )}
        <SortableContext items={issueIds} strategy={verticalListSortingStrategy}>
          {resolvedIssues.map((issue) => (
            <DraggableBoardCard key={issue.id} issue={issue} childProgress={childProgressMap?.get(issue.id)} disableSorting={!!sortLabel} />
          ))}
        </SortableContext>
        {issueIds.length === 0 && (
          <p className="py-8 text-center text-xs text-muted-foreground">
            {t(($) => $.board.empty_column)}
          </p>
        )}
        {footer}
      </div>
    </div>
  );
});

function BoardGroupHeading({
  group,
  count,
  liveCount,
}: {
  group: BoardColumnGroup;
  count: number;
  liveCount?: number;
}) {
  if (group.status) {
    return <StatusHeading status={group.status} count={count} liveCount={liveCount} />;
  }

  const actorIcon =
    group.assigneeType && group.assigneeId ? (
      <ActorAvatar
        actorType={group.assigneeType}
        actorId={group.assigneeId}
        size={18}
        showStatusDot={group.assigneeType === "agent"}
      />
    ) : (
      <span className="flex size-[18px] shrink-0 items-center justify-center rounded-full bg-background text-muted-foreground">
        <UserMinus className="size-3.5" />
      </span>
    );

  return (
    <div className="flex min-w-0 items-center gap-2">
      {actorIcon}
      <span className="truncate text-sm font-medium" title={group.title}>
        {group.title}
      </span>
      <span className="shrink-0 rounded-full bg-background px-1.5 py-0.5 text-[11px] font-medium tabular-nums text-muted-foreground">
        {count}
      </span>
    </div>
  );
}
