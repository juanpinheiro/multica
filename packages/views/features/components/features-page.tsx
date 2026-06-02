"use client";

import { useCallback, useMemo, useState } from "react";
import { FolderKanban, Rows3, LayoutGrid, Search } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { featureListOptions } from "@multica/core/features/queries";
import { useUpdateFeature } from "@multica/core/features/mutations";
import { useWorkspaceId } from "@multica/core/hooks";
import { useWorkspacePaths } from "@multica/core/paths";
import { AppLink } from "../../navigation";
import { ActorAvatar } from "../../common/actor-avatar";
import { Skeleton } from "@multica/ui/components/ui/skeleton";
import { Input } from "@multica/ui/components/ui/input";
import { cn } from "@multica/ui/lib/utils";
import type { Feature, UpdateFeatureRequest } from "@multica/core/types";
import { PageHeader } from "../../layout/page-header";
import { FeatureIcon } from "./feature-icon";
import { useT } from "../../i18n";
import { matchesPinyin } from "../../editor/extensions/pinyin-match";
import { useFormatRelativeDate } from "./labels";
import { useFeatureViewStore } from "@multica/core/features";
import { FeatureStatusBadge, FeaturePriorityBadge } from "./feature-badge";
import { FeatureLeadPicker } from "./feature-lead-picker";

const COMPACT_GRID = "grid w-full min-w-[740px] grid-cols-[24px_minmax(200px,1fr)_96px_96px_80px_80px_80px]";

function FeatureCard({ feature }: { feature: Feature }) {
  const { t } = useT("features");
  const wsPaths = useWorkspacePaths();
  const formatRelativeDate = useFormatRelativeDate();
  const updateFeatureMut = useUpdateFeature();

  const handleUpdate = useCallback(
    (data: UpdateFeatureRequest) => {
      updateFeatureMut.mutate({ id: feature.id, ...data });
    },
    [feature.id, updateFeatureMut],
  );

  const progressPercent = feature.issue_count > 0 ? Math.round((feature.done_count / feature.issue_count) * 100) : 0;

  return (
    <div className="group/card flex flex-col rounded-md border bg-card hover:border-primary/50 transition-colors">
      <div className="p-3 pb-2">
        <div className="flex items-center gap-2">
          <AppLink
            href={wsPaths.featureDetail(feature.id)}
            className="flex items-center gap-2 min-w-0 flex-1"
          >
            <FeatureIcon feature={feature} size="sm" />
            <h3 className="font-medium text-sm truncate">{feature.title}</h3>
          </AppLink>
          <FeatureStatusBadge feature={feature} handleUpdate={handleUpdate} triggerClassName="shrink-0" />
        </div>

        {feature.issue_count > 0 ? (
          <div className="flex justify-end items-center gap-1.5 pt-2">
            <div className="relative h-4 w-4">
              <svg className="h-4 w-4 -rotate-90" viewBox="0 0 16 16">
                <circle
                  className="text-muted"
                  strokeWidth="2"
                  stroke="currentColor"
                  fill="none"
                  r="6"
                  cx="8"
                  cy="8"
                />
                <circle
                  className="text-emerald-500"
                  strokeWidth="2"
                  stroke="currentColor"
                  fill="none"
                  r="6"
                  cx="8"
                  cy="8"
                  strokeDasharray={`${progressPercent * 0.377} 37.7`}
                  strokeLinecap="round"
                />
              </svg>
            </div>
            <span className="text-[10px] text-muted-foreground tabular-nums">
              {feature.done_count}/{feature.issue_count}
            </span>
          </div>
        ) : (
          <span className="text-[10px] text-muted-foreground pt-2 flex justify-end">{t(($) => $.detail.no_issues_yet)}</span>
        )}
      </div>

      <div className="flex items-center justify-between px-3 pb-3 border-t mt-0 pt-2">
        <FeatureLeadPicker
          feature={feature}
          handleUpdate={handleUpdate}
          renderTrigger={(leadName) => (
            <button type="button" className="flex items-center gap-1.5 rounded px-1.5 py-0.5 -mx-1.5 hover:bg-accent/60 transition-colors cursor-pointer">
              {feature.lead_type && feature.lead_id ? (
                <ActorAvatar actorType={feature.lead_type} actorId={feature.lead_id} size={20} enableHoverCard />
              ) : (
                <span className="inline-flex h-5 w-5 rounded-full border border-dashed border-muted-foreground/30" />
              )}
              <span className="text-[10px] text-muted-foreground truncate max-w-[60px]">
                {leadName ?? t(($) => $.lead.no_lead)}
              </span>
            </button>
          )}
        />

        <div className="flex items-center gap-2">
          <FeaturePriorityBadge feature={feature} handleUpdate={handleUpdate} align="start" />
          <span className="text-[10px] text-muted-foreground">
            {formatRelativeDate(feature.created_at)}
          </span>
        </div>
      </div>
    </div>
  );
}

