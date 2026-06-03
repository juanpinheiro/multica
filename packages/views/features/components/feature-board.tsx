"use client";

import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import type { Issue, IssueStatus } from "@multica/core/types";
import { featureIssueListOptions } from "@multica/core/issues/queries";
import { BOARD_STATUSES, STATUS_CONFIG } from "@multica/core/issues/config";
import { useWorkspaceId } from "@multica/core/hooks";
import { useWorkspacePaths } from "@multica/core/paths";
import { cn } from "@multica/ui/lib/utils";
import { AppLink } from "../../navigation";
import { useIssueLiveState, BoardCardLiveLayer } from "../../issues/components/board-card";

function FeatureIssueCard({ issue, featureId }: { issue: Issue; featureId: string }) {
  const wsId = useWorkspaceId();
  const p = useWorkspacePaths();
  const { liveness, counters } = useIssueLiveState(issue.id, wsId, issue.status);

  return (
    <AppLink href={p.initiativeIssue(featureId, issue.id)} className="block">
      <div
        className={cn(
          "rounded-lg border-[0.5px] border-border bg-card px-2.5 py-2 hover:bg-accent transition-colors",
          liveness?.active && "ring-1 ring-brand/40 border-brand/30",
        )}
      >
        <p className="text-xs text-muted-foreground">{issue.identifier}</p>
        <p className="mt-0.5 text-xs font-medium leading-snug line-clamp-2">{issue.title}</p>
        <BoardCardLiveLayer liveness={liveness} counters={counters} />
      </div>
    </AppLink>
  );
}

function FeatureBoardColumn({
  status,
  issues,
  featureId,
}: {
  status: IssueStatus;
  issues: Issue[];
  featureId: string;
}) {
  const cfg = STATUS_CONFIG[status];
  return (
    <div data-testid={`board-column-${status}`} className="flex w-52 shrink-0 flex-col gap-1.5">
      <div className="flex items-center gap-1.5 px-1">
        <div className={cn("h-2 w-2 rounded-full", cfg.dividerColor)} />
        <span className="text-xs font-medium text-muted-foreground">{cfg.label}</span>
        {issues.length > 0 && (
          <span className="ml-auto text-xs text-muted-foreground/60 tabular-nums">
            {issues.length}
          </span>
        )}
      </div>
      <div className="space-y-1.5">
        {issues.map((issue) => (
          <FeatureIssueCard key={issue.id} issue={issue} featureId={featureId} />
        ))}
      </div>
    </div>
  );
}

export function FeatureBoardView({ featureId }: { featureId: string }) {
  const wsId = useWorkspaceId();
  const { data: issues = [] } = useQuery(featureIssueListOptions(wsId, featureId));

  const byStatus = useMemo(() => {
    const map = new Map<IssueStatus, Issue[]>();
    for (const s of BOARD_STATUSES) map.set(s, []);
    for (const issue of issues) {
      map.get(issue.status)?.push(issue);
    }
    return map;
  }, [issues]);

  return (
    <div className="flex gap-3 overflow-x-auto pb-2">
      {BOARD_STATUSES.map((status) => (
        <FeatureBoardColumn
          key={status}
          status={status}
          issues={byStatus.get(status) ?? []}
          featureId={featureId}
        />
      ))}
    </div>
  );
}
