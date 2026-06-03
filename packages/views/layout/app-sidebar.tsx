"use client";

import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { cn } from "@multica/ui/lib/utils";
import { useScrollFade } from "@multica/ui/hooks/use-scroll-fade";
import { AppLink, useNavigation } from "../navigation";
import { HelpLauncher } from "./help-launcher";
import {
  DndContext,
  PointerSensor,
  useSensor,
  useSensors,
  closestCenter,
  type DragEndEvent,
} from "@dnd-kit/core";
import { SortableContext, verticalListSortingStrategy, useSortable, arrayMove } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import {
  Activity,
  BarChart3,
  BookOpenText,
  Bot,
  ChevronRight,
  GitBranch,
  Inbox,
  Layers,
  ScrollText,
  Settings,
  Sparkles,
  Wrench,
  X,
  Zap,
} from "lucide-react";
import { Tooltip, TooltipTrigger, TooltipContent } from "@multica/ui/components/ui/tooltip";
import { Collapsible, CollapsibleTrigger, CollapsibleContent } from "@multica/ui/components/ui/collapsible";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarRail,
} from "@multica/ui/components/ui/sidebar";
import { useAuthStore } from "@multica/core/auth";
import { useCurrentWorkspace, useWorkspacePaths } from "@multica/core/paths";
import { useQueries, useQuery } from "@tanstack/react-query";

import { inboxKeys, deduplicateInboxItems } from "@multica/core/inbox/queries";
import { api } from "@multica/core/api";
import { pinListOptions } from "@multica/core/pins/queries";
import { useDeletePin, useReorderPins } from "@multica/core/pins/mutations";
import { featureDetailOptions } from "@multica/core/features/queries";
import { agentTaskSnapshotOptions } from "@multica/core/agents/queries";
import type { PinnedItem, AgentTask, Feature } from "@multica/core/types";
import type { InitiativeStatus } from "@multica/core/initiative/status";
import { FeatureIcon } from "../features/components/feature-icon";
import { useT } from "../i18n";

// Top-level nav items stay active when the user is on a child route
// (e.g. "Initiatives" stays lit on /:slug/initiatives/:id).
function isNavActive(pathname: string, href: string): boolean {
  return pathname === href || pathname.startsWith(href + "/");
}

const EMPTY_PINS: PinnedItem[] = [];
const EMPTY_INBOX: Awaited<ReturnType<typeof api.listInbox>> = [];
const EMPTY_TASKS: AgentTask[] = [];

type NavKey =
  | "live"
  | "initiatives"
  | "decisions"
  | "inbox"
  | "agents"
  | "usage"
  | "skills"
  | "runtimes"
  | "autopilots";
type NavLabelKey = NavKey | "costs";

interface NavEntry {
  key: NavKey;
  labelKey: NavLabelKey;
  icon: typeof Inbox;
}

// Primary nav leads with day-to-day execution surfaces.
const primaryNav: NavEntry[] = [
  { key: "live", labelKey: "live", icon: Activity },
  { key: "initiatives", labelKey: "initiatives", icon: Layers },
  { key: "decisions", labelKey: "decisions", icon: ScrollText },
  { key: "inbox", labelKey: "inbox", icon: Inbox },
];

// Workbench groups inventory / status surfaces below the primary execution row.
const workbenchNav: NavEntry[] = [
  { key: "agents", labelKey: "agents", icon: Bot },
  { key: "usage", labelKey: "costs", icon: BarChart3 },
  { key: "skills", labelKey: "skills", icon: BookOpenText },
  { key: "runtimes", labelKey: "runtimes", icon: Wrench },
  { key: "autopilots", labelKey: "autopilots", icon: Zap },
];

// Pinned-initiative status groups follow the prototype's ordering: running
// first, then in_review, then everything else not yet done, then done.
type PinGroup = {
  labelKey: "pinned_group_running" | "pinned_group_in_review" | "pinned_group_pending" | "pinned_group_done";
  statuses: InitiativeStatus[];
};
const PIN_GROUPS: PinGroup[] = [
  { labelKey: "pinned_group_running", statuses: ["running"] },
  { labelKey: "pinned_group_in_review", statuses: ["in_review"] },
  { labelKey: "pinned_group_pending", statuses: ["draft", "ready", "blocked", "cancelled"] },
  { labelKey: "pinned_group_done", statuses: ["done"] },
];

