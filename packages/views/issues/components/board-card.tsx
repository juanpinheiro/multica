"use client";

import { useCallback, useEffect, useMemo, useState, memo } from "react";
import { AppLink } from "../../navigation";
import { useSortable, defaultAnimateLayoutChanges } from "@dnd-kit/sortable";
import type { AnimateLayoutChanges } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { toast } from "sonner";
import type { Issue, AgentTask, UpdateIssueRequest, IssueStatus } from "@multica/core/types";
import { CalendarClock, CalendarDays } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { ActorAvatar } from "../../common/actor-avatar";
import { useUpdateIssue } from "@multica/core/issues/mutations";
import { useWorkspacePaths } from "@multica/core/paths";
import { useWorkspaceId } from "@multica/core/hooks";
import { useActorName } from "@multica/core/workspace/hooks";
import { useTimeAgo } from "../../i18n";
import { featureListOptions } from "@multica/core/features/queries";
import { issueDetailOptions } from "@multica/core/issues/queries";
import { agentTaskSnapshotOptions } from "@multica/core/agents";
import { taskMessagesOptions } from "@multica/core/chat/queries";
import {
  deriveLiveness,
  deriveActivityCounters,
  type Liveness,
  type LivenessPhase,
  type ActivityCounters,
} from "@multica/core/tasks";
import { cn } from "@multica/ui/lib/utils";
import { FeatureIcon } from "../../features/components/feature-icon";
import { PriorityIcon } from "./priority-icon";
import { PriorityPicker, AssigneePicker, StartDatePicker, DueDatePicker } from "./pickers";
import { useViewStore } from "@multica/core/issues/stores/view-store-context";
import { ProgressRing } from "./progress-ring";
import type { ChildProgress } from "./list-row";
import { IssueActionsContextMenu } from "../actions";
import { LabelChip } from "../../labels/label-chip";
import { IssueAgentActivityIndicator } from "./issue-agent-activity-indicator";
import { useT } from "../../i18n";

const UUID_RE = /[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/i;

function parseHolderTaskId(waitReason: string | null | undefined): string | null {
  if (!waitReason) return null;
  return waitReason.match(UUID_RE)?.[0] ?? null;
}

function pickLiveTask(tasks: AgentTask[], issueId: string): AgentTask | null {
  let waiting: AgentTask | null = null;
  for (const t of tasks) {
    if (t.issue_id !== issueId) continue;
    if (t.status === "running") return t;
    if (t.status === "waiting_local_directory" && !waiting) waiting = t;
  }
  return waiting;
}

// Re-renders the caller ~once per second so heartbeat freshness and elapsed
// time advance between server events. Inert when disabled (no live task) so a
// dense board doesn't tick every card every second.
function useNow(enabled: boolean, intervalMs = 1000): number {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (!enabled) return;
    const id = setInterval(() => setNow(Date.now()), intervalMs);
    return () => clearInterval(id);
  }, [enabled, intervalMs]);
  return now;
}

function useHolderIssueKey(task: AgentTask | null, snapshot: AgentTask[], wsId: string): string | null {
  const holderTaskId = useMemo(
    () => (task?.status === "waiting_local_directory" ? parseHolderTaskId(task.wait_reason) : null),
    [task],
  );
  const holderIssueId = useMemo(() => {
    if (!holderTaskId) return null;
    return snapshot.find((t) => t.id === holderTaskId)?.issue_id || null;
  }, [holderTaskId, snapshot]);
  const { data: holderIssue } = useQuery({
    ...issueDetailOptions(wsId, holderIssueId ?? ""),
    enabled: !!holderIssueId,
  });
  return holderIssue?.identifier ?? null;
}

