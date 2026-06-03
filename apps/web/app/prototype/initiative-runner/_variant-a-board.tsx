"use client";

import Link from "next/link";
import {
  Activity,
  Brain,
  CheckCircle2,
  CircleDashed,
  Cog,
  Edit,
  GitBranch,
  PauseCircle,
  TestTube2,
  Timer,
  TriangleAlert,
} from "lucide-react";
import {
  AGENTS,
  INITIATIVES,
  ISSUES,
  ISSUE_STATUS_LABEL,
  agentById,
  initiativeById,
} from "./_data";
import type { Issue, IssueStatus, Phase } from "./_data";

const LANES: { key: IssueStatus; label: string }[] = [
  { key: "todo", label: "Backlog" },
  { key: "in_progress", label: "Running" },
  { key: "in_review", label: "Validating" },
  { key: "done", label: "Done" },
];

const phaseIcon = (phase: Phase) => {
  switch (phase) {
    case "thinking": return <Brain className="size-3" />;
    case "editing": return <Edit className="size-3" />;
    case "running_tests": return <TestTube2 className="size-3" />;
    case "committing": return <Cog className="size-3" />;
    case "waiting_local_dir": return <PauseCircle className="size-3" />;
  }
};

const phaseLabel = (phase: Phase): string => ({
  thinking: "Thinking",
  editing: "Editing",
  running_tests: "Tests",
  committing: "Commit",
  waiting_local_dir: "Waiting dir",
}[phase]);

const heartbeatTone = (ms?: number): string => {
  if (ms == null) return "bg-muted";
  if (ms < 2000) return "bg-emerald-500 animate-pulse";
  if (ms < 10000) return "bg-amber-500";
  return "bg-red-500";
};

export function VariantBoard({ variant }: { variant: string }) {
  const groups = LANES.map((lane) => ({
    ...lane,
    issues: ISSUES.filter((i) => i.status === lane.key),
  }));

  const running = ISSUES.filter((i) => i.status === "in_progress");
  const activeInitiatives = new Set(running.map((i) => i.initiativeId)).size;
  const activeAgents = new Set(running.map((i) => i.assigneeId).filter(Boolean)).size;

  return (
    <div className="flex h-screen flex-col">
      {/* ambient bar */}
      <header className="flex items-center justify-between border-b border-border px-6 py-3 text-sm">
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2">
            <span className="relative inline-flex size-2">
              <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75" />
              <span className="relative inline-flex size-2 rounded-full bg-emerald-500" />
            </span>
            <span className="font-medium">Live monitor</span>
          </div>
          <span className="text-muted-foreground">
            {activeInitiatives} initiatives • {activeAgents} agents running
          </span>
        </div>
        <div className="flex items-center gap-3 text-xs text-muted-foreground">
          <span className="flex items-center gap-1.5">
            <GitBranch className="size-3.5" /> ~/code/upgrade
          </span>
          <span>worktree mode</span>
        </div>
      </header>

      {/* board */}
      <div className="grid flex-1 grid-cols-4 gap-4 overflow-hidden p-4">
        {groups.map((lane) => (
          <Lane key={lane.key} label={lane.label} issues={lane.issues} variant={variant} />
        ))}
      </div>
    </div>
  );
}

function Lane({ label, issues, variant }: { label: string; issues: Issue[]; variant: string }) {
  // group within lane by initiative
  const byInit = new Map<string, Issue[]>();
  for (const i of issues) {
    const arr = byInit.get(i.initiativeId) ?? [];
    arr.push(i);
    byInit.set(i.initiativeId, arr);
  }

  return (
    <div className="flex flex-col rounded-md border border-border bg-card/30">
      <div className="flex items-center justify-between border-b border-border px-3 py-2 text-xs font-medium uppercase tracking-wider text-muted-foreground">
        <span>{label}</span>
        <span className="tabular-nums">{issues.length}</span>
      </div>
      <div className="flex-1 space-y-4 overflow-y-auto p-2">
        {[...byInit.entries()].map(([initId, items]) => {
          const init = initiativeById(initId);
          if (!init) return null;
          return (
            <div key={initId} className="space-y-1.5">
              <Link
                href={`/prototype/initiative-runner/initiatives/${initId}?variant=${variant}`}
                className="block truncate px-2 text-[11px] font-medium uppercase tracking-wider text-muted-foreground/80 hover:text-foreground"
                title={init.title}
              >
                {init.title}
              </Link>
              {items.map((issue) => (
                <IssueCard key={issue.id} issue={issue} variant={variant} />
              ))}
            </div>
          );
        })}
        {issues.length === 0 && (
          <div className="flex h-24 items-center justify-center text-xs text-muted-foreground/60">
            empty
          </div>
        )}
      </div>
    </div>
  );
}

function IssueCard({ issue, variant }: { issue: Issue; variant: string }) {
  const agent = agentById(issue.assigneeId);
  const isLive = issue.status === "in_progress" && issue.phase !== undefined;

  return (
    <Link
      href={`/prototype/initiative-runner/initiatives/${issue.initiativeId}/issues/${issue.id}?variant=${variant}`}
      className="block rounded-md border border-border bg-card p-2.5 text-sm shadow-sm transition hover:border-foreground/30 hover:shadow-md"
    >
      <div className="flex items-start justify-between gap-2">
        <span className="text-[11px] font-mono text-muted-foreground">#{issue.number}</span>
        <span className="text-[10px] uppercase tracking-wider text-muted-foreground">
          {issue.repo}
        </span>
      </div>
      <div className="mt-1 line-clamp-2 leading-snug">{issue.title}</div>

      {isLive && (
        <div className="mt-2 flex items-center gap-2 rounded border border-emerald-500/30 bg-emerald-500/5 px-2 py-1 text-[11px]">
          <span className={`size-1.5 rounded-full ${heartbeatTone(issue.heartbeatMs)}`} />
          {phaseIcon(issue.phase!)}
          <span>{phaseLabel(issue.phase!)}</span>
          <span className="ml-auto text-muted-foreground tabular-nums">
            {issue.toolCount}t · {issue.editCount}e
          </span>
        </div>
      )}

      {issue.status === "blocked" && (
        <div className="mt-2 flex items-center gap-1.5 text-[11px] text-amber-500">
          <TriangleAlert className="size-3" /> blocked
        </div>
      )}

      <div className="mt-2 flex items-center justify-between">
        {agent ? (
          <div
            className="flex items-center gap-1.5 text-[11px]"
            title={`${agent.name} • ${agent.backend}`}
          >
            <span
              className="inline-block size-2 rounded-full"
              style={{ background: `hsl(${agent.hue} 70% 55%)` }}
            />
            <span className="text-muted-foreground">{agent.name}</span>
          </div>
        ) : (
          <div className="text-[11px] text-muted-foreground/60">unassigned</div>
        )}
        {issue.status === "done" && <CheckCircle2 className="size-3.5 text-emerald-500" />}
      </div>
    </Link>
  );
}

export const __unused = { Activity, Timer, CircleDashed, AGENTS, INITIATIVES, ISSUE_STATUS_LABEL };