// Project header: brand mark, workspace name, branch (slug) + execution mode.
// Replaces the workspace switcher dropdown — one Claude session targets one
// workspace, so the chrome reflects identity, not selection.
function ProjectHeader({
  name,
  branch,
  mode,
}: {
  name: string;
  branch: string | null;
  mode: "worktree" | "in_place";
}) {
  return (
    <div className="flex items-center gap-2 px-2 py-1.5">
      <div className="grid size-7 shrink-0 place-items-center rounded-md bg-foreground text-background">
        <Sparkles className="size-3.5" />
      </div>
      <div className="min-w-0">
        <div className="truncate text-sm font-semibold">{name}</div>
        <div className="flex items-center gap-1.5 text-[11px] text-muted-foreground">
          <GitBranch className="size-3" />
          {branch && <span className="truncate">{branch}</span>}
          {branch && <span aria-hidden>·</span>}
          <span className="font-mono">{mode}</span>
        </div>
      </div>
    </div>
  );
}

// One rail entry. Self-contained: resolves its own href, active state, and
// label so both nav groups render identically.
function NavItem({ item, badge }: { item: NavEntry; badge?: React.ReactNode }) {
  const { t } = useT("layout");
  const { pathname } = useNavigation();
  const p = useWorkspacePaths();
  const href = p[item.key]();
  const isActive = isNavActive(pathname, href);
  return (
    <SidebarMenuItem>
      <SidebarMenuButton
        isActive={isActive}
        render={<AppLink href={href} />}
        className="text-muted-foreground hover:not-data-active:bg-sidebar-accent/70 data-active:bg-sidebar-accent data-active:text-sidebar-accent-foreground"
      >
        <item.icon />
        <span>{t(($) => $.nav[item.labelKey])}</span>
        {badge}
      </SidebarMenuButton>
    </SidebarMenuItem>
  );
}

/**
 * Presentational pinned-initiative row. Drag wiring lives here; the
 * `label` and `iconNode` come from the parent so this stays dumb.
 */
function SortableInitiativePin({
  pin,
  href,
  pathname,
  onUnpin,
  label,
  iconNode,
}: {
  pin: PinnedItem;
  href: string;
  pathname: string;
  onUnpin: () => void;
  label: string;
  iconNode: React.ReactNode;
}) {
  const { t } = useT("layout");
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({ id: pin.id });
  const wasDragged = useRef(false);

  useEffect(() => {
    if (isDragging) wasDragged.current = true;
  }, [isDragging]);

  const style = { transform: CSS.Transform.toString(transform), transition };
  const isActive = pathname === href;

  return (
    <SidebarMenuItem
      ref={setNodeRef}
      style={style}
      className={cn("group/pin", isDragging && "opacity-30")}
      {...attributes}
      {...listeners}
    >
      <SidebarMenuButton
        size="sm"
        isActive={isActive}
        render={<AppLink href={href} draggable={false} />}
        onClick={(event) => {
          if (wasDragged.current) {
            wasDragged.current = false;
            event.preventDefault();
            return;
          }
        }}
        className={cn(
          "text-muted-foreground hover:not-data-active:bg-sidebar-accent/70 data-active:bg-sidebar-accent data-active:text-sidebar-accent-foreground",
          isDragging && "pointer-events-none",
        )}
      >
        {iconNode}
        <span
          className="min-w-0 flex-1 overflow-hidden whitespace-nowrap"
          style={{
            maskImage: "linear-gradient(to right, black calc(100% - 12px), transparent)",
            WebkitMaskImage: "linear-gradient(to right, black calc(100% - 12px), transparent)",
          }}
        >{label}</span>
        <Tooltip>
          <TooltipTrigger
            render={<span role="button" />}
            className="hidden size-2.5 shrink-0 items-center justify-center rounded-sm text-muted-foreground group-hover/pin:flex hover:text-foreground"
            onClick={(event) => {
              event.preventDefault();
              event.stopPropagation();
              onUnpin();
            }}
          >
            <X className="size-1" />
          </TooltipTrigger>
          <TooltipContent side="top" sideOffset={4}>{t(($) => $.sidebar.unpin_tooltip)}</TooltipContent>
        </Tooltip>
      </SidebarMenuButton>
    </SidebarMenuItem>
  );
}

