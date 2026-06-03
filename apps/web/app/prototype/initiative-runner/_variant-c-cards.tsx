"use client";

import Link from "next/link";
import {
  Activity,
  ArrowUpRight,
  CheckCircle2,
  Clock,
  GitBranch,
  GitPullRequest,
  PauseCircle,
  Sparkles,
  TriangleAlert,
} from "lucide-react";
import {
  INITIATIVES,
  activityByInitiative,
  agentById,
  formatTsAgo,
  issuesByInitiative,
} from "./_data";
import type { Initiative, Status } from "./_data";

const statusTone = (s: Status): { dot: string; pill: string; label: string } => {
  switch (s) {
    case "running": return { dot: "bg-emerald-500", pill: "bg-emerald-500/10 text-emerald-500 border-emerald-500/30", label: "Running" };
    case "in_review": return { dot: "bg-indigo-500", pill: "bg-indigo-500/10 text-indigo-500 border-indigo-500/30", label: "In review" };
    case "done": return { dot: "bg-muted-foreground", pill: "bg-muted text-muted-foreground border-border", label: "Done" };
    case "blocked": return { dot: "bg-red-500", pill: "bg-red-500/10 text-red-500 border-red-500/30", label: "Blocked" };
    case "ready": return { dot: "bg-amber-500", pill: "bg-amber-500/10 text-amber-500 border-amber-500/30", label: "Ready" };
  }
};

export function VariantCards({ variant }: { variant: string }) {
  // sort: running first, then in_review, ready, blocked, done
  const order: Record<Status, number> = { running: 0, in_review: 1, ready: 2, blocked: 3, done: 4 };
  const sorted = [...INITIATIVES].sort((a, b) => order[a.status] - order[b.status]);

  const counts = INITIATIVES.reduce<Record<Status, number>>(
    (acc, i) => ({ ...acc, [i.status]: (acc[i.status] ?? 0) + 1 }),
    { ready: 0, running: 0, in_review: 0, done: 0, blocked: 0 },
  );

  return (
    <div className="flex h-full flex-col">
      <header className="border-b border-border px-8 py-6">
        <div className="flex items-baseline justify-between">
          <div>
            <h1 className="font-serif text-4xl tracking-tight">Initiatives</h1>
            <p className="mt-1 text-sm text-muted-foreground">
              Claude Code plans, agents execute. You watch.
            </p>
          </div>
          <div className="flex gap-3 text-xs uppercase tracking-wider text-muted-foreground">
            <span><span className="text-foreground tabular-nums">{counts.running}</span> running</span>
            <span><span className="text-foreground tabular-nums">{counts.in_review}</span> in review</span>
            <span><span className="text-foreground tabular-nums">{counts.blocked}</span> blocked</span>
          </div>
        </div>
      </header>

      <div className="grid flex-1 grid-cols-1 gap-5 overflow-y-auto p-8 md:grid-cols-2 xl:grid-cols-3">
        {sorted.map((init) => (
          <InitiativeTile key={init.id} init={init} variant={variant} />
        ))}
      </div>
    </div>
  );
}

function InitiativeTile({ init, variant }: { init: Initiative; variant: string }) {
  const tone = statusTone(init.status);
  const issues = issuesByInitiative(init.id);
  const running = issues.filter((i) => i.status === "in_progress");
  const activity = activityByInitiative(init.id).slice(0, 3);

  const milestonePct = init.milestonesTotal > 0
    ? Math.round((init.milestonesPassed / init.milestonesTotal) * 100)
    : 0;

  return (
    <Link
      href={`/prototype/initiative-runner/initiatives/${init.id}?variant=${variant}`}
      className="group flex flex-col rounded-lg border border-border bg-card p-5 transition hover:border-foreground/30 hover:shadow-xl"
    >
      {/* header */}
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-center gap-2 text-[10px] uppercase tracking-wider text-muted-foreground">
          <GitBranch className="size-3" /> {init.branchSlug}
        </div>
        <span className={`flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-[11px] font-medium ${tone.pill}`}>
          <span className={`size-1.5 rounded-full ${tone.dot} ${init.status === "running" ? "animate-pulse" : ""}`} />
          {tone.label}
        </span>
      </div>

      <h2 className="mt-3 line-clamp-2 text-lg font-semibold leading-snug group-hover:underline">
        {init.title}
      </h2>

      {/* progress */}
      <div className="mt-4">
        <div className="flex items-baseline justify-between text-xs">
          <span className="text-muted-foreground">
            {init.milestonesPassed}/{init.milestonesTotal} milestones · {init.issuesDone}/{init.issuesTotal} issues
          </span>
          <span className="tabular-nums font-medium">{milestonePct}%</span>
        </div>
        <div className="mt-1.5 h-1.5 overflow-hidden rounded-full bg-muted">
          <div
            className={`h-full transition-all ${init.status === "done" ? "bg-muted-foreground" : init.status === "blocked" ? "bg-red-500" : "bg-emerald-500"}`}
            style={{ width: `${milestonePct}%` }}
          />
        </div>
      </div>

      {/* running agents */}
      {running.length > 0 && (
        <div className="mt-4 flex items-center gap-2">
          <Activity className="size-3.5 text-emerald-500" />
          <span className="text-xs">{running.length} agent{running.length > 1 ? "s" : ""} on it</span>
          <div className="flex items-center gap-1">
            {running.map((i) => {
              const agent = agentById(i.assigneeId);
              if (!agent) return null;
              return (
                <span
                  key={i.id}
                  className="size-2.5 rounded-full ring-2 ring-background"
                  style={{ background: `hsl(${agent.hue} 70% 55%)` }}
                  title={agent.name}
                />
              );
            })}
          </div>
        </div>
      )}

      {/* mini feed */}
      <div className="mt-4 flex-1 space-y-1 border-t border-border pt-3">
        {activity.map((a) => (
          <div key={a.id} className="flex items-baseline gap-2 text-xs text-muted-foreground">
            <span className="tabular-nums w-12 shrink-0">{formatTsAgo(a.tsMinutesAgo)}</span>
            <span className="line-clamp-1 flex-1">{a.message}</span>
          </div>
        ))}
        {activity.length === 0 && (
          <div className="text-xs text-muted-foreground/60">no activity yet</div>
        )}
      </div>

      {/* footer */}
      <div className="mt-4 flex items-center justify-between border-t border-border pt-3 text-xs text-muted-foreground">
        <span className="flex items-center gap-1.5">
          <Clock className="size-3" /> {init.lastActivityAt}
        </span>
        <div className="flex items-center gap-3">
          {init.prUrl && (
            <span className="flex items-center gap-1.5">
              <GitPullRequest className="size-3" /> PR open
            </span>
          )}
          {init.status === "blocked" && (
            <span className="flex items-center gap-1.5 text-red-500">
              <TriangleAlert className="size-3" /> needs you
            </span>
          )}
          {init.status === "done" && <CheckCircle2 className="size-3 text-emerald-500" />}
          <ArrowUpRight className="size-3.5 transition group-hover:translate-x-0.5 group-hover:-translate-y-0.5" />
        </div>
      </div>
    </Link>
  );
}

export const __unused = { Sparkles, PauseCircle };
