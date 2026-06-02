"use client";

import { useQuery } from "@tanstack/react-query";
import type { DecisionLogEntry } from "@multica/core/types";
import { decisionLogOptions } from "@multica/core/decision-log/queries";
import { cn } from "@multica/ui/lib/utils";
import { Skeleton } from "@multica/ui/components/ui/skeleton";
import { useT } from "../../i18n";

function RefChip({ children, className }: { children: React.ReactNode; className?: string }) {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-sm border px-1 py-px text-[10px] leading-4 text-muted-foreground",
        className,
      )}
    >
      {children}
    </span>
  );
}

function DecisionRow({ entry }: { entry: DecisionLogEntry }) {
  const { t } = useT("features");
  return (
    <div data-testid={`decision-row-${entry.id}`} className="space-y-1 border-l-2 border-border pl-3">
      <p className="text-sm font-medium leading-snug">{entry.title}</p>
      <p className="text-xs leading-snug text-foreground/80">{entry.decision}</p>
      {entry.learning ? (
        <p className="text-xs leading-snug text-muted-foreground">
          <span className="font-medium">{t(($) => $.detail.decision_learned)}</span> {entry.learning}
        </p>
      ) : null}
      {(entry.adr_refs.length > 0 || entry.context_terms.length > 0) && (
        <div className="flex flex-wrap gap-1 pt-0.5">
          {entry.adr_refs.map((ref) => (
            <RefChip key={`adr-${ref}`} className="border-primary/30 text-primary/80">
              ADR-{ref}
            </RefChip>
          ))}
          {entry.context_terms.map((term) => (
            <RefChip key={`term-${term}`}>{term}</RefChip>
          ))}
        </div>
      )}
    </div>
  );
}

export function DecisionLogSection({ featureId }: { featureId: string }) {
  const { t } = useT("features");
  const { data: decisions = [], isLoading } = useQuery(decisionLogOptions(featureId));

  if (isLoading) {
    return (
      <div className="space-y-2">
        <Skeleton className="h-4 w-40" />
        <Skeleton className="h-4 w-56" />
      </div>
    );
  }

  if (decisions.length === 0) {
    return <p className="text-xs text-muted-foreground">{t(($) => $.detail.decisions_empty)}</p>;
  }

  return (
    <div className="space-y-3">
      {decisions.map((entry) => (
        <DecisionRow key={entry.id} entry={entry} />
      ))}
    </div>
  );
}