// Resolves a feature pin's display data + status from the cached
// featureDetailOptions query. Returns the feature so the parent can group
// pins by current status. Renders nothing while loading / on miss to avoid
// flashing into the wrong group.
function FeaturePinRow({
  pin,
  href,
  pathname,
  onUnpin,
  wsId,
}: {
  pin: PinnedItem;
  href: string;
  pathname: string;
  onUnpin: () => void;
  wsId: string;
}) {
  const featureQuery = useQuery(featureDetailOptions(wsId, pin.item_id));
  if (featureQuery.isPending || featureQuery.isError || !featureQuery.data) return null;
  const feature = featureQuery.data;
  return (
    <SortableInitiativePin
      pin={pin}
      href={href}
      pathname={pathname}
      onUnpin={onUnpin}
      label={feature.title}
      iconNode={<FeatureIcon feature={feature} size="sm" />}
    />
  );
}

// Maps a feature's lifecycle status to its pin group.
function findPinGroup(status: InitiativeStatus): PinGroup {
  return PIN_GROUPS.find((g) => g.statuses.includes(status)) ?? PIN_GROUPS[2]!;
}

interface AppSidebarProps {
  /** Rendered above SidebarHeader (e.g. desktop traffic light spacer) */
  topSlot?: React.ReactNode;
  /** Rendered in the header between project header and nav (e.g. search trigger) */
  searchSlot?: React.ReactNode;
  /** Extra className for SidebarHeader */
  headerClassName?: string;
  /** Extra style for SidebarHeader */
  headerStyle?: React.CSSProperties;
}

