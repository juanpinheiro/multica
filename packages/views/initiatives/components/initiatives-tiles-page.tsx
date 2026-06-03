"use client";

import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  Activity,
  ArrowUpRight,
  CheckCircle2,
  Clock,
  GitBranch,
  Layers,
  TriangleAlert,
} from "lucide-react";
import { featureListOptions } from "@multica/core/features/queries";
import { issueListOptions } from "@multica/core/issues/queries";
import { milestoneListOptions } from "@multica/core/milestones/queries";
import { agentTaskSnapshotOptions } from "@multica/core/agents/queries";
import { useLiveEvents } from "@multica/core/tasks/use-live-events";
import { useWorkspaceId } from "@multica/core/hooks";
import { useWorkspacePaths } from "@multica/core/paths";
import type {
  AgentTask,
  Feature,
  FeatureStatus,
  Issue,
  Milestone,
} from "@multica/core/types";
import type { ActivityEvent } from "@multica/core/tasks/build-live-events";
import { cn } from "@multica/ui/lib/utils";
import { AppLink } from "../../navigation";
import { PageHeader } from "../../layout/page-header";
import { useT, useTimeAgo } from "../../i18n";

const STATUS_ORDER: Record<FeatureStatus, number> = {
  running: 0,
  in_review: 1,
  ready: 2,
  draft: 3,
  blocked: 4,
  done: 5,
  cancelled: 6,
};

const STATUS_TONE: Record<
  FeatureStatus,
  { dot: string; pill: string; bar: string }
> = {
  running: {
    dot: "bg-emerald-500 animate-pulse",
    pill: "bg-emerald-500/10 text-emerald-500 border-emerald-500/30",
    bar: "bg-emerald-500",
  },
  in_review: {
    dot: "bg-indigo-500",
    pill: "bg-indigo-500/10 text-indigo-500 border-indigo-500/30",
    bar: "bg-emerald-500",
  },
  ready: {
    dot: "bg-amber-500",
    pill: "bg-amber-500/10 text-amber-500 border-amber-500/30",
    bar: "bg-emerald-500",
  },
  draft: {
    dot: "bg-muted-foreground",
    pill: "bg-muted text-muted-foreground border-border",
    bar: "bg-emerald-500",
  },
  blocked: {
    dot: "bg-red-500",
    pill: "bg-red-500/10 text-red-500 border-red-500/30",
    bar: "bg-red-500",
  },
  done: {
    dot: "bg-muted-foreground",
    pill: "bg-muted text-muted-foreground border-border",
    bar: "bg-muted-foreground",
  },
  cancelled: {
    dot: "bg-muted-foreground",
    pill: "bg-muted text-muted-foreground border-border",
    bar: "bg-muted-foreground",
  },
};

interface MilestoneTally {
  passed: number;
  total: number;
}

function tallyMilestones(milestones: readonly Milestone[]): Map<string, MilestoneTally> {
  const out = new Map<string, MilestoneTally>();
  for (const m of milestones) {
    const t = out.get(m.feature_id) ?? { passed: 0, total: 0 };
    t.total += 1;
    if (m.validation_status === "passed") t.passed += 1;
    out.set(m.feature_id, t);
  }
  return out;
}

function runningTasksByFeature(
  tasks: readonly AgentTask[],
  issues: readonly Issue[],
): Map<string, AgentTask[]> {
  const issueToFeature = new Map(issues.map((i) => [i.id, i.feature_id ?? null]));
  const out = new Map<string, AgentTask[]>();
  for (const task of tasks) {
    if (task.status !== "running" && task.status !== "waiting_local_directory") continue;
    if (!task.issue_id) continue;
    const featureId = issueToFeature.get(task.issue_id) ?? null;
    if (!featureId) continue;
    const list = out.get(featureId) ?? [];
    list.push(task);
    out.set(featureId, list);
  }
  return out;
}

function eventsByFeature(events: readonly ActivityEvent[]): Map<string, ActivityEvent[]> {
  const out = new Map<string, ActivityEvent[]>();
  for (const e of events) {
    if (!e.initiativeId) continue;
    const list = out.get(e.initiativeId) ?? [];
    list.push(e);
    out.set(e.initiativeId, list);
  }
  return out;
}