function FeatureCardCompact({ feature }: { feature: Feature }) {
  const wsPaths = useWorkspacePaths();
  const formatRelativeDate = useFormatRelativeDate();
  const updateFeatureMut = useUpdateFeature();

  const handleUpdate = useCallback(
    (data: UpdateFeatureRequest) => {
      updateFeatureMut.mutate({ id: feature.id, ...data });
    },
    [feature.id, updateFeatureMut],
  );

  return (
    <div className={cn(COMPACT_GRID, "h-10 items-center gap-2 px-4 text-sm transition-colors hover:bg-accent/40 border-b")}>
      <FeatureIcon feature={feature} size="sm" />
      <AppLink
        href={wsPaths.featureDetail(feature.id)}
        className="flex items-center justify-start gap-2 min-w-0 overflow-hidden"
      >
        <span className="font-medium truncate text-left">{feature.title}</span>
      </AppLink>

      <div className="flex items-center justify-start">
        <FeaturePriorityBadge feature={feature} handleUpdate={handleUpdate} align="start" />
      </div>

      <div className="flex items-center justify-start">
        <FeatureStatusBadge feature={feature} handleUpdate={handleUpdate} align="start" />
      </div>

      <span className="flex items-center justify-start gap-1.5 text-xs text-muted-foreground tabular-nums">
        {feature.issue_count > 0 ? `${feature.done_count}/${feature.issue_count}` : "--"}
      </span>

      <FeatureLeadPicker
        feature={feature}
        handleUpdate={handleUpdate}
        align="start"
        renderTrigger={(leadName) => (
          <button type="button" className="flex items-center justify-start gap-1.5 rounded px-1 py-0.5 hover:bg-accent/60 transition-colors cursor-pointer">
            <span className="shrink-0">
              {feature.lead_type && feature.lead_id ? (
                <ActorAvatar actorType={feature.lead_type} actorId={feature.lead_id} size={20} enableHoverCard />
              ) : (
                <span className="inline-flex h-5 w-5 rounded-full border border-dashed border-muted-foreground/30" />
              )}
            </span>
            <span className="text-xs text-muted-foreground truncate max-w-[50px]">
              {leadName ?? "--"}
            </span>
          </button>
        )}
      />

      <span className="text-left text-xs text-muted-foreground tabular-nums">
        {formatRelativeDate(feature.created_at)}
      </span>
    </div>
  );
}

