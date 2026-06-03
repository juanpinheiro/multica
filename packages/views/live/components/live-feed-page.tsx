"use client";

import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Activity } from "lucide-react";
import { useWorkspaceId } from "@multica/core/hooks";
import { useLiveEvents } from "@multica/core/tasks/use-live-events";
import { agentTaskSnapshotOptions } from "@multica/core/agents/queries";
import { issueListOptions } from "@multica/core/issues/queries";
import type {
  ActivityEvent,
  ActivityEventType,
} from "@multica/core/tasks/build-live-events";
import type { AgentTask, Issue } from "@multica/core/types";
import { useWorkspacePaths } from "@multica/core/paths";
import { PageHeader } from "../../layout/page-header";
import { AppLink } from "../../navigation";
import { useT } from "../../i18n";
import { LiveEventRow } from "./live-event-row";

type FilterKey = "all" | "running" | "decisions" | "failures";

const FILTERS: FilterKey[] = ["all", "running", "decisions", "failures"];

const FILTER_TYPES: Record<Exclude<FilterKey, "all">, ReadonlySet<ActivityEventType>> = {
  running: new Set<ActivityEventType>(["agent_started", "tool_use", "edit"]),
  decisions: new Set<ActivityEventType>([
    "milestone_passed",
    "initiative_ready_for_review",
  ]),
  failures: new Set<ActivityEventType>([
    "milestone_failed",
    "dod_failed",
    "tripwire_paused",
  ]),
};

function passesFilter(event: ActivityEvent, filter: FilterKey): boolean {
  if (filter === "all") return true;
  return FILTER_TYPES[filter].has(event.type);
}

function pickRunningTasks(tasks: AgentTask[]): AgentTask[] {
  return tasks.filter(
    (t) =>
      (t.status === "running" || t.status === "waiting_local_directory") &&
      t.issue_id,
  );
}

function LiveNowChips({
  tasks,
  issues,
}: {
  tasks: AgentTask[];
  issues: Issue[];
}) {
  const p = useWorkspacePaths();
  const issueById = useMemo(
    () => new Map(issues.map((i) => [i.id, i])),
    [issues],
  );
  const running = useMemo(() => pickRunningTasks(tasks), [tasks]);
  if (running.length === 0) return null;
  return (
    <div className="mt-4 flex flex-wrap gap-2">
      {running.map((task) => {
        const issue = issueById.get(task.issue_id);
        if (!issue || !issue.feature_id) return null;
        return (
          <AppLink
            key={task.id}
            data-testid="live-now-chip"
            href={p.initiativeIssue(issue.feature_id, issue.id)}
            className="flex items-center gap-2 rounded-full border border-border bg-card px-3 py-1 text-xs hover:border-foreground/30"
          >
            <span className="relative inline-flex size-2">
              <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75" />
              <span className="relative inline-flex size-2 rounded-full bg-emerald-500" />
            </span>
            <span className="font-medium">{issue.identifier}</span>
            <span className="text-muted-foreground truncate max-w-32">{issue.title}</span>
          </AppLink>
        );
      })}
    </div>
  );
}

function FilterChips({
  filter,
  onChange,
  labels,
}: {
  filter: FilterKey;
  onChange: (next: FilterKey) => void;
  labels: Record<FilterKey, string>;
}) {
  return (
    <div className="sticky top-2 z-10 flex gap-2 self-start rounded-full border border-border bg-background/80 p-1 backdrop-blur">
      {FILTERS.map((key) => (
        <button
          key={key}
          type="button"
          data-testid={`live-filter-${key}`}
          data-active={filter === key ? "" : undefined}
          onClick={() => onChange(key)}
          className={
            filter === key
              ? "rounded-full bg-foreground px-3 py-1 text-xs text-background"
              : "rounded-full px-3 py-1 text-xs text-muted-foreground transition hover:text-foreground"
          }
        >
          {labels[key]}
        </button>
      ))}
    </div>
  );
}

export function LiveFeedPage() {
  const { t } = useT("layout");
  const wsId = useWorkspaceId();
  const [filter, setFilter] = useState<FilterKey>("all");

  const { events, runningAgents, runningInitiatives } = useLiveEvents(wsId);
  const { data: tasks = [] } = useQuery(agentTaskSnapshotOptions(wsId));
  const { data: issues = [] } = useQuery(issueListOptions(wsId));

  const visibleEvents = useMemo(
    () => events.filter((e) => passesFilter(e, filter)),
    [events, filter],
  );

  const filterLabels: Record<FilterKey, string> = {
    all: t(($) => $.live_page.filter_all),
    running: t(($) => $.live_page.filter_running),
    decisions: t(($) => $.live_page.filter_decisions),
    failures: t(($) => $.live_page.filter_failures),
  };

  return (
    <div className="flex flex-1 min-h-0 flex-col">
      <PageHeader className="gap-1.5">
        <Activity className="size-4 text-muted-foreground" />
        <span className="text-sm font-medium">{t(($) => $.nav.live)}</span>
      </PageHeader>

      <div className="flex-1 overflow-y-auto">
        <div className="mx-auto flex max-w-3xl flex-col gap-6 px-6 py-8">
          <header>
            <div className="flex items-center gap-2 text-xs uppercase tracking-wider text-muted-foreground">
              <span className="relative inline-flex size-2">
                <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75" />
                <span className="relative inline-flex size-2 rounded-full bg-emerald-500" />
              </span>
              {t(($) => $.live_page.kicker)}
            </div>
            <h1 data-testid="live-headline" className="mt-2 text-3xl font-semibold tracking-tight">
              {t(($) => $.live_page.headline, { count: runningAgents })}
            </h1>
            <p className="mt-1 text-sm text-muted-foreground">
              {t(($) => $.live_page.subhead, { count: runningInitiatives })}
            </p>
            <LiveNowChips tasks={tasks} issues={issues} />
          </header>

          <FilterChips filter={filter} onChange={setFilter} labels={filterLabels} />

          {visibleEvents.length === 0 ? (
            <div data-testid="live-empty" className="rounded-lg border border-dashed border-border p-8 text-center">
              <p className="text-sm font-medium">{t(($) => $.live_page.empty_title)}</p>
              <p className="mt-1 text-xs text-muted-foreground">
                {t(($) => $.live_page.empty_hint)}
              </p>
            </div>
          ) : (
            <div className="relative pl-4">
              <div aria-hidden="true" className="absolute left-[7px] top-2 bottom-2 w-px bg-border" />
              <ul data-testid="live-feed" className="space-y-3">
                {visibleEvents.map((event) => (
                  <LiveEventRow key={event.id} event={event} />
                ))}
              </ul>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
