import Link from "next/link";
import { notFound } from "next/navigation";
import {
  ArrowLeft,
  BookOpen,
  CheckCircle2,
  Circle,
  GitBranch,
  GitPullRequest,
  ScrollText,
} from "lucide-react";
import {
  activityByInitiative,
  agentById,
  formatTsAgo,
  initiativeById,
  issuesByInitiative,
} from "../../_data";
import { PrototypeSwitcher } from "../../_switcher";

export const dynamic = "force-dynamic";

export default async function InitiativeDetailPage({
  params,
  searchParams,
}: {
  params: Promise<{ id: string }>;
  searchParams: Promise<{ variant?: string }>;
}) {
  const { id } = await params;
  const sp = await searchParams;
  const variant = (sp.variant ?? "A").toUpperCase();
  const init = initiativeById(id);
  if (!init) notFound();

  const issues = issuesByInitiative(init.id);
  const activity = activityByInitiative(init.id);

  // group issues by milestone
  const milestoneGroups = new Map<string, typeof issues>();
  for (const i of issues) {
    const arr = milestoneGroups.get(i.milestone) ?? [];
    arr.push(i);
    milestoneGroups.set(i.milestone, arr);
  }
  const milestoneOrder = [...milestoneGroups.keys()].sort();

  return (
    <>
      <div className="mx-auto max-w-4xl px-8 py-8">
        <Link
          href={`/prototype/initiative-runner?variant=${variant}`}
          className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="size-3.5" /> Back to monitor
        </Link>

        <header className="mt-4 flex items-start justify-between gap-6">
          <div>
            <div className="flex items-center gap-2 text-xs uppercase tracking-wider text-muted-foreground">
              <GitBranch className="size-3" /> {init.branchSlug} · {init.mode} mode
            </div>
            <h1 className="mt-2 font-serif text-3xl tracking-tight">{init.title}</h1>
          </div>
          <span className="rounded-full border border-border bg-card px-3 py-1 text-xs font-medium uppercase tracking-wider">
            {init.status.replace("_", " ")}
          </span>
        </header>

        {/* PRD */}
        <section className="mt-6 rounded-lg border border-border bg-card/40 p-5">
          <div className="flex items-center gap-2 text-xs uppercase tracking-wider text-muted-foreground">
            <BookOpen className="size-3" /> PRD
          </div>
          <p className="mt-2 leading-relaxed">{init.prd}</p>
          {init.prUrl && (
            <a
              href={init.prUrl}
              className="mt-3 inline-flex items-center gap-1.5 text-sm text-indigo-500 hover:underline"
            >
              <GitPullRequest className="size-3.5" /> View pull request
            </a>
          )}
        </section>

        {/* Progress at a glance */}
        <section className="mt-6 grid grid-cols-3 gap-3">
          <Stat label="Milestones" value={`${init.milestonesPassed}/${init.milestonesTotal}`} />
          <Stat label="Issues done" value={`${init.issuesDone}/${init.issuesTotal}`} />
          <Stat label="Last activity" value={init.lastActivityAt} />
        </section>

        {/* Milestones + issues */}
        <section className="mt-8">
          <h2 className="text-sm font-semibold uppercase tracking-wider text-muted-foreground">
            Milestones
          </h2>
          <div className="mt-3 space-y-5">
            {milestoneOrder.map((ms) => {
              const groupIssues = milestoneGroups.get(ms)!;
              const passed = groupIssues.every((i) => i.status === "done");
              return (
                <div key={ms}>
                  <div className="mb-2 flex items-center gap-2 text-sm font-medium">
                    {passed ? (
                      <CheckCircle2 className="size-4 text-emerald-500" />
                    ) : (
                      <Circle className="size-4 text-muted-foreground" />
                    )}
                    {ms}
                  </div>
                  <ul className="space-y-1.5">
                    {groupIssues.map((issue) => {
                      const agent = agentById(issue.assigneeId);
                      return (
                        <li key={issue.id}>
                          <Link
                            href={`/prototype/initiative-runner/initiatives/${init.id}/issues/${issue.id}?variant=${variant}`}
                            className="flex items-center gap-3 rounded-md border border-border bg-card px-3 py-2 text-sm transition hover:border-foreground/30"
                          >
                            <span className="font-mono text-xs text-muted-foreground">
                              #{issue.number}
                            </span>
                            <span className="flex-1">{issue.title}</span>
                            {issue.status === "in_progress" && (
                              <span className="inline-flex items-center gap-1.5 rounded-full bg-emerald-500/10 px-2 py-0.5 text-[11px] text-emerald-500">
                                <span className="size-1.5 animate-pulse rounded-full bg-emerald-500" />
                                running
                              </span>
                            )}
                            {issue.status === "blocked" && (
                              <span className="rounded-full bg-red-500/10 px-2 py-0.5 text-[11px] text-red-500">
                                blocked
                              </span>
                            )}
                            {agent && (
                              <span
                                className="size-2 rounded-full"
                                style={{ background: `hsl(${agent.hue} 70% 55%)` }}
                                title={agent.name}
                              />
                            )}
                            {issue.status === "done" && (
                              <CheckCircle2 className="size-3.5 text-emerald-500" />
                            )}
                          </Link>
                        </li>
                      );
                    })}
                  </ul>
                </div>
              );
            })}
          </div>
        </section>

        {/* Recent activity preview + decision-log link */}
        <section className="mt-8 grid gap-5 md:grid-cols-2">
          <div className="rounded-lg border border-border bg-card/40 p-5">
            <h3 className="text-sm font-semibold uppercase tracking-wider text-muted-foreground">
              Recent activity
            </h3>
            <ul className="mt-3 space-y-2">
              {activity.slice(0, 5).map((a) => (
                <li key={a.id} className="flex gap-2 text-xs">
                  <span className="w-12 shrink-0 tabular-nums text-muted-foreground">
                    {formatTsAgo(a.tsMinutesAgo)}
                  </span>
                  <span>{a.message}</span>
                </li>
              ))}
            </ul>
          </div>
          <Link
            href={`/prototype/initiative-runner/initiatives/${init.id}/decision-log?variant=${variant}`}
            className="flex flex-col rounded-lg border border-border bg-card/40 p-5 transition hover:border-foreground/30"
          >
            <div className="flex items-center gap-2 text-sm font-semibold uppercase tracking-wider text-muted-foreground">
              <ScrollText className="size-4" /> Decision log
            </div>
            <p className="mt-2 text-sm text-muted-foreground">
              Decisions and learnings recorded during execution. Reviewed by
              the retrospective on every Milestone closeout.
            </p>
            <span className="mt-auto pt-3 text-xs text-indigo-500">Read decisions →</span>
          </Link>
        </section>
      </div>
      <PrototypeSwitcher current={variant} />
    </>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border border-border bg-card/40 p-4">
      <div className="text-xs uppercase tracking-wider text-muted-foreground">{label}</div>
      <div className="mt-1 font-mono text-xl tabular-nums">{value}</div>
    </div>
  );
}
