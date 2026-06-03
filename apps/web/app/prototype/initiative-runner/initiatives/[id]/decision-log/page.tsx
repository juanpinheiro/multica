import Link from "next/link";
import { notFound } from "next/navigation";
import { ArrowLeft, BookMarked, ScrollText } from "lucide-react";
import {
  decisionsByInitiative,
  formatTsAgo,
  initiativeById,
} from "../../../_data";
import { PrototypeSwitcher } from "../../../_switcher";

export const dynamic = "force-dynamic";

export default async function DecisionLogPage({
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

  const decisions = decisionsByInitiative(init.id);

  return (
    <>
      <div className="mx-auto max-w-3xl px-8 py-8">
        <Link
          href={`/prototype/initiative-runner/initiatives/${init.id}?variant=${variant}`}
          className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="size-3.5" /> {init.title}
        </Link>

        <header className="mt-4 flex items-center gap-2">
          <ScrollText className="size-5" />
          <h1 className="font-serif text-3xl tracking-tight">Decision log</h1>
        </header>
        <p className="mt-1 text-sm text-muted-foreground">
          Captured by the retrospective Run at every Milestone closeout.
          Each entry pairs a decision with the learning that made it stick.
        </p>

        {decisions.length === 0 && (
          <div className="mt-8 rounded-lg border border-dashed border-border p-6 text-center text-sm text-muted-foreground">
            No decisions recorded yet — they appear once a Milestone closes.
          </div>
        )}

        <ol className="mt-6 space-y-4">
          {decisions.map((d) => (
            <li
              key={d.id}
              className="rounded-lg border border-border bg-card/40 p-5"
            >
              <div className="flex items-baseline justify-between gap-3">
                <h2 className="text-lg font-semibold leading-snug">{d.title}</h2>
                <span className="shrink-0 text-xs tabular-nums text-muted-foreground">
                  {formatTsAgo(d.tsMinutesAgo)}
                </span>
              </div>
              <div className="mt-3 grid gap-3 md:grid-cols-2">
                <div>
                  <div className="text-[11px] uppercase tracking-wider text-muted-foreground">
                    Decision
                  </div>
                  <p className="mt-1 text-sm leading-relaxed">{d.decision}</p>
                </div>
                <div>
                  <div className="text-[11px] uppercase tracking-wider text-muted-foreground">
                    Learning
                  </div>
                  <p className="mt-1 text-sm leading-relaxed">{d.learning}</p>
                </div>
              </div>
              {(d.adrRefs?.length || d.contextTerms?.length) && (
                <div className="mt-4 flex flex-wrap gap-2 border-t border-border pt-3">
                  {d.adrRefs?.map((ref) => (
                    <span
                      key={ref}
                      className="inline-flex items-center gap-1 rounded border border-border bg-card px-2 py-0.5 text-[11px]"
                    >
                      <BookMarked className="size-3" /> {ref}
                    </span>
                  ))}
                  {d.contextTerms?.map((t) => (
                    <span
                      key={t}
                      className="inline-flex items-center rounded bg-muted px-2 py-0.5 text-[11px] text-muted-foreground"
                    >
                      {t}
                    </span>
                  ))}
                </div>
              )}
            </li>
          ))}
        </ol>
      </div>
      <PrototypeSwitcher current={variant} />
    </>
  );
}