export function FeaturesPage() {
  const { t } = useT("features");
  const wsId = useWorkspaceId();
  const viewMode = useFeatureViewStore((s) => s.viewMode);
  const setViewMode = useFeatureViewStore((s) => s.setViewMode);
  const isCompact = viewMode === "compact";
  const { data: features = [], isLoading } = useQuery(featureListOptions(wsId));
  const [search, setSearch] = useState("");
  const filteredProjects = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return features;
    return features.filter((p) =>
      p.title.toLowerCase().includes(q) || matchesPinyin(p.title, q)
    );
  }, [features, search]);

  return (
    <div className="flex flex-1 min-h-0 flex-col">
      <PageHeader className="justify-between px-5">
        <div className="flex items-center gap-2">
          <FolderKanban className="h-4 w-4 text-muted-foreground" />
          <h1 className="text-sm font-medium">{t(($) => $.page.title)}</h1>
          {!isLoading && features.length > 0 && (
            <span className="text-xs text-muted-foreground tabular-nums">{features.length}</span>
          )}
        </div>
      </PageHeader>

      <div className="flex flex-1 min-h-0 flex-col overflow-hidden">
        {(features.length > 0 || isLoading) && (
          <div className="flex h-12 shrink-0 items-center justify-between border-b px-4 gap-2 sm:gap-3">
            <div className="relative flex-1 sm:flex-none">
              <Search className="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
              <Input
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder={t(($) => $.page.search_placeholder)}
                className="h-8 w-full sm:w-64 pl-8 text-sm"
              />
            </div>

            <div className="flex items-center gap-2 sm:gap-4 shrink-0">
              <span className="hidden sm:inline-block font-mono text-xs tabular-nums text-muted-foreground/70">
                {filteredProjects.length} / {features.length}
              </span>
              <div className="flex items-center gap-0.5 rounded-md bg-muted p-0.5">
                <button
                  type="button"
                  onClick={() => setViewMode("compact")}
                  className={cn(
                    "inline-flex items-center gap-1.5 rounded p-1 sm:px-2.5 sm:py-1 text-xs font-medium transition-colors",
                    isCompact ? "bg-background text-foreground shadow-sm" : "text-muted-foreground hover:text-foreground"
                  )}
                >
                  <Rows3 className="size-3.5" />
                  <span className="hidden sm:inline-block">{t(($) => $.page.view_compact)}</span>
                </button>
                <button
                  type="button"
                  onClick={() => setViewMode("comfortable")}
                  className={cn(
                    "inline-flex items-center gap-1.5 rounded p-1 sm:px-2.5 sm:py-1 text-xs font-medium transition-colors",
                    !isCompact ? "bg-background text-foreground shadow-sm" : "text-muted-foreground hover:text-foreground"
                  )}
                >
                  <LayoutGrid className="size-3.5" />
                  <span className="hidden sm:inline-block">{t(($) => $.page.view_comfortable)}</span>
                </button>
              </div>
            </div>
          </div>
        )}

        <div key={viewMode} className={cn("flex-1", isCompact ? "overflow-hidden flex flex-col" : "overflow-y-auto")}>
          {isLoading ? (
            isCompact ? (
              <div className="pt-4 mx-5 overflow-x-auto rounded-md border pb-4 mb-5">
                <div className="min-w-[740px]">
                  <div className={cn(COMPACT_GRID, "h-10 items-center gap-2 px-4 border-b")}>
                    <Skeleton className="h-6 w-6 rounded" />
                    <Skeleton className="h-4 w-48" />
                  </div>
                  {Array.from({ length: 6 }).map((_, i) => (
                    <div key={i} className={cn(COMPACT_GRID, "h-10 items-center gap-2 px-4 border-b")}>
                      <Skeleton className="h-6 w-6 rounded" />
                      <Skeleton className="h-4 w-48" />
                    </div>
                  ))}
                </div>
              </div>
            ) : (
              <div className="pt-4 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3 px-5">
                {Array.from({ length: 8 }).map((_, i) => (
                  <div key={i} className="flex flex-col rounded-md border p-3 gap-2">
                    <div className="flex items-center gap-2">
                      <Skeleton className="h-8 w-8 rounded" />
                      <Skeleton className="h-4 w-3/4" />
                    </div>
                    <div className="flex gap-1.5">
                      <Skeleton className="h-5 w-16 rounded" />
                      <Skeleton className="h-5 w-20 rounded" />
                    </div>
                    <div className="flex items-center justify-between">
                      <Skeleton className="h-5 w-5 rounded-full" />
                      <Skeleton className="h-3 w-12" />
                    </div>
                  </div>
                ))}
              </div>
            )
          ) : features.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-24 text-muted-foreground">
              <FolderKanban className="h-10 w-10 mb-3 opacity-30" />
              <p className="text-sm">{t(($) => $.page.empty)}</p>
            </div>
          ) : filteredProjects.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-24 text-muted-foreground">
              <Search className="h-10 w-10 mb-3 opacity-30" />
              <p className="text-sm">{t(($) => $.page.no_search_results)}</p>
            </div>
          ) : isCompact ? (
            <div className="mt-4 mx-5 rounded-md border mb-5 overflow-auto flex-1">
              <div className="min-w-[740px]">
                <div className={cn(COMPACT_GRID, "h-8 shrink-0 items-center gap-2 px-4 text-xs font-medium text-muted-foreground border-b bg-muted/30 sticky top-0 z-10")}>
                  <span />
                  <span className="text-left">{t(($) => $.table.name)}</span>
                  <span className="text-left">{t(($) => $.table.priority)}</span>
                  <span className="text-left">{t(($) => $.table.status)}</span>
                  <span className="text-left">{t(($) => $.table.progress)}</span>
                  <span className="text-left">{t(($) => $.table.lead)}</span>
                  <span className="text-left">{t(($) => $.table.created)}</span>
                </div>
                <div className="pb-4">
                  {filteredProjects.map((feature) => (
                    <FeatureCardCompact key={feature.id} feature={feature} />
                  ))}
                </div>
              </div>
            </div>
          ) : (
            <div className="pt-4 pb-5 px-5 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
              {filteredProjects.map((feature) => (
                <FeatureCard key={feature.id} feature={feature} />
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
