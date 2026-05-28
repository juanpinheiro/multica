"use client";

import { useMemo, useState, useCallback, useRef } from "react";
import { useDefaultLayout, usePanelRef } from "react-resizable-panels";
import { Check, ChevronRight, GitBranch, Link2, MoreHorizontal, PanelRight, Pin, PinOff, Trash2, UserMinus } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { cn } from "@multica/ui/lib/utils";
import { toast } from "sonner";
import type { FeatureStatus, FeaturePriority, FeatureIssueSummary, FeatureBlockedIssueSummary, FeaturePRSummary } from "@multica/core/types";
import { useAuthStore } from "@multica/core/auth";
import { featureDetailOptions, featureIssuesOptions } from "@multica/core/features/queries";
import { useUpdateFeature, useDeleteFeature } from "@multica/core/features/mutations";
import { pinListOptions } from "@multica/core/pins";
import { useCreatePin, useDeletePin } from "@multica/core/pins";
import { memberListOptions, agentListOptions } from "@multica/core/workspace/queries";
import { useWorkspaceId } from "@multica/core/hooks";
import { useCurrentWorkspace, useWorkspacePaths } from "@multica/core/paths";
import { useActorName } from "@multica/core/workspace/hooks";
import { FEATURE_STATUS_ORDER, FEATURE_STATUS_CONFIG, FEATURE_PRIORITY_ORDER } from "@multica/core/features/config";
import { STATUS_CONFIG } from "@multica/core/issues/config";
import { ActorAvatar } from "../../common/actor-avatar";
import { AppLink, useNavigation } from "../../navigation";
import { TitleEditor, ContentEditor, type ContentEditorRef } from "../../editor";
import { PriorityIcon } from "../../issues/components/priority-icon";
import { StatusIcon } from "../../issues/components/status-icon";
import { FeatureResourcesSection } from "./feature-resources-section";
import { Skeleton } from "@multica/ui/components/ui/skeleton";
import { Button } from "@multica/ui/components/ui/button";
import { ResizablePanelGroup, ResizablePanel, ResizableHandle } from "@multica/ui/components/ui/resizable";
import { Sheet, SheetContent } from "@multica/ui/components/ui/sheet";
import { useIsMobile } from "@multica/ui/hooks/use-mobile";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@multica/ui/components/ui/dropdown-menu";
import {
  Popover,
  PopoverTrigger,
  PopoverContent,
} from "@multica/ui/components/ui/popover";
import {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from "@multica/ui/components/ui/tooltip";
import { EmojiPicker } from "@multica/ui/components/common/emoji-picker";
import { PageHeader } from "../../layout/page-header";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@multica/ui/components/ui/alert-dialog";
import { useT } from "../../i18n";
import { useFeatureStatusLabels, useFeaturePriorityLabels } from "./labels";
import { matchesPinyin } from "../../editor/extensions/pinyin-match";
import { getFeatureIssueMetrics } from "./feature-issue-metrics";

// ---------------------------------------------------------------------------
// Property row — sidebar property display
// ---------------------------------------------------------------------------

function PropRow({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex min-h-8 items-center gap-2 rounded-md px-2 -mx-2 hover:bg-accent/50 transition-colors">
      <span className="w-16 shrink-0 text-xs text-muted-foreground">{label}</span>
      <div className="flex min-w-0 flex-1 items-center gap-1.5 text-xs truncate">
        {children}
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Issue pipeline — ready/blocked two-section view
// ---------------------------------------------------------------------------

function IssueRow({ issue, blockedBy }: { issue: FeatureIssueSummary; blockedBy?: string[] }) {
  const statusCfg = STATUS_CONFIG[issue.status] ?? STATUS_CONFIG.backlog;
  return (
    <div className="flex items-start gap-2 rounded-md px-2 py-1.5 -mx-2 hover:bg-accent/40 transition-colors">
      <StatusIcon status={issue.status} className="h-3.5 w-3.5 mt-0.5 shrink-0" />
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-1.5 min-w-0">
          <span className="text-xs font-medium text-muted-foreground shrink-0">{issue.identifier}</span>
          <span className="text-xs truncate">{issue.title}</span>
          {issue.status === "in_progress" && (
            <span className={cn("ml-auto shrink-0 inline-flex items-center rounded-full px-1.5 py-0.5 text-[10px] font-medium", statusCfg.hoverBg, "text-foreground")}>
              running
            </span>
          )}
        </div>
        {blockedBy && blockedBy.length > 0 && (
          <p className="mt-0.5 text-[11px] text-muted-foreground">
            blocked by {blockedBy.join(", ")}
          </p>
        )}
      </div>
      <PriorityIcon priority={issue.priority} />
    </div>
  );
}

function IssueSection({
  title,
  issues,
  blockedMap,
}: {
  title: string;
  issues: FeatureIssueSummary[];
  blockedMap?: Map<string, string[]>;
}) {
  return (
    <div>
      <div className="flex items-center gap-1.5 mb-1">
        <span className="text-xs font-medium text-muted-foreground">{title}</span>
        <span className="text-xs text-muted-foreground/60">({issues.length})</span>
      </div>
      {issues.map((issue) => (
        <IssueRow
          key={issue.id}
          issue={issue}
          blockedBy={blockedMap?.get(issue.id)}
        />
      ))}
    </div>
  );
}

function FeatureIssuePipelineView({
  readyNow,
  blocked,
}: {
  readyNow: FeatureIssueSummary[];
  blocked: FeatureBlockedIssueSummary[];
}) {
  const { t } = useT("features");

  const blockedMap = useMemo(() => {
    const m = new Map<string, string[]>();
    for (const b of blocked) {
      m.set(b.id, b.blocked_by);
    }
    return m;
  }, [blocked]);

  if (readyNow.length === 0 && blocked.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center gap-2 py-8 text-muted-foreground">
        <p className="text-sm">{t(($) => $.detail.empty_issues_title)}</p>
        <p className="text-xs">{t(($) => $.detail.empty_issues_hint)}</p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <IssueSection
        title={t(($) => $.detail.section_ready_now)}
        issues={readyNow}
      />
      {blocked.length > 0 && (
        <IssueSection
          title={t(($) => $.detail.section_blocked)}
          issues={blocked}
          blockedMap={blockedMap}
        />
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// PR header badge
// ---------------------------------------------------------------------------

function PRHeaderBadge({ prs }: { prs: FeaturePRSummary[] }) {
  const openPRs = prs.filter((pr) => pr.state === "open" || pr.state === "draft");
  if (openPRs.length === 0) return null;

  const first = openPRs[0];
  if (openPRs.length === 1 && first) {
    return (
      <a
        href={first.html_url}
        target="_blank"
        rel="noopener noreferrer"
        className="inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-xs bg-accent hover:bg-accent/80 transition-colors text-foreground"
        data-testid="pr-link"
      >
        {`PR #${first.number}`}
      </a>
    );
  }

  return (
    <span className="inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-xs bg-accent text-foreground" data-testid="pr-count">
      {`${openPRs.length} PRs`}
    </span>
  );
}

// ---------------------------------------------------------------------------
// FeatureDetail
// ---------------------------------------------------------------------------

export function FeatureDetail({ featureId }: { featureId: string }) {
  const { t } = useT("features");
  const statusLabels = useFeatureStatusLabels();
  const priorityLabels = useFeaturePriorityLabels();
  const wsId = useWorkspaceId();
  const wsPaths = useWorkspacePaths();
  const router = useNavigation();
  const userId = useAuthStore((s) => s.user?.id);
  const workspace = useCurrentWorkspace();
  const workspaceName = workspace?.name;
  const { data: feature, isLoading } = useQuery(featureDetailOptions(wsId, featureId));
  const { data: featureIssues } = useQuery(featureIssuesOptions(wsId, featureId));

  const { data: members = [] } = useQuery(memberListOptions(wsId));
  const { data: agents = [] } = useQuery(agentListOptions(wsId));
  const { getActorName } = useActorName();
  const updateFeatureMut = useUpdateFeature();
  const deleteFeatureMut = useDeleteFeature();
  const { data: pinnedItems = [] } = useQuery({
    ...pinListOptions(wsId, userId ?? ""),
    enabled: !!userId,
  });
  const isPinned = pinnedItems.some((p) => p.item_type === "feature" && p.item_id === featureId);
  const createPin = useCreatePin();
  const deletePinMut = useDeletePin();
  const descEditorRef = useRef<ContentEditorRef>(null);
  const isMobile = useIsMobile();
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [iconPickerOpen, setIconPickerOpen] = useState(false);
  const [propertiesOpen, setPropertiesOpen] = useState(true);
  const [progressOpen, setProgressOpen] = useState(true);

  const { defaultLayout, onLayoutChanged } = useDefaultLayout({
    id: "multica_project_detail_layout",
  });
  const sidebarRef = usePanelRef();
  const [desktopSidebarOpen, setDesktopSidebarOpen] = useState(true);
  const [mobileSidebarOpen, setMobileSidebarOpen] = useState(false);
  const sidebarOpen = isMobile ? mobileSidebarOpen : desktopSidebarOpen;

  const [leadOpen, setLeadOpen] = useState(false);
  const [leadFilter, setLeadFilter] = useState("");
  const leadQuery = leadFilter.toLowerCase();
  const filteredMembers = members.filter((m) => m.name.toLowerCase().includes(leadQuery) || matchesPinyin(m.name, leadQuery));
  const filteredAgents = agents.filter((a) => !a.archived_at && (a.name.toLowerCase().includes(leadQuery) || matchesPinyin(a.name, leadQuery)));

  const handleUpdateField = useCallback(
    (data: Parameters<typeof updateFeatureMut.mutate>[0] extends { id: string } & infer R ? R : never) => {
      if (!feature) return;
      updateFeatureMut.mutate({ id: feature.id, ...data });
    },
    [feature, updateFeatureMut],
  );

  const handleDelete = useCallback(() => {
    if (!feature) return;
    deleteFeatureMut.mutate(feature.id, {
      onSuccess: () => {
        toast.success(t(($) => $.detail.toast_feature_deleted));
        router.push(wsPaths.features());
      },
    });
  }, [feature, deleteFeatureMut, router, wsPaths, t]);

  const handleApprove = useCallback(() => {
    if (!feature) return;
    updateFeatureMut.mutate({ id: feature.id, status: "in_progress" });
  }, [feature, updateFeatureMut]);

  if (isLoading) {
    return (
      <div className="mx-auto w-full max-w-4xl px-8 py-10 space-y-4">
        <Skeleton className="h-5 w-32" />
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-4 w-96" />
        <Skeleton className="h-40 w-full mt-8" />
      </div>
    );
  }

  if (!feature) {
    return <div className="flex items-center justify-center h-full text-muted-foreground">{t(($) => $.detail.not_found)}</div>;
  }

  const issueMetrics = getFeatureIssueMetrics(feature);
  const statusCfg = FEATURE_STATUS_CONFIG[feature.status];
  const prs: FeaturePRSummary[] = featureIssues?.pull_requests ?? [];
  const readyNow: FeatureIssueSummary[] = featureIssues?.ready_now ?? [];
  const blocked: FeatureBlockedIssueSummary[] = featureIssues?.blocked ?? [];

  const sidebarContent = (
    <div className="space-y-5">
      {/* Icon + Title */}
      <div>
        <Popover open={iconPickerOpen} onOpenChange={setIconPickerOpen}>
          <PopoverTrigger
            render={
              <button
                type="button"
                className="text-2xl cursor-pointer rounded-lg p-1 -ml-1 hover:bg-accent/60 transition-colors"
                title={t(($) => $.detail.icon_tooltip)}
              >
                {feature.icon || "📁"}
              </button>
            }
          />
          <PopoverContent align="start" className="w-auto p-0">
            <EmojiPicker
              onSelect={(emoji) => {
                handleUpdateField({ icon: emoji });
                setIconPickerOpen(false);
              }}
            />
          </PopoverContent>
        </Popover>
        <TitleEditor
          key={`title-${featureId}`}
          defaultValue={feature.title}
          placeholder={t(($) => $.detail.title_placeholder)}
          className="mt-2 w-full text-base font-semibold leading-snug tracking-tight"
          onBlur={(value) => {
            const trimmed = value.trim();
            if (trimmed && trimmed !== feature.title) handleUpdateField({ title: trimmed });
          }}
        />
      </div>

      {/* Properties */}
      <div>
        <button
          className={`flex w-full items-center gap-1 rounded-md px-2 py-1 text-xs font-medium transition-colors mb-2 hover:bg-accent/70 ${propertiesOpen ? "" : "text-muted-foreground hover:text-foreground"}`}
          onClick={() => setPropertiesOpen(!propertiesOpen)}
        >
          {t(($) => $.detail.section_properties)}
          <ChevronRight className={`!size-3 shrink-0 stroke-[2.5] text-muted-foreground transition-transform ${propertiesOpen ? "rotate-90" : ""}`} />
        </button>
        {propertiesOpen && <div className="space-y-0.5 pl-2">
          <PropRow label={t(($) => $.table.status)}>
            <DropdownMenu>
              <DropdownMenuTrigger
                render={
                  <button type="button" className="inline-flex items-center gap-1.5 text-xs hover:text-foreground transition-colors">
                    <span className={cn("size-2 rounded-full", statusCfg.dotColor)} />
                    <span>{statusLabels[feature.status]}</span>
                  </button>
                }
              />
              <DropdownMenuContent align="start" className="w-44">
                {FEATURE_STATUS_ORDER.map((s) => (
                  <DropdownMenuItem key={s} onClick={() => handleUpdateField({ status: s as FeatureStatus })}>
                    <span className={cn("size-2 rounded-full", FEATURE_STATUS_CONFIG[s].dotColor)} />
                    <span>{statusLabels[s]}</span>
                    {s === feature.status && <Check className="ml-auto h-3.5 w-3.5" />}
                  </DropdownMenuItem>
                ))}
              </DropdownMenuContent>
            </DropdownMenu>
          </PropRow>
          <PropRow label={t(($) => $.table.priority)}>
            <DropdownMenu>
              <DropdownMenuTrigger
                render={
                  <button type="button" className="inline-flex items-center gap-1.5 text-xs hover:text-foreground transition-colors">
                    <PriorityIcon priority={feature.priority} />
                    <span>{priorityLabels[feature.priority]}</span>
                  </button>
                }
              />
              <DropdownMenuContent align="start" className="w-44">
                {FEATURE_PRIORITY_ORDER.map((p) => (
                  <DropdownMenuItem key={p} onClick={() => handleUpdateField({ priority: p as FeaturePriority })}>
                    <PriorityIcon priority={p} />
                    <span>{priorityLabels[p]}</span>
                    {p === feature.priority && <Check className="ml-auto h-3.5 w-3.5" />}
                  </DropdownMenuItem>
                ))}
              </DropdownMenuContent>
            </DropdownMenu>
          </PropRow>
          <PropRow label={t(($) => $.table.lead)}>
            <Popover open={leadOpen} onOpenChange={(v) => { setLeadOpen(v); if (!v) setLeadFilter(""); }}>
              <PopoverTrigger
                render={
                  <button type="button" className="inline-flex items-center gap-1.5 text-xs hover:text-foreground transition-colors">
                    {feature.lead_type && feature.lead_id ? (
                      <>
                        <ActorAvatar actorType={feature.lead_type} actorId={feature.lead_id} size={16} enableHoverCard showStatusDot />
                        <span className="cursor-pointer">{getActorName(feature.lead_type, feature.lead_id)}</span>
                      </>
                    ) : (
                      <span className="text-muted-foreground">{t(($) => $.lead.no_lead)}</span>
                    )}
                  </button>
                }
              />
              <PopoverContent align="start" className="w-52 p-0">
                <div className="px-2 py-1.5 border-b">
                  <input
                    type="text"
                    value={leadFilter}
                    onChange={(e) => setLeadFilter(e.target.value)}
                    placeholder={t(($) => $.lead.assign_placeholder)}
                    className="w-full bg-transparent text-sm placeholder:text-muted-foreground outline-none"
                  />
                </div>
                <div className="p-1 max-h-60 overflow-y-auto">
                  <button
                    type="button"
                    onClick={() => { handleUpdateField({ lead_type: null, lead_id: null }); setLeadOpen(false); }}
                    className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm hover:bg-accent transition-colors"
                  >
                    <UserMinus className="h-3.5 w-3.5 text-muted-foreground" />
                    <span className="text-muted-foreground">{t(($) => $.lead.no_lead)}</span>
                  </button>
                  {filteredMembers.length > 0 && (
                    <>
                      <div className="px-2 pt-2 pb-1 text-xs font-medium text-muted-foreground uppercase tracking-wider">{t(($) => $.lead.members_group)}</div>
                      {filteredMembers.map((m) => (
                        <button
                          type="button"
                          key={m.user_id}
                          onClick={() => { handleUpdateField({ lead_type: "member", lead_id: m.user_id }); setLeadOpen(false); }}
                          className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm hover:bg-accent transition-colors"
                        >
                          <ActorAvatar actorType="member" actorId={m.user_id} size={16} />
                          <span>{m.name}</span>
                        </button>
                      ))}
                    </>
                  )}
                  {filteredAgents.length > 0 && (
                    <>
                      <div className="px-2 pt-2 pb-1 text-xs font-medium text-muted-foreground uppercase tracking-wider">{t(($) => $.lead.agents_group)}</div>
                      {filteredAgents.map((a) => (
                        <button
                          type="button"
                          key={a.id}
                          onClick={() => { handleUpdateField({ lead_type: "agent", lead_id: a.id }); setLeadOpen(false); }}
                          className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm hover:bg-accent transition-colors"
                        >
                          <ActorAvatar actorType="agent" actorId={a.id} size={16} showStatusDot />
                          <span>{a.name}</span>
                        </button>
                      ))}
                    </>
                  )}
                  {filteredMembers.length === 0 && filteredAgents.length === 0 && leadFilter && (
                    <div className="px-2 py-3 text-center text-sm text-muted-foreground">{t(($) => $.lead.no_results)}</div>
                  )}
                </div>
              </PopoverContent>
            </Popover>
          </PropRow>
        </div>}
      </div>

      {/* Progress */}
      {issueMetrics.totalCount > 0 && (() => {
        const pct = Math.round((issueMetrics.completedCount / issueMetrics.totalCount) * 100);
        return (
          <div>
            <button
              className={`flex w-full items-center gap-1 rounded-md px-2 py-1 text-xs font-medium transition-colors mb-2 hover:bg-accent/70 ${progressOpen ? "" : "text-muted-foreground hover:text-foreground"}`}
              onClick={() => setProgressOpen(!progressOpen)}
            >
              {t(($) => $.detail.section_progress)}
              <ChevronRight className={`!size-3 shrink-0 stroke-[2.5] text-muted-foreground transition-transform ${progressOpen ? "rotate-90" : ""}`} />
            </button>
            {progressOpen && <div className="pl-2 flex items-center gap-3">
              <div className="relative h-2 flex-1 rounded-full bg-muted overflow-hidden">
                <div
                  className="absolute inset-y-0 left-0 rounded-full bg-emerald-500 transition-all"
                  style={{ width: `${pct}%` }}
                />
              </div>
              <span className="text-xs text-muted-foreground tabular-nums shrink-0">
                {issueMetrics.completedCount}/{issueMetrics.totalCount}
              </span>
            </div>}
          </div>
        );
      })()}

      {/* Resources */}
      <FeatureResourcesSection featureId={featureId} />
    </div>
  );

  return (
    <>
      <ResizablePanelGroup orientation="horizontal" className="flex-1 min-h-0" defaultLayout={defaultLayout} onLayoutChanged={onLayoutChanged}>
        <ResizablePanel id="content" minSize="50%">
          <div className="flex h-full flex-col">
            <PageHeader className="gap-2 bg-background text-sm">
              <div className="flex flex-1 items-center gap-1.5 min-w-0 flex-wrap">
                <AppLink href={wsPaths.features()} className="text-muted-foreground hover:text-foreground transition-colors shrink-0">
                  {workspaceName ?? t(($) => $.detail.breadcrumb_fallback)}
                </AppLink>
                <ChevronRight className="h-3 w-3 text-muted-foreground/50 shrink-0" />
                <span className="truncate">{feature.title}</span>
                {feature.target_branch && (
                  <span className="inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-xs bg-accent text-foreground shrink-0" data-testid="branch-indicator">
                    <GitBranch className="h-3 w-3" />
                    {feature.target_branch}
                  </span>
                )}
                <PRHeaderBadge prs={prs} />
              </div>
              <div className="flex items-center gap-1 shrink-0">
                <Button
                  variant="ghost"
                  size="icon-sm"
                  className={cn("text-muted-foreground", isPinned && "text-foreground")}
                  title={isPinned ? t(($) => $.detail.unpin_tooltip) : t(($) => $.detail.pin_tooltip)}
                  onClick={() => {
                    if (isPinned) {
                      deletePinMut.mutate({ itemType: "feature", itemId: featureId });
                    } else {
                      createPin.mutate({ item_type: "feature", item_id: featureId });
                    }
                  }}
                >
                  {isPinned ? <PinOff /> : <Pin />}
                </Button>
                <DropdownMenu>
                  <DropdownMenuTrigger
                    render={
                      <Button variant="ghost" size="icon-sm" className="text-muted-foreground">
                        <MoreHorizontal />
                      </Button>
                    }
                  />
                  <DropdownMenuContent align="end" className="w-auto">
                    <DropdownMenuItem onClick={() => {
                      navigator.clipboard.writeText(window.location.href);
                      toast.success(t(($) => $.detail.toast_link_copied));
                    }}>
                      <Link2 className="h-3.5 w-3.5" />
                      {t(($) => $.detail.copy_link)}
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem
                      variant="destructive"
                      onClick={() => setDeleteDialogOpen(true)}
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                      {t(($) => $.detail.delete_action)}
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
                <Tooltip>
                  <TooltipTrigger
                    render={
                      <Button
                        variant={sidebarOpen ? "secondary" : "ghost"}
                        size="icon-sm"
                        className={sidebarOpen ? "" : "text-muted-foreground"}
                        onClick={() => {
                          if (isMobile) {
                            setMobileSidebarOpen((open) => !open);
                          } else {
                            const panel = sidebarRef.current;
                            if (!panel) return;
                            if (panel.isCollapsed()) panel.expand();
                            else panel.collapse();
                          }
                        }}
                      >
                        <PanelRight />
                      </Button>
                    }
                  />
                  <TooltipContent side="bottom">{t(($) => $.detail.sidebar_tooltip)}</TooltipContent>
                </Tooltip>
              </div>
            </PageHeader>

            {/* Main scrollable content */}
            <div className="flex-1 overflow-y-auto">
              <div className="px-6 py-5 max-w-3xl space-y-6">
                {/* Approve button — only when planned */}
                {feature.status === "planned" && (
                  <Button
                    data-testid="approve-button"
                    onClick={handleApprove}
                    disabled={updateFeatureMut.isPending}
                  >
                    {t(($) => $.detail.approve_button)}
                  </Button>
                )}

                {/* Description — primary content */}
                <ContentEditor
                  ref={descEditorRef}
                  key={featureId}
                  defaultValue={feature.description || ""}
                  placeholder={t(($) => $.detail.description_placeholder)}
                  onUpdate={(md) => handleUpdateField({ description: md || null })}
                  debounceMs={1500}
                />

                {/* Issues pipeline */}
                <div>
                  <h3 className="text-sm font-medium mb-3">{t(($) => $.detail.section_issues)}</h3>
                  <FeatureIssuePipelineView
                    readyNow={readyNow}
                    blocked={blocked}
                  />
                </div>
              </div>
            </div>
          </div>
        </ResizablePanel>
        {!isMobile && <ResizableHandle />}
        {!isMobile && (
          <ResizablePanel
            id="sidebar"
            defaultSize={desktopSidebarOpen ? 320 : 0}
            minSize={260}
            maxSize={420}
            collapsible
            groupResizeBehavior="preserve-pixel-size"
            panelRef={sidebarRef}
            onResize={(size) => setDesktopSidebarOpen(size.inPixels > 0)}
          >
            <div className="overflow-y-auto border-l h-full">
              <div className="p-4">
                {sidebarContent}
              </div>
            </div>
          </ResizablePanel>
        )}
        {isMobile && (
          <Sheet open={mobileSidebarOpen} onOpenChange={setMobileSidebarOpen}>
            <SheetContent side="right" showCloseButton={false} className="w-[320px] overflow-y-auto p-4">
              {sidebarContent}
            </SheetContent>
          </Sheet>
        )}
      </ResizablePanelGroup>

      {/* Delete confirmation */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t(($) => $.delete_dialog.title)}</AlertDialogTitle>
            <AlertDialogDescription>
              {t(($) => $.delete_dialog.description)}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t(($) => $.delete_dialog.cancel)}</AlertDialogCancel>
            <AlertDialogAction onClick={handleDelete} className="bg-destructive text-white hover:bg-destructive/90">
              {t(($) => $.delete_dialog.confirm)}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