export function AppSidebar({ topSlot, searchSlot, headerClassName, headerStyle }: AppSidebarProps = {}) {
  const { t } = useT("layout");
  const { pathname } = useNavigation();
  const userId = useAuthStore((s) => s.user?.id);
  const workspace = useCurrentWorkspace();
  const p = useWorkspacePaths();

  const wsId = workspace?.id;
  const { data: inboxItems = EMPTY_INBOX } = useQuery({
    queryKey: wsId ? inboxKeys.list(wsId) : ["inbox", "disabled"],
    queryFn: () => api.listInbox(),
    enabled: !!wsId,
  });
  const unreadCount = React.useMemo(
    () => deduplicateInboxItems(inboxItems).filter((i) => !i.read).length,
    [inboxItems],
  );
  const { data: pinnedItems = EMPTY_PINS } = useQuery({
    ...pinListOptions(wsId ?? "", userId ?? ""),
    enabled: !!wsId && !!userId,
  });
  const { data: taskSnapshot = EMPTY_TASKS } = useQuery({
    ...agentTaskSnapshotOptions(wsId ?? ""),
    enabled: !!wsId,
  });
  const runningAgentCount = useMemo(
    () => taskSnapshot.filter((t) => t.status === "running").length,
    [taskSnapshot],
  );

  const deletePin = useDeletePin();
  const reorderPins = useReorderPins();
  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 5 } }));
  const sidebarScrollRef = useRef<HTMLDivElement>(null);
  const sidebarFadeStyle = useScrollFade(sidebarScrollRef, 24);

  // Issue pins are dropped from the sidebar — issues are visited from their
  // initiative now. Only feature pins survive into the new chrome.
  const featurePins = useMemo(
    () => pinnedItems.filter((pin) => pin.item_type === "feature"),
    [pinnedItems],
  );

  // Local presentational copy of pins for drop-animation stability.
  const [localPinned, setLocalPinned] = useState<PinnedItem[]>(featurePins);
  const isDraggingRef = useRef(false);
  useEffect(() => {
    if (!isDraggingRef.current) {
      setLocalPinned(featurePins);
    }
  }, [featurePins]);

  const handleDragStart = useCallback(() => {
    isDraggingRef.current = true;
  }, []);
  const handleDragEnd = useCallback(
    (event: DragEndEvent) => {
      isDraggingRef.current = false;
      const { active, over } = event;
      if (!over || active.id === over.id) return;
      const oldIndex = localPinned.findIndex((p) => p.id === active.id);
      const newIndex = localPinned.findIndex((p) => p.id === over.id);
      if (oldIndex === -1 || newIndex === -1) return;
      const reordered = arrayMove(localPinned, oldIndex, newIndex);
      setLocalPinned(reordered);
      reorderPins.mutate(reordered);
    },
    [localPinned, reorderPins],
  );

  return (
      <Sidebar variant="inset">
        {topSlot}
        <SidebarHeader className={cn("py-3", headerClassName)} style={headerStyle}>
          <SidebarMenu>
            <SidebarMenuItem>
              <ProjectHeader
                name={workspace?.name ?? "Multica"}
                branch={workspace?.slug ?? null}
                mode={workspace?.mode ?? "worktree"}
              />
            </SidebarMenuItem>
          </SidebarMenu>
          {searchSlot && (
            <SidebarMenu>
              <SidebarMenuItem>{searchSlot}</SidebarMenuItem>
            </SidebarMenu>
          )}
        </SidebarHeader>

        {/* Navigation */}
        <SidebarContent ref={sidebarScrollRef} style={sidebarFadeStyle}>
          <SidebarGroup>
            <SidebarGroupContent>
              <SidebarMenu className="gap-0.5">
                {primaryNav.map((item) => (
                  <NavItem
                    key={item.key}
                    item={item}
                    badge={
                      item.key === "inbox" && unreadCount > 0 ? (
                        <span className="ml-auto text-xs">{unreadCount > 99 ? "99+" : unreadCount}</span>
                      ) : undefined
                    }
                  />
                ))}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>

          <SidebarGroup>
            <SidebarGroupLabel>{t(($) => $.sidebar.workbench_label)}</SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu className="gap-0.5">
                {workbenchNav.map((item) => (
                  <NavItem key={item.key} item={item} />
                ))}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>

          {localPinned.length > 0 && (
            <Collapsible defaultOpen>
              <SidebarGroup className="group/pinned">
                <SidebarGroupLabel
                  render={<CollapsibleTrigger />}
                  className="group/trigger cursor-pointer hover:bg-sidebar-accent/70 hover:text-sidebar-accent-foreground"
                >
                  <span>{t(($) => $.sidebar.pinned_initiatives_label)}</span>
                  <ChevronRight className="!size-3 ml-1 stroke-[2.5] transition-transform duration-200 group-data-[panel-open]/trigger:rotate-90" />
                  <span className="ml-auto text-[10px] text-muted-foreground opacity-0 transition-opacity group-hover/pinned:opacity-100">{localPinned.length}</span>
                </SidebarGroupLabel>
                <CollapsibleContent>
                  <SidebarGroupContent>
                    <PinnedInitiativesByStatus
                      pins={localPinned}
                      pathname={pathname}
                      wsId={wsId ?? ""}
                      sensors={sensors}
                      onDragStart={handleDragStart}
                      onDragEnd={handleDragEnd}
                      onUnpin={(itemType, itemId) =>
                        deletePin.mutate({ itemType, itemId })
                      }
                      featureHref={(id) => p.initiativeDetail(id)}
                    />
                  </SidebarGroupContent>
                </CollapsibleContent>
              </SidebarGroup>
            </Collapsible>
          )}
        </SidebarContent>

        <SidebarFooter className="p-2">
          <div className="flex items-center justify-between gap-2 px-2 py-1 text-[11px] text-muted-foreground">
            <AgentsActiveIndicator count={runningAgentCount} suffix={t(($) => $.sidebar.agents_active_suffix)} idleLabel={t(($) => $.sidebar.no_agents_active)} />
            <div className="flex items-center gap-1">
              <SettingsLink href={p.settings()} pathname={pathname} label={t(($) => $.nav.settings)} />
              <HelpLauncher />
            </div>
          </div>
        </SidebarFooter>
        <SidebarRail />
      </Sidebar>
  );
}

