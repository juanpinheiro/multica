"use client";

import { useQuery } from "@tanstack/react-query";
import {
  ArrowRight,
  CheckCircle2,
  Edit3,
  GitCommit,
  PauseCircle,
  Play,
  ShieldAlert,
  Sparkles,
  TriangleAlert,
  Wrench,
} from "lucide-react";
import { featureListOptions } from "@multica/core/features/queries";
import { issueListOptions } from "@multica/core/issues/queries";
import { useWorkspaceId } from "@multica/core/hooks";
import { useWorkspacePaths } from "@multica/core/paths";
import type {
  ActivityEvent,
  ActivityEventType,
} from "@multica/core/tasks/build-live-events";
import { AppLink } from "../../navigation";
import { useT, useTimeAgo } from "../../i18n";

const ICONS: Record<ActivityEventType, typeof Play> = {
  agent_started: Play,
  tool_use: Wrench,
  edit: Edit3,
  commit: GitCommit,
  milestone_passed: CheckCircle2,
  milestone_failed: ShieldAlert,
  dod_failed: TriangleAlert,
  issue_done: Sparkles,
  initiative_ready_for_review: ArrowRight,
  tripwire_paused: PauseCircle,
};

function toneFor(type: ActivityEventType): string {
  switch (type) {
    case "milestone_passed":
    case "issue_done":
      return "text-emerald-500 ring-emerald-500/40 bg-emerald-500/10";
    case "milestone_failed":
    case "dod_failed":
      return "text-amber-500 ring-amber-500/40 bg-amber-500/10";
    case "tripwire_paused":
      return "text-red-500 ring-red-500/40 bg-red-500/10";
    case "initiative_ready_for_review":
      return "text-indigo-500 ring-indigo-500/40 bg-indigo-500/10";
    case "agent_started":
    case "commit":
      return "text-sky-500 ring-sky-500/40 bg-sky-500/10";
    default:
      return "text-muted-foreground ring-border bg-muted";
  }
}

export interface LiveEventRowProps {
  event: ActivityEvent;
}

function useMessageFor(event: ActivityEvent): string {
  const { t } = useT("layout");
  if (event.message) return event.message;
  if (event.type === "agent_started") return t(($) => $.live_page.agent_started_message);
  if (event.type === "tool_use") return t(($) => $.live_page.agent_activity_message);
  return "";
}

export function LiveEventRow({ event }: LiveEventRowProps) {
  const wsId = useWorkspaceId();
  const p = useWorkspacePaths();
  const timeAgo = useTimeAgo();
  const message = useMessageFor(event);
  const { data: features = [] } = useQuery(featureListOptions(wsId));
  const { data: issues = [] } = useQuery(issueListOptions(wsId));

  const Icon = ICONS[event.type];
  const tone = toneFor(event.type);
  const feature = event.initiativeId
    ? features.find((f) => f.id === event.initiativeId)
    : undefined;
  const issue = event.issueId
    ? issues.find((i) => i.id === event.issueId)
    : undefined;

  return (
    <li data-testid="live-event-row" data-event-type={event.type} className="relative pl-6">
      <span
        aria-hidden="true"
        className={`absolute left-0 top-1 grid size-4 -translate-x-1/2 place-items-center rounded-full ring-2 ${tone}`}
      >
        <Icon className="size-3" />
      </span>
      <div className="text-xs tabular-nums text-muted-foreground">
        {timeAgo(event.at)}
        {event.phase && (
          <>
            {" • "}
            <span data-testid="live-event-phase">{event.phase}</span>
          </>
        )}
        {event.heartbeat && (
          <>
            {" • "}
            <span data-testid="live-event-heartbeat">{event.heartbeat}</span>
          </>
        )}
      </div>
      <p className="mt-0.5 text-sm">{message}</p>
      {(feature || issue) && (
        <div className="mt-1.5 flex flex-wrap gap-2">
          {feature && (
            <AppLink
              href={p.initiativeDetail(feature.id)}
              data-testid="live-event-initiative-chip"
              className="rounded border border-border bg-card px-1.5 py-0.5 text-[11px] hover:border-foreground/30"
            >
              {feature.title}
            </AppLink>
          )}
          {issue && (
            <AppLink
              href={
                feature
                  ? p.initiativeIssue(feature.id, issue.id)
                  : p.issueDetail(issue.id)
              }
              data-testid="live-event-issue-chip"
              className="rounded border border-border bg-card px-1.5 py-0.5 text-[11px] hover:border-foreground/30"
            >
              {issue.identifier} {issue.title}
            </AppLink>
          )}
        </div>
      )}
    </li>
  );
}
