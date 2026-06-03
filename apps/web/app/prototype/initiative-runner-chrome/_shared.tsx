"use client";

import Link from "next/link";
import {
  ArrowUpRight,
  Brain,
  Edit,
  TestTube2,
} from "lucide-react";
import { INITIATIVES, ISSUES, agentById, initiativeById } from "../initiative-runner/_data";
import type { Phase } from "../initiative-runner/_data";

// Shared mid-density content that sits inside every chrome variant — a
// compact list of running initiatives + running issues. Stays identical
// across variants so the chrome difference is what stands out.

export function SharedBody() {
  const runningInits = INITIATIVES.filter((i) => i.status === "running");
  const runningIssues = ISSUES.filter((i) => i.status === "in_progress");

  return (
    <div className="space-y-8">
      <header>
        <div className="flex items-center gap-2 text-xs uppercase tracking-wider text-muted-foreground">
          <span className="relative inline-flex size-2">
            <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75" />
            <span className="relative inline-flex size-2 rounded-full bg-emerald-500" />
          </span>
          live monitor
        </div>
        <h1 className="mt-2 text-2xl font-semibold tracking-tight">
          {runningIssues.length} agents at work
        </h1>
        <p className="mt-1 text-sm text-muted-foreground">
          across {runningInits.length} running initiatives.
        </p>
      </header>

      <section>
        <h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
          Active initiatives
        </h2>
        <div className="mt-3 grid gap-2">
          {runningInits.map((init) => (
            <Link
              key={init.id}
              href={`/prototype/initiative-runner/initiatives/${init.id}?variant=A`}
              className="flex items-center justify-between gap-3 rounded-md border border-border bg-card p-3 transition hover:border-foreground/30"
            >
              <div className="min-w-0 flex-1">
                <div className="truncate font-medium">{init.title}</div>
                <div className="mt-0.5 text-xs text-muted-foreground">
                  {init.milestonesPassed}/{init.milestonesTotal} milestones · {init.issuesDone}/{init.issuesTotal} issues · {init.lastActivityAt}
                </div>
              </div>
              <ArrowUpRight className="size-4 shrink-0 text-muted-foreground" />
            </Link>
          ))}
        </div>
      </section>

      <section>
        <h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
          Running now
        </h2>
        <ul className="mt-3 space-y-1.5">
          {runningIssues.map((issue) => {
            const agent = agentById(issue.assigneeId);
            const init = initiativeById(issue.initiativeId);
            return (
              <li key={issue.id}>
                <div className="flex items-center gap-3 rounded-md border border-border bg-card px-3 py-2 text-sm">
                  <span className="font-mono text-xs text-muted-foreground">#{issue.number}</span>
                  <span className="flex-1 truncate">{issue.title}</span>
                  {issue.phase && (
                    <span className="inline-flex items-center gap-1 text-[11px] text-muted-foreground">
                      {phaseIcon(issue.phase)} {phaseLabel(issue.phase)}
                    </span>
                  )}
                  {agent && (
                    <span
                      className="size-2 shrink-0 rounded-full"
                      style={{ background: `hsl(${agent.hue} 70% 55%)` }}
                      title={agent.name}
                    />
                  )}
                  <span className="text-[11px] text-muted-foreground/70 truncate">{init?.branchSlug}</span>
                </div>
              </li>
            );
          })}
        </ul>
      </section>
    </div>
  );
}

function phaseIcon(p: Phase) {
  switch (p) {
    case "thinking": return <Brain className="size-3" />;
    case "editing": return <Edit className="size-3" />;
    case "running_tests": return <TestTube2 className="size-3" />;
    case "committing": return <Brain className="size-3" />;
    case "waiting_local_dir": return <Brain className="size-3" />;
  }
}

function phaseLabel(p: Phase): string {
  return ({
    thinking: "Thinking",
    editing: "Editing",
    running_tests: "Tests",
    committing: "Commit",
    waiting_local_dir: "Waiting dir",
  } as const)[p];
}