function useIssueLiveState(
  issueId: string,
  wsId: string,
  issueStatus?: IssueStatus,
): { liveness: Liveness | null; counters: ActivityCounters | null } {
  const { data: snapshot = [] } = useQuery(agentTaskSnapshotOptions(wsId));
  const task = useMemo(() => pickLiveTask(snapshot, issueId), [snapshot, issueId]);
  const { data: messages = [] } = useQuery(taskMessagesOptions(task?.id ?? ""));
  const holderKey = useHolderIssueKey(task, snapshot, wsId);
  const now = useNow(task !== null);
  const liveness = useMemo(
    () => (task ? deriveLiveness(task, now, { issueStatus, holderKey }) : null),
    [task, now, issueStatus, holderKey],
  );
  const counters = useMemo(
    () => (task ? deriveActivityCounters(messages, task.started_at, now) : null),
    [messages, task, now],
  );
  return { liveness, counters };
}

const PHASES: LivenessPhase[] = ["claim", "run", "push", "pr"];

function formatElapsed(ms: number): string {
  const secs = Math.floor(ms / 1000);
  if (secs < 60) return `${secs}s`;
  const mins = Math.floor(secs / 60);
  if (mins < 60) return `${mins}m`;
  const hrs = Math.floor(mins / 60);
  return `${hrs}h ${mins % 60}m`;
}

export function BoardCardLiveLayer({
  liveness,
  counters = null,
}: {
  liveness: Liveness | null;
  counters?: ActivityCounters | null;
}) {
  if (!liveness?.active) return null;

  const currentIdx = PHASES.indexOf(liveness.phase);

  return (
    <div className="mt-2 space-y-1.5">
      <div data-testid="phase-stepper" className="flex items-center gap-1">
        {PHASES.map((phase, idx) => {
          const isActive = phase === liveness.phase;
          const isReached = idx < currentIdx;
          return (
            <span key={phase} className="flex items-center gap-1">
              <span
                data-testid={`phase-step-${phase}`}
                aria-current={isActive ? "step" : undefined}
                className={cn(
                  "text-[10px] leading-none",
                  isActive && "text-brand font-medium",
                  isReached && "text-muted-foreground/60",
                  !isActive && !isReached && "text-muted-foreground/35",
                )}
              >
                {phase}
              </span>
              {idx < PHASES.length - 1 && (
                <span aria-hidden="true" className="text-[9px] text-muted-foreground/25">›</span>
              )}
            </span>
          );
        })}
      </div>
      <div
        data-testid="task-progress-shimmer"
        className="relative h-0.5 overflow-hidden rounded-full bg-muted"
      >
        <span className="absolute inset-y-0 w-1/3 rounded-full bg-brand animate-task-progress-sweep" />
      </div>
      {liveness.waiting ? (
        <WaitingBlock waiting={liveness.waiting} />
      ) : (
        <Heartbeat heartbeat={liveness.heartbeat} quietMs={liveness.quietMs} />
      )}
      {counters && (
        <div
          data-testid="activity-counters"
          className="flex items-center gap-1.5 text-[10px] leading-none text-muted-foreground tabular-nums"
        >
          <span>{formatElapsed(counters.elapsedMs)}</span>
          {counters.activityCount > 0 && (
            <>
              <span aria-hidden="true">·</span>
              <span>
                {counters.activityCount}{" "}
                {counters.activityCount === 1 ? "action" : "actions"}
              </span>
            </>
          )}
        </div>
      )}
    </div>
  );
}

function WaitingBlock({ waiting }: { waiting: NonNullable<Liveness["waiting"]> }) {
  return (
    <div
      data-testid="waiting-block"
      className="rounded-md border border-warning/25 bg-warning/10 px-2 py-1 space-y-0.5"
    >
      <div className="flex items-center gap-1.5 text-[10px] leading-none">
        <span className="size-1.5 shrink-0 rounded-full bg-warning" />
        <span className="font-medium text-warning">waiting</span>
        {waiting.holderKey && (
          <span className="text-warning/80">· {waiting.holderKey}</span>
        )}
      </div>
      <p className="pl-3 text-[10px] leading-snug text-warning/70 truncate">{waiting.reason}</p>
    </div>
  );
}

