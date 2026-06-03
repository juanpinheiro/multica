import Link from "next/link";
import { notFound } from "next/navigation";
import {
  ArrowLeft,
  Brain,
  CheckCircle2,
  Cog,
  Edit,
  GitBranch,
  PauseCircle,
  TestTube2,
  TriangleAlert,
} from "lucide-react";
import {
  ACTIVITY,
  agentById,
  formatTsAgo,
  initiativeById,
  issueById,
} from "../../../../_data";
import type { Phase } from "../../../../_data";
import { PrototypeSwitcher } from "../../../../_switcher";

export const dynamic = "force-dynamic";

const phaseIcon = (p: Phase) => {
  switch (p) {
    case "thinking": return <Brain className="size-4" />;
    case "editing": return <Edit className="size-4" />;
    case "running_tests": return <TestTube2 className="size-4" />;
    case "committing": return <Cog className="size-4" />;
    case "waiting_local_dir": return <PauseCircle className="size-4" />;
  }
};

const phaseText = (p: Phase): string => ({
  thinking: "Thinking",
  editing: "Editing files",
  running_tests: "Running tests",
  committing: "Committing",
  waiting_local_dir: "Waiting on local directory",
}[p]);

export default async function IssueDetailPage({
  params,
  searchParams,
}: {
  params: Promise<{ id: string; issueId: string }>;
  searchParams: Promise<{ variant?: string }>;
}) {
  const { id, issueId } = await params;
  const sp = await searchParams;
  const variant = (sp.variant ?? "A").toUpperCase();

  const init = initiativeById(id);
  const issue = issueById(issueId);
  if (!init || !issue) notFound();

  const agent = agentById(issue.assigneeId);
  const issueEvents = ACTIVITY.filter((a) => a.issueId === issue.id).sort(
    (a, b) => a.tsMinutesAgo - b.tsMinutesAgo,
  );

  return (
    <>
      <div className="mx-auto max-w-3xl px-8 py-8">
        <Link
          href={`/prototype/initiative-runner/initiatives/${init.id}?variant=${variant}`}
          className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="size-3.5" /> {init.title}
        </Link>

        <header className="mt-4">
          <div className="flex items-center gap-3 text-xs uppercase tracking-wider text-muted-foreground">
            <span className="font-mono">#{issue.number}</span>
            <span className="flex items-center gap-1"><GitBranch className="size-3" /> {issue.repo}</span>
            <span>{issue.milestone}</span>
          </div>
          <h1 className="mt-2 text-2xl font-semibold tracking-tight">{issue.title}</h1>
        </header>

        {/* live state card */}
        {issue.status === "in_progress" && issue.phase && (
          <section className="mt-6 rounded-lg border border-emerald-500/30 bg-emerald-500/5 p-5">
            <div className="flex items-center gap-3">
              <span className="relative inline-flex size-3">
                <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75" />
                <span className="relative inline-flex size-3 rounded-full bg-emerald-500" />
              </span>
              <span className="font-medium">Live</span>
              <div className="flex items-center gap-1.5 text-sm text-muted-foreground">
                {phaseIcon(issue.phase)} {phaseText(issue.phase)}
              </div>
              <span className="ml-auto text-xs text-muted-foreground tabular-nums">
                heartbeat {issue.heartbeatMs}ms ago
              </span>
            </div>
            {agent && (
              <div className="mt-3 flex items-center gap-2 text-sm">
                <span
                  className="size-2.5 rounded-full"
                  style={{ background: `hsl(${agent.hue} 70% 55%)` }}
                />
                <span className="font-medium">{agent.name}</span>
                <span className="text-muted-foreground">({agent.backend} backend)</span>
              </div>
            )}
            <div className="mt-3 grid grid-cols-2 gap-3 text-sm">
              <Counter label="Tool calls" value={issue.toolCount ?? 0} />
              <Counter label="Edits" value={issue.editCount ?? 0} />
            </div>
          </section>
        )}

        {issue.status === "blocked" && (
          <section className="mt-6 flex items-center gap-2 rounded-lg border border-red-500/30 bg-red-500/5 p-4 text-sm text-red-500">
            <TriangleAlert className="size-4" /> Blocked — initiative paused by tripwire.
          </section>
        )}

        {issue.status === "done" && (
          <section className="mt-6 flex items-center gap-2 rounded-lg border border-emerald-500/30 bg-emerald-500/5 p-4 text-sm text-emerald-500">
            <CheckCircle2 className="size-4" /> Done — handed off to the next issue.
          </section>
        )}

        {/* per-issue timeline */}
        <section className="mt-8">
          <h2 className="text-sm font-semibold uppercase tracking-wider text-muted-foreground">
            Run timeline
          </h2>
          <div className="relative mt-3 pl-4">
            <div className="absolute left-[7px] top-2 bottom-2 w-px bg-border" />
            <ul className="space-y-3">
              {issueEvents.length === 0 && (
                <li className="text-sm text-muted-foreground">No events for this issue yet.</li>
              )}
              {issueEvents.map((e) => (
                <li key={e.id} className="relative pl-6">
                  <span className="absolute left-0 top-1.5 size-2 -translate-x-1/2 rounded-full bg-foreground/40 ring-2 ring-background" />
                  <div className="text-xs tabular-nums text-muted-foreground">
                    {formatTsAgo(e.tsMinutesAgo)}
                  </div>
                  <div className="text-sm">{e.message}</div>
                </li>
              ))}
            </ul>
          </div>
        </section>
      </div>
      <PrototypeSwitcher current={variant} />
    </>
  );
}

function Counter({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-md border border-border bg-card/60 px-3 py-2">
      <div className="text-[11px] uppercase tracking-wider text-muted-foreground">{label}</div>
      <div className="mt-0.5 font-mono text-lg tabular-nums">{value}</div>
    </div>
  );
}