function MiniFeedRow({ event }: { event: ActivityEvent }) {
  const timeAgo = useTimeAgo();
  const { t } = useT("layout");
  const message = event.message
    || (event.type === "agent_started" ? t(($) => $.live_page.agent_started_message) : "")
    || (event.type === "tool_use" ? t(($) => $.live_page.agent_activity_message) : "");
  return (
    <div
      data-testid="initiative-mini-feed-row"
      className="flex items-baseline gap-2 text-xs text-muted-foreground"
    >
      <span className="tabular-nums w-12 shrink-0">{timeAgo(event.at)}</span>
      <span className="line-clamp-1 flex-1">{message}</span>
    </div>
  );
}

function RunningAgentsRow({ tasks }: { tasks: AgentTask[] }) {
  const { t } = useT("layout");
  return (
    <div
      data-testid="running-agents-row"
      className="mt-4 flex items-center gap-2"
    >
      <Activity className="size-3.5 text-emerald-500" />
      <span className="text-xs">
        {t(($) => $.initiatives_page.agents_on_it, {
          count: tasks.length,
        })}
      </span>
      <div className="flex items-center gap-1">
        {tasks.map((task) => (
          <span
            key={task.id}
            className="size-2.5 rounded-full bg-emerald-500 ring-2 ring-background"
            title={task.agent_id}
          />
        ))}
      </div>
    </div>
  );
}

interface InitiativeTileProps {
  feature: Feature;
  milestoneTally: MilestoneTally;
  runningTasks: AgentTask[];
  events: ActivityEvent[];
}

function InitiativeTile({
  feature,
  milestoneTally,
  runningTasks,
  events,
}: InitiativeTileProps) {
  const { t } = useT("layout");
  const p = useWorkspacePaths();
  const timeAgo = useTimeAgo();
  const tone = STATUS_TONE[feature.status];
  const statusLabel = t(($) => $.initiatives_page.status[feature.status]);

  const issuesTotal = feature.issue_count;
  const issuesDone = feature.done_count;
  const milestonesPct = milestoneTally.total > 0
    ? Math.round((milestoneTally.passed / milestoneTally.total) * 100)
    : issuesTotal > 0
      ? Math.round((issuesDone / issuesTotal) * 100)
      : 0;

  return (
    <AppLink
      data-testid={`initiative-tile-${feature.id}`}
      href={p.initiativeDetail(feature.id)}
      className="group flex flex-col rounded-lg border border-border bg-card p-5 transition hover:border-foreground/30 hover:shadow-lg"
    >
      <div className="flex items-start justify-between gap-3">
        {feature.branch_slug ? (
          <div className="flex items-center gap-2 text-[10px] uppercase tracking-wider text-muted-foreground">
            <GitBranch className="size-3" /> {feature.branch_slug}
          </div>
        ) : (
          <span aria-hidden />
        )}
        <span
          className={cn(
            "flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-[11px] font-medium",
            tone.pill,
          )}
        >
          <span className={cn("size-1.5 rounded-full", tone.dot)} />
          {statusLabel}
        </span>
      </div>

      <h2 className="mt-3 line-clamp-2 text-lg font-semibold leading-snug group-hover:underline">
        {feature.title}
      </h2>

      <div className="mt-4">
        <div className="flex items-baseline justify-between text-xs">
          <span className="text-muted-foreground">
            {milestoneTally.total > 0 && (
              <>
                {t(($) => $.initiatives_page.progress_milestones, {
                  passed: milestoneTally.passed,
                  total: milestoneTally.total,
                })}
                {" · "}
              </>
            )}
            {t(($) => $.initiatives_page.progress_issues, {
              done: issuesDone,
              total: issuesTotal,
            })}
          </span>
          <span className="tabular-nums font-medium">{milestonesPct}%</span>
        </div>
        <div className="mt-1.5 h-1.5 overflow-hidden rounded-full bg-muted">
          <div
            className={cn("h-full transition-all", tone.bar)}
            style={{ width: `${milestonesPct}%` }}
          />
        </div>
      </div>

      {runningTasks.length > 0 && <RunningAgentsRow tasks={runningTasks} />}

      <div className="mt-4 flex-1 space-y-1 border-t border-border pt-3">
        {events.length === 0 ? (
          <div className="text-xs text-muted-foreground/60">
            {t(($) => $.initiatives_page.no_activity)}
          </div>
        ) : (
          events.slice(0, 3).map((event) => (
            <MiniFeedRow key={event.id} event={event} />
          ))
        )}
      </div>

      <div className="mt-4 flex items-center justify-between border-t border-border pt-3 text-xs text-muted-foreground">
        <span className="flex items-center gap-1.5">
          <Clock className="size-3" /> {timeAgo(feature.updated_at)}
        </span>
        <div className="flex items-center gap-3">
          {feature.status === "blocked" && (
            <span
              data-testid="blocked-indicator"
              className="flex items-center gap-1.5 text-red-500"
            >
              <TriangleAlert className="size-3" />
              {t(($) => $.initiatives_page.blocked_indicator)}
            </span>
          )}
          {feature.status === "done" && (
            <CheckCircle2 className="size-3 text-emerald-500" />
          )}
          <ArrowUpRight className="size-3.5 transition group-hover:translate-x-0.5 group-hover:-translate-y-0.5" />
        </div>
      </div>
    </AppLink>
  );
}