function Heartbeat({ heartbeat, quietMs }: { heartbeat: Liveness["heartbeat"]; quietMs: number }) {
  const quiet = heartbeat === "quiet";
  return (
    <div data-testid="heartbeat" className="flex items-center gap-1.5 text-[10px] leading-none">
      <span
        className={cn(
          "size-1.5 rounded-full",
          quiet ? "bg-warning" : "bg-brand animate-pulse",
        )}
      />
      <span className={cn("tabular-nums", quiet ? "text-warning" : "text-muted-foreground")}>
        {quiet ? `quiet ${Math.floor(quietMs / 1000)}s` : "now"}
      </span>
    </div>
  );
}

function formatDate(date: string): string {
  return new Date(date).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
  });
}

function descriptionPreview(markdown: string): string {
  return markdown
    .replace(/!file\[[^\]]*\]\([^)]*\)/g, "")
    .replace(/!\[[^\]]*\]\([^)]*\)/g, "")
    .replace(/\[([^\]]+)\]\([^)]+\)/g, "$1")
    .replace(/[*_`~]+/g, "")
    .replace(/^[\s>#]+/gm, "")
    .replace(/\s+/g, " ")
    .trim();
}

/** Stops event from bubbling to Link/drag handlers */
function PickerWrapper({ children, className }: { children: React.ReactNode; className?: string }) {
  const stop = (e: React.SyntheticEvent) => {
    e.stopPropagation();
    e.preventDefault();
  };
  return (
    <div onClick={stop} onMouseDown={stop} onPointerDown={stop} className={className}>
      {children}
    </div>
  );
}

export const BoardCardContent = memo(function BoardCardContent({
  issue,
  editable = false,
  childProgress,
}: {
  issue: Issue;
  editable?: boolean;
  childProgress?: ChildProgress;
}) {
  const { t } = useT("issues");
  const timeAgo = useTimeAgo();
  const storeProperties = useViewStore((s) => s.cardProperties);
  const wsId = useWorkspaceId();
  const { liveness, counters } = useIssueLiveState(issue.id, wsId, issue.status);
  const { data: features = [] } = useQuery({
    ...featureListOptions(wsId),
    enabled: storeProperties.feature && !!issue.feature_id,
  });
  const feature = issue.feature_id ? features.find((p) => p.id === issue.feature_id) : undefined;
  const labels = issue.labels ?? [];

  const updateIssueMutation = useUpdateIssue();
  const handleUpdate = useCallback(
    (updates: Partial<UpdateIssueRequest>) => {
      updateIssueMutation.mutate(
        { id: issue.id, ...updates },
        {
          onError: (err) =>
            toast.error(
              err instanceof Error && err.message
                ? err.message
                : t(($) => $.card.update_failed),
            ),
        },
      );
    },
    [issue.id, updateIssueMutation, t],
  );

  const showPriority = storeProperties.priority;
  const showDescription = storeProperties.description && issue.description;
  const showAssigneeSection = storeProperties.assignee;
  const hasAssignee = !!issue.assignee_type && !!issue.assignee_id;
  const showStartDate = storeProperties.startDate && issue.start_date;
  const showDueDate = storeProperties.dueDate && issue.due_date;
  const showFeature = storeProperties.feature && feature;
  const showChildProgress = storeProperties.childProgress && childProgress;
  const showLabels = storeProperties.labels && labels.length > 0;

  const showAssigneeName = showAssigneeSection && hasAssignee && !showStartDate && !showDueDate;
  const showUpdatedHint = showAssigneeName && !showChildProgress;
  const { getActorName } = useActorName();
  const assigneeName =
    showAssigneeName && issue.assignee_type && issue.assignee_id
      ? getActorName(issue.assignee_type, issue.assignee_id)
      : null;

  const priorityLabel = t(($) => $.priority[issue.priority]);
  const priorityIconNode = showPriority ? (
    editable ? (
      <PickerWrapper>
        <PriorityPicker
          priority={issue.priority}
          onUpdate={handleUpdate}
          triggerRender={
            <button
              type="button"
              aria-label={priorityLabel}
              className="inline-flex items-center justify-center rounded hover:bg-muted/60"
            >
              <PriorityIcon priority={issue.priority} />
            </button>
          }
        />
      </PickerWrapper>
    ) : (
      <span aria-label={priorityLabel} className="inline-flex items-center justify-center">
        <PriorityIcon priority={issue.priority} />
      </span>
    )
  ) : null;

  // The parent row gives this container the leftover space; min-w-0 and
  // max-w-full make the nested picker trigger respect that limit.
  const assigneeContainerClass = assigneeName
    ? "flex min-w-0 max-w-full items-center"
    : "inline-flex items-center";

  const assigneeInner = hasAssignee ? (
    <span className="flex min-w-0 max-w-full items-center gap-1.5">
      <ActorAvatar
        actorType={issue.assignee_type!}
        actorId={issue.assignee_id!}
        size={20}
        enableHoverCard
        className="shrink-0"
      />
      {assigneeName && (
        <span className="min-w-0 truncate text-xs text-foreground">{assigneeName}</span>
      )}
    </span>
  ) : (
    <span className="text-xs text-muted-foreground">{t(($) => $.pickers.assignee.trigger_unassigned)}</span>
  );

  const assigneeNode = showAssigneeSection ? (
    editable ? (
      <PickerWrapper className={assigneeContainerClass}>
        <AssigneePicker
          assigneeType={issue.assignee_type}
          assigneeId={issue.assignee_id}
          onUpdate={handleUpdate}
          trigger={assigneeInner}
        />
      </PickerWrapper>
    ) : (
      <span className={assigneeContainerClass}>{assigneeInner}</span>
    )
  ) : null;

  const showMetaRow = showAssigneeSection || showStartDate || showDueDate || showChildProgress;
  const showRightMeta = !!showStartDate || !!showDueDate || !!showChildProgress || showUpdatedHint;

  return (
    <div className={cn(
      "rounded-lg border-[0.5px] border-border bg-card py-3 px-2.5 shadow-[0_3px_6px_-2px_rgba(0,0,0,0.02),0_1px_1px_0_rgba(0,0,0,0.04)] transition-colors group-hover/card:border-accent group-hover/card:bg-accent group-data-[popup-open]/card:border-accent group-data-[popup-open]/card:bg-accent",
      liveness?.active && "ring-1 ring-brand/40 border-brand/30",
    )}>
      {/* Row 1: priority + identifier (left), agent activity + assignee (right) */}
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-1.5 min-w-0">
          {priorityIconNode}
          <p className="text-xs text-muted-foreground truncate">{issue.identifier}</p>
        </div>
        <IssueAgentActivityIndicator issueId={issue.id} />
      </div>

      {/* Row 2: Title */}
      <p className="mt-1 text-sm font-medium leading-snug line-clamp-2">
        {issue.title}
      </p>

      {showDescription && (() => {
        const preview = descriptionPreview(issue.description!);
        if (!preview) return null;
        return (
          <p className="mt-1 text-xs text-muted-foreground line-clamp-1">
            {preview}
          </p>
        );
      })()}

      {/* Chip row: feature + labels */}
      {(showFeature || showLabels) && (
        <div className="mt-1.5 flex items-center gap-1.5 flex-wrap">
          {showFeature && (
            <span className="inline-flex items-center gap-1 rounded-full bg-muted/60 px-1.5 py-0.5 text-[11px] text-muted-foreground max-w-[160px]">
              <FeatureIcon feature={feature} size="sm" />
              <span className="truncate">{feature!.title}</span>
            </span>
          )}
          {showLabels && labels.map((label) => (
            <LabelChip key={label.id} label={label} />
          ))}
        </div>
      )}

      {/* Meta row: assignee (left), start date, due date, child progress (right) */}
      {showMetaRow && (
        <div className="mt-2 flex items-center justify-between gap-2">
          {showAssigneeSection && (
            <div className="min-w-0 flex-1">
              {assigneeNode}
            </div>
          )}
          {showRightMeta && (
            <div className="ml-auto flex shrink-0 items-center gap-2">
              {showStartDate && (
                editable ? (
                  <PickerWrapper className="shrink-0">
                    <StartDatePicker
                      startDate={issue.start_date}
                      onUpdate={handleUpdate}
                      trigger={
                        <span className="flex items-center gap-1 text-xs text-muted-foreground">
                          <CalendarClock className="size-3" />
                          {formatDate(issue.start_date!)}
                        </span>
                      }
                    />
                  </PickerWrapper>
                ) : (
                  <span className="flex shrink-0 items-center gap-1 text-xs text-muted-foreground">
                    <CalendarClock className="size-3" />
                    {formatDate(issue.start_date!)}
                  </span>
                )
              )}
              {showDueDate && (
                editable ? (
                  <PickerWrapper className="shrink-0">
                    <DueDatePicker
                      dueDate={issue.due_date}
                      onUpdate={handleUpdate}
                      trigger={
                        <span
                          className={`flex items-center gap-1 text-xs ${
                            new Date(issue.due_date!) < new Date()
                              ? "text-destructive"
                              : "text-muted-foreground"
                          }`}
                        >
                          <CalendarDays className="size-3" />
                          {formatDate(issue.due_date!)}
                        </span>
                      }
                    />
                  </PickerWrapper>
                ) : (
                  <span
                    className={`flex shrink-0 items-center gap-1 text-xs ${
                      new Date(issue.due_date!) < new Date()
                        ? "text-destructive"
                        : "text-muted-foreground"
                    }`}
                  >
                    <CalendarDays className="size-3" />
                    {formatDate(issue.due_date!)}
                  </span>
                )
              )}
              {showChildProgress && (
                <div className="inline-flex shrink-0 items-center gap-1">
                  <ProgressRing done={childProgress!.done} total={childProgress!.total} size={14} />
                  <span className="text-[11px] text-muted-foreground tabular-nums font-medium">
                    {childProgress!.done}/{childProgress!.total}
                  </span>
                </div>
              )}
              {showUpdatedHint && (
                <span className="shrink-0 text-xs text-muted-foreground">
                  {t(($) => $.card.updated_ago, { time: timeAgo(issue.updated_at) })}
                </span>
              )}
            </div>
          )}
        </div>
      )}
      <BoardCardLiveLayer liveness={liveness} counters={counters} />
    </div>
  );
});

const animateLayoutChanges: AnimateLayoutChanges = (args) => {
  const { isSorting, wasDragging } = args;
  if (isSorting || wasDragging) return false;
  return defaultAnimateLayoutChanges(args);
};

export const DraggableBoardCard = memo(function DraggableBoardCard({ issue, childProgress, disableSorting }: { issue: Issue; childProgress?: ChildProgress; disableSorting?: boolean }) {
  const p = useWorkspacePaths();
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({
    id: issue.id,
    data: { status: issue.status },
    animateLayoutChanges,
    disabled: disableSorting ? { droppable: true } : undefined,
  });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  };

  return (
    <IssueActionsContextMenu issue={issue}>
      <div
        ref={setNodeRef}
        style={style}
        {...attributes}
        {...listeners}
        className={`group/card ${isDragging ? "opacity-30" : ""}`}
      >
        <AppLink
          href={p.issueDetail(issue.id)}
          className={`group block transition-colors ${isDragging ? "pointer-events-none" : ""}`}
        >
          <BoardCardContent issue={issue} editable childProgress={childProgress} />
        </AppLink>
      </div>
    </IssueActionsContextMenu>
  );
});