function AgentsActiveIndicator({
  count,
  suffix,
  idleLabel,
}: {
  count: number;
  suffix: string;
  idleLabel: string;
}) {
  if (count === 0) {
    return <span className="text-muted-foreground">{idleLabel}</span>;
  }
  return (
    <span className="inline-flex items-center gap-1.5">
      <span className="relative inline-flex size-1.5">
        <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75" />
        <span className="relative inline-flex size-1.5 rounded-full bg-emerald-500" />
      </span>
      <span>
        {count} {suffix}
      </span>
    </span>
  );
}

function SettingsLink({
  href,
  pathname,
  label,
}: {
  href: string;
  pathname: string;
  label: string;
}) {
  const isActive = isNavActive(pathname, href);
  return (
    <Tooltip>
      <TooltipTrigger
        render={<AppLink href={href} />}
        aria-label={label}
        className={cn(
          "inline-flex size-7 items-center justify-center rounded-full transition-colors hover:bg-accent hover:text-foreground",
          isActive ? "text-foreground" : "text-muted-foreground",
        )}
      >
        <Settings className="size-3.5" />
      </TooltipTrigger>
      <TooltipContent side="top" sideOffset={4}>{label}</TooltipContent>
    </Tooltip>
  );
}

function PinnedInitiativesByStatus({
  pins,
  pathname,
  wsId,
  sensors,
  onDragStart,
  onDragEnd,
  onUnpin,
  featureHref,
}: {
  pins: PinnedItem[];
  pathname: string;
  wsId: string;
  sensors: ReturnType<typeof useSensors>;
  onDragStart: () => void;
  onDragEnd: (event: DragEndEvent) => void;
  onUnpin: (itemType: PinnedItem["item_type"], itemId: string) => void;
  featureHref: (id: string) => string;
}) {
  const { t } = useT("layout");
  const featureQueries = useQueries({
    queries: pins.map((pin) => featureDetailOptions(wsId, pin.item_id)),
  });
  const statusKey = featureQueries.map((q) => q.data?.status ?? "?").join("|");

  // Bucket pins by current feature status. Pins for which we don't yet have
  // a cached feature land in "other" so they still render somewhere.
  const groupedIds = useMemo(() => {
    const byGroup = new Map<PinGroup["labelKey"], PinnedItem[]>();
    for (const g of PIN_GROUPS) byGroup.set(g.labelKey, []);
    pins.forEach((pin, idx) => {
      const feature: Feature | undefined = featureQueries[idx]?.data;
      const group = feature ? findPinGroup(feature.status) : PIN_GROUPS[2]!;
      byGroup.get(group.labelKey)!.push(pin);
    });
    return byGroup;
    // featureQueries identity churns every render; the joined status snapshot
    // is the actual dependency that should retrigger this memo.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pins, statusKey]);

  return (
    <DndContext sensors={sensors} collisionDetection={closestCenter} onDragStart={onDragStart} onDragEnd={onDragEnd}>
      <SortableContext items={pins.map((p) => p.id)} strategy={verticalListSortingStrategy}>
        {PIN_GROUPS.map((g) => {
          const items = groupedIds.get(g.labelKey) ?? [];
          if (items.length === 0) return null;
          return (
            <div key={g.labelKey} className="mb-1">
              <SidebarGroupLabel className="text-[10px]">
                {t(($) => $.sidebar[g.labelKey])}
              </SidebarGroupLabel>
              <SidebarMenu className="gap-0.5">
                {items.map((pin) => (
                  <FeaturePinRow
                    key={pin.id}
                    pin={pin}
                    href={featureHref(pin.item_id)}
                    pathname={pathname}
                    onUnpin={() => onUnpin(pin.item_type, pin.item_id)}
                    wsId={wsId}
                  />
                ))}
              </SidebarMenu>
            </div>
          );
        })}
      </SortableContext>
    </DndContext>
  );
}