function sortByStatus(features: readonly Feature[]): Feature[] {
  return [...features].sort((a, b) => {
    const diff = STATUS_ORDER[a.status] - STATUS_ORDER[b.status];
    if (diff !== 0) return diff;
    return (
      new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime()
    );
  });
}

export function InitiativesTilesPage() {
  const { t } = useT("layout");
  const wsId = useWorkspaceId();
  const { data: features = [] } = useQuery(featureListOptions(wsId));
  const { data: issues = [] } = useQuery(issueListOptions(wsId));
  const { data: milestones = [] } = useQuery(milestoneListOptions(wsId));
  const { data: tasks = [] } = useQuery(agentTaskSnapshotOptions(wsId));
  const { events } = useLiveEvents(wsId);

  const sorted = useMemo(() => sortByStatus(features), [features]);
  const milestoneTallies = useMemo(() => tallyMilestones(milestones), [milestones]);
  const runningByFeature = useMemo(
    () => runningTasksByFeature(tasks, issues),
    [tasks, issues],
  );
  const eventsByFeatureId = useMemo(() => eventsByFeature(events), [events]);

  const counts = useMemo(() => {
    let running = 0;
    let inReview = 0;
    let blocked = 0;
    for (const f of features) {
      if (f.status === "running") running += 1;
      else if (f.status === "in_review") inReview += 1;
      else if (f.status === "blocked") blocked += 1;
    }
    return { running, inReview, blocked };
  }, [features]);

  return (
    <div className="flex flex-1 min-h-0 flex-col">
      <PageHeader className="gap-1.5">
        <Layers className="size-4 text-muted-foreground" />
        <span className="text-sm font-medium">{t(($) => $.nav.initiatives)}</span>
      </PageHeader>

      <div className="flex-1 overflow-y-auto">
        <header className="border-b border-border px-8 py-6">
          <div className="flex items-baseline justify-between gap-4">
            <div>
              <h1 className="text-3xl font-semibold tracking-tight">
                {t(($) => $.initiatives_page.headline)}
              </h1>
              <p className="mt-1 text-sm text-muted-foreground">
                {t(($) => $.initiatives_page.subhead)}
              </p>
            </div>
            <div className="hidden gap-3 text-xs uppercase tracking-wider text-muted-foreground sm:flex">
              <span>
                <span className="text-foreground tabular-nums">{counts.running}</span>{" "}
                {t(($) => $.initiatives_page.counter_running)}
              </span>
              <span>
                <span className="text-foreground tabular-nums">{counts.inReview}</span>{" "}
                {t(($) => $.initiatives_page.counter_in_review)}
              </span>
              <span>
                <span className="text-foreground tabular-nums">{counts.blocked}</span>{" "}
                {t(($) => $.initiatives_page.counter_blocked)}
              </span>
            </div>
          </div>
        </header>

        {sorted.length === 0 ? (
          <div
            data-testid="initiatives-empty"
            className="mx-8 my-8 rounded-lg border border-dashed border-border p-12 text-center"
          >
            <p className="text-sm font-medium">
              {t(($) => $.initiatives_page.empty_title)}
            </p>
            <p className="mt-1 text-xs text-muted-foreground">
              {t(($) => $.initiatives_page.empty_hint)}
            </p>
          </div>
        ) : (
          <div className="grid grid-cols-1 gap-5 px-8 py-6 md:grid-cols-2 xl:grid-cols-3">
            {sorted.map((feature) => (
              <InitiativeTile
                key={feature.id}
                feature={feature}
                milestoneTally={milestoneTallies.get(feature.id) ?? { passed: 0, total: 0 }}
                runningTasks={runningByFeature.get(feature.id) ?? []}
                events={eventsByFeatureId.get(feature.id) ?? []}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
