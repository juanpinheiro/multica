"use client";

import { useQuery } from "@tanstack/react-query";
import { decisionLogOptions } from "@multica/core/decision-log/queries";
import { Skeleton } from "@multica/ui/components/ui/skeleton";
import { useT } from "../../i18n";
import { DecisionRow } from "./decision-row";

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
