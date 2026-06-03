"use client";

import Link from "next/link";
import { useState } from "react";
import {
  Activity,
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
  Zap,
} from "lucide-react";
import {
  ACTIVITY,
  ISSUES,
  agentById,
  formatTsAgo,
  initiativeById,
  issueById,
} from "./_data";
import type { ActivityType } from "./_data";

const FILTERS = [
  { key: "all", label: "All activity" },
  { key: "running", label: "Live now" },
  { key: "decisions", label: "Decisions" },
  { key: "failures", label: "Failures" },
] as const;
type FilterKey = (typeof FILTERS)[number]["key"];

const eventIcon = (t: ActivityType) => {
  const cls = "size-3.5";
  switch (t) {
    case "agent_started": return <Play className={cls} />;
    case "tool_use": return <Wrench className={cls} />;
    case "edit": return <Edit3 className={cls} />;
    case "commit": return <GitCommit className={cls} />;
    case "milestone_passed": return <CheckCircle2 className={cls} />;
    case "milestone_failed": return <ShieldAlert className={cls} />;
    case "dod_failed": return <TriangleAlert className={cls} />;
    case "issue_done": return <Sparkles className={cls} />;
    case "initiative_ready_for_review": return <ArrowRight className={cls} />;
    case "tripwire_paused": return <PauseCircle className={cls} />;
  }
};

const eventTone = (t: ActivityType): { ring: string; bg: string; text: string } => {
  const muted = { ring: "ring-border", bg: "bg-muted", text: "text-muted-foreground" };
  switch (t) {
    case "milestone_passed":
    case "issue_done":
      return { ring: "ring-emerald-500/40", bg: "bg-emerald-500/10", text: "text-emerald-500" };
    case "milestone_failed":
    case "dod_failed":
      return { ring: "ring-amber-500/40", bg: "bg-amber-500/10", text: "text-amber-500" };
    case "tripwire_paused":
      return { ring: "ring-red-500/40", bg: "bg-red-500/10", text: "text-red-500" };
    case "initiative_ready_for_review":
      return { ring: "ring-indigo-500/40", bg: "bg-indigo-500/10", text: "text-indigo-500" };
    case "agent_started":
    case "commit":
      return { ring: "ring-sky-500/40", bg: "bg-sky-500/10", text: "text-sky-500" };
    default:
      return muted;
  }
};

const passesFilter = (type: ActivityType, f: FilterKey): boolean => {
  if (f === "all") return true;
  if (f === "running") return type === "agent_started" || type === "tool_use" || type === "edit";
  if (f === "decisions") return type === "milestone_passed" || type === "initiative_ready_for_review" || type === "issue_done";
  if (f === "failures") return type === "milestone_failed" || type === "dod_failed" || type === "tripwire_paused";
  return true;
};

export function VariantFeed({ variant }: { variant: string }) {
  const [filter, setFilter] = useState<FilterKey>("all");

  const runningIssues = ISSUES.filter((i) => i.status === "in_progress" && i.assigneeId);
  const events = ACTIVITY.filter((e) => passesFilter(e.type, filter)).sort(
    (a, b) => a.tsMinutesAgo - b.tsMinutesAgo,
  );

  return (
    <div className="mx-auto flex max-w-3xl flex-col gap-6 px-6 py-8">
      <header>
        <div className="flex items-center gap-2 text-xs uppercase tracking-wider text-muted-foreground">
          <span className="relative inline-flex size-2">
            <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75" />
            <span className="relative inline-flex size-2 rounded-full bg-emerald-500" />
          </span>
          live activity
        </div>
        <h1 className="mt-2 text-3xl font-semibold tracking-tight">
          {runningIssues.length} agents at work
        </h1>
        <p className="mt-1 text-sm text-muted-foreground">
          across {new Set(runningIssues.map((i) => i.initiativeId)).size} initiatives. Claude Code is the planner — you read here.
        </p>

        {/* live now strip */}
        <div className="mt-4 flex flex-wrap gap-2">
          {runningIssues.map((i) => {
            const agent = agentById(i.assigneeId);
            const init = initiativeById(i.initiativeId);
            if (!agent || !init) return null;
            return (
              <Link
                key={i.id}
                href={`/prototype/initiative-runner/initiatives/${i.initiativeId}/issues/${i.id}?variant=${variant}`}
                className="flex items-center gap-2 rounded-full border border-border bg-card px-3 py-1 text-xs hover:border-foreground/30"
              >
                <span
                  className="size-2 rounded-full"
                  style={{ background: `hsl(${agent.hue} 70% 55%)` }}
                />
                <span className="font-medium">{agent.name}</span>
                <span className="text-muted-foreground">→ #{i.number}</span>
              </Link>
            );
          })}
        </div>
      </header>

      {/* filter chips */}
      <div className="sticky top-2 z-10 flex gap-2 rounded-full border border-border bg-background/80 p-1 backdrop-blur self-start">
        {FILTERS.map((f) => (
          <button
            key={f.key}
            onClick={() => setFilter(f.key)}
            className={`rounded-full px-3 py-1 text-xs transition ${
              filter === f.key
                ? "bg-foreground text-background"
                : "text-muted-foreground hover:text-foreground"
            }`}
          >
            {f.label}
          </button>
        ))}
      </div>

      {/* timeline */}
      <div className="relative pl-4">
        <div className="absolute left-[7px] top-2 bottom-2 w-px bg-border" />
        <ul className="space-y-3">
          {events.map((e) => {
            const tone = eventTone(e.type);
            const agent = agentById(e.agentId);
            const init = initiativeById(e.initiativeId);
            const issue = e.issueId ? issueById(e.issueId) : undefined;
            const isLive = e.tsMinutesAgo === 0;

            return (
              <li key={e.id} className="relative pl-6">
                <div
                  className={`absolute left-0 top-1 grid size-3.5 -translate-x-1/2 place-items-center rounded-full ring-2 ${tone.ring} ${tone.bg} ${
                    isLive ? "animate-pulse" : ""
                  }`}
                >
                  <span className={tone.text}>{eventIcon(e.type)}</span>
                </div>
                <div className="text-xs tabular-nums text-muted-foreground">
                  {formatTsAgo(e.tsMinutesAgo)}
                  {agent && (
                    <>
                      {" • "}
                      <span style={{ color: `hsl(${agent.hue} 70% 60%)` }}>{agent.name}</span>
                    </>
                  )}
                </div>
                <div className="mt-0.5 text-sm">{e.message}</div>
                <div className="mt-1.5 flex flex-wrap gap-2">
                  {init && (
                    <Link
                      href={`/prototype/initiative-runner/initiatives/${init.id}?variant=${variant}`}
                      className="rounded border border-border bg-card px-1.5 py-0.5 text-[11px] hover:border-foreground/30"
                    >
                      {init.title}
                    </Link>
                  )}
                  {issue && (
                    <Link
                      href={`/prototype/initiative-runner/initiatives/${issue.initiativeId}/issues/${issue.id}?variant=${variant}`}
                      className="rounded border border-border bg-card px-1.5 py-0.5 text-[11px] hover:border-foreground/30"
                    >
                      #{issue.number} {issue.title}
                    </Link>
                  )}
                </div>
              </li>
            );
          })}
        </ul>
      </div>
    </div>
  );
}

export const __unused = { Activity, Zap };
