"use client";

import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { GitMerge } from "lucide-react";
import { useWorkspaceId } from "@multica/core/hooks";
import { useWorkspacePaths } from "@multica/core/paths";
import { workspaceDecisionsOptions } from "@multica/core/decision-log/queries";
import { featureListOptions } from "@multica/core/features/queries";
import type { DecisionLogEntry, Feature } from "@multica/core/types";
import { Skeleton } from "@multica/ui/components/ui/skeleton";
import { PageHeader } from "../../layout/page-header";
import { useT } from "../../i18n";
import { DecisionRow } from "../../features/components/decision-row";

function featureChipFor(
  entry: DecisionLogEntry,
  featuresById: Map<string, Feature>,
  href: (featureId: string) => string,
) {
  const feature = featuresById.get(entry.feature_id);
  if (!feature) return undefined;
  return { title: feature.title, href: href(feature.id) };
}

export function DecisionsPage() {
  const { t } = useT("layout");
  const wsId = useWorkspaceId();
  const paths = useWorkspacePaths();

  const { data: decisions = [], isLoading } = useQuery(workspaceDecisionsOptions(wsId));
  const { data: features = [] } = useQuery(featureListOptions(wsId));

  const featuresById = useMemo(
    () => new Map(features.map((f) => [f.id, f])),
    [features],
  );

  return (
    <div className="flex flex-1 min-h-0 flex-col">
      <PageHeader className="gap-1.5">
        <GitMerge className="size-4 text-muted-foreground" />
        <span className="text-sm font-medium">{t(($) => $.nav.decisions)}</span>
      </PageHeader>

      <div className="flex-1 overflow-y-auto">
        <div className="mx-auto flex max-w-3xl flex-col gap-6 px-6 py-8">
          <header>
            <h1 className="text-3xl font-semibold tracking-tight">
              {t(($) => $.decisions_page.headline)}
            </h1>
            <p className="mt-1 text-sm text-muted-foreground">
              {t(($) => $.decisions_page.subhead)}
            </p>
          </header>

          {isLoading ? (
            <DecisionsSkeleton />
          ) : decisions.length === 0 ? (
            <div
              data-testid="decisions-empty"
              className="rounded-lg border border-dashed border-border p-8 text-center"
            >
              <p className="text-sm font-medium">{t(($) => $.decisions_page.empty_title)}</p>
              <p className="mt-1 text-xs text-muted-foreground">
                {t(($) => $.decisions_page.empty_hint)}
              </p>
            </div>
          ) : (
            <ul data-testid="decisions-list" className="space-y-5">
              {decisions.map((entry) => (
                <li key={entry.id}>
                  <DecisionRow
                    entry={entry}
                    featureChip={featureChipFor(entry, featuresById, paths.initiativeDetail)}
                  />
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </div>
  );
}

function DecisionsSkeleton() {
  return (
    <div data-testid="decisions-loading" className="space-y-4">
      {[0, 1, 2].map((i) => (
        <div key={i} className="space-y-2 border-l-2 border-border pl-3">
          <Skeleton className="h-4 w-40" />
          <Skeleton className="h-3 w-72" />
          <Skeleton className="h-3 w-56" />
        </div>
      ))}
    </div>
  );
}
