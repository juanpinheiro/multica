"use client";

import React, { useCallback, useEffect, useRef, useState } from "react";
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
  Inbox,
  ListTodo,
  Bot,
  ChevronDown,
  ChevronRight,
  Settings,
  Check,
  BookOpenText,
  FolderKanban,
  BarChart3,
  X,
  Zap,
} from "lucide-react";
import { WorkspaceAvatar } from "../workspace/workspace-avatar";
import { Tooltip, TooltipTrigger, TooltipContent } from "@multica/ui/components/ui/tooltip";
import { Collapsible, CollapsibleTrigger, CollapsibleContent } from "@multica/ui/components/ui/collapsible";
import { StatusIcon } from "../issues/components/status-icon";
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
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@multica/ui/components/ui/dropdown-menu";
import { useAuthStore } from "@multica/core/auth";
import { useCurrentWorkspace, useWorkspacePaths, paths } from "@multica/core/paths";
import { workspaceListOptions } from "@multica/core/workspace/queries";
import { useQuery } from "@tanstack/react-query";

import { inboxKeys, deduplicateInboxItems } from "@multica/core/inbox/queries";
import { api, ApiError } from "@multica/core/api";
import { pinListOptions } from "@multica/core/pins/queries";
import { useDeletePin, useReorderPins } from "@multica/core/pins/mutations";
import { issueDetailOptions } from "@multica/core/issues/queries";
import { featureDetailOptions } from "@multica/core/features/queries";
import type { PinnedItem } from "@multica/core/types";
import { FeatureIcon } from "../features/components/feature-icon";
import { useT } from "../i18n";

// Top-level nav items stay active when the user is on a child route
// (e.g. "Projects" stays lit on /:slug/features/:id). Pinned items keep
// strict equality elsewhere — a pinned project shouldn't highlight on
// sub-pages of itself.
function isNavActive(pathname: string, href: string): boolean {
  return pathname === href || pathname.startsWith(href + "/");
}

// Stable empty arrays for query defaults. Using an inline `= []` default on
// `useQuery` creates a new array reference on every render when `data` is
// undefined (e.g. query disabled or loading) — which in turn breaks any
// `useEffect`/`useMemo` that depends on the value, and can trigger infinite
// re-render loops when the effect itself calls `setState`.
const EMPTY_PINS: PinnedItem[] = [];
const EMPTY_WORKSPACES: Awaited<ReturnType<typeof api.listWorkspaces>> = [];
const EMPTY_INBOX: Awaited<ReturnType<typeof api.listInbox>> = [];

// Nav items reference WorkspacePaths method names so they can be resolved
// against the current workspace slug at render time (see NavItem). Labels
// resolve at render via useT("layout"). Only parameterless paths are valid
// nav destinations.
type NavKey = "inbox" | "issues" | "features" | "autopilots" | "agents" | "usage" | "skills" | "settings";
type NavLabelKey = NavKey;

interface NavEntry {
  key: NavKey;
  labelKey: NavLabelKey;
  icon: typeof Inbox;
}

// Execution and planning lead the rail; supporting surfaces follow below a gap.
const primaryNav: NavEntry[] = [
  { key: "issues", labelKey: "issues", icon: ListTodo },
  { key: "inbox", labelKey: "inbox", icon: Inbox },
  { key: "features", labelKey: "features", icon: FolderKanban },
];

const secondaryNav: NavEntry[] = [
  { key: "agents", labelKey: "agents", icon: Bot },
  { key: "autopilots", labelKey: "autopilots", icon: Zap },
  { key: "skills", labelKey: "skills", icon: BookOpenText },
  { key: "usage", labelKey: "usage", icon: BarChart3 },
  { key: "settings", labelKey: "settings", icon: Settings },
];

// The Multica brand mark (the favicon glyph), inlined so it adapts to the
// active theme via semantic tokens rather than a fixed-color asset.
function MulticaMark({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 100 100" role="img" aria-label="Multica" className={className}>
      <rect width="100" height="100" rx="20" className="fill-foreground" />
      <polygon
        className="fill-background"
        points="45,62.1 45,100 55,100 55,62.1 81.8,88.9 88.9,81.8 62.1,55 100,55 100,45 62.1,45 88.9,18.2 81.8,11.1 55,37.9 55,0 45,0 45,37.9 18.2,11.1 11.1,18.2 37.9,45 0,45 0,55 37.9,55 11.1,81.8 18.2,88.9"
      />
    </svg>
  );
}

function BrandMark() {
  return (
    <div className="flex items-center px-2 py-1">
      <MulticaMark className="size-6 shrink-0" />
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
 * Presentational pin row. The `label` and `iconNode` are computed by the
 * parent `PinRow` from cached issue / project detail queries — keeping
 * this component dumb means the dnd-kit / navigation wiring lives in
 * one place and the data flow is explicit.
 */
function SortablePinItem({
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

/**
 * Smart wrapper that resolves a pin's display data (label + status/icon)
 * from the issue / project detail query cache. Both queries are declared
 * unconditionally with `enabled` gates so the hook order stays stable
 * regardless of `pin.item_type`.
 *
 * Loading: render a flat skeleton so the sidebar height doesn't jump.
 * Missing (deleted item / 404): render nothing — the row hides itself
 * until the user unpins manually or a server-side cascade catches up.
 */
function PinRow({
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
  const isIssue = pin.item_type === "issue";
  const issueQuery = useQuery({
    ...issueDetailOptions(wsId, pin.item_id),
    enabled: isIssue,
  });
  const featureQuery = useQuery({
    ...featureDetailOptions(wsId, pin.item_id),
    enabled: !isIssue,
  });

  const triggeredRef = useRef(false);
  useEffect(() => {
    const err = isIssue ? issueQuery.error : featureQuery.error;
    if (err instanceof ApiError && err.status === 404 && !triggeredRef.current) {
      triggeredRef.current = true;
      onUnpin();
    }
  }, [isIssue, issueQuery.error, onUnpin, featureQuery.error]);

  if (isIssue) {
    if (issueQuery.isPending) return <PinSkeleton />;
    if (issueQuery.isError || !issueQuery.data) return null;
    const issue = issueQuery.data;
    const label = issue.identifier ? `${issue.identifier} ${issue.title}` : issue.title;
    const iconNode = (
      /* Override parent [&_svg]:size-4 — pinned items need smaller icons to match sm size */
      <StatusIcon status={issue.status} className="!size-3.5 shrink-0" />
    );
    return (
      <SortablePinItem
        pin={pin}
        href={href}
        pathname={pathname}
        onUnpin={onUnpin}
        label={label}
        iconNode={iconNode}
      />
    );
  }

  if (featureQuery.isPending) return <PinSkeleton />;
  if (featureQuery.isError || !featureQuery.data) return null;
  const feature = featureQuery.data;
  const iconNode = <FeatureIcon feature={feature} size="sm" />;
  return (
    <SortablePinItem
      pin={pin}
      href={href}
      pathname={pathname}
      onUnpin={onUnpin}
      label={feature.title}
      iconNode={iconNode}
    />
  );
}

function PinSkeleton() {
  return (
    <SidebarMenuItem>
      <div className="flex h-7 w-full items-center gap-2 px-2">
        <div className="size-3.5 shrink-0 rounded-sm bg-sidebar-accent/40" />
        <div className="h-3 w-24 rounded bg-sidebar-accent/40" />
      </div>
    </SidebarMenuItem>
  );
}

interface AppSidebarProps {
  /** Rendered above SidebarHeader (e.g. desktop traffic light spacer) */
  topSlot?: React.ReactNode;
  /** Rendered in the header between workspace switcher and new-issue button (e.g. search trigger) */
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
  const { data: workspaces = EMPTY_WORKSPACES } = useQuery(workspaceListOptions());

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
  const deletePin = useDeletePin();
  const reorderPins = useReorderPins();
  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 5 } }));
  const sidebarScrollRef = useRef<HTMLDivElement>(null);
  const sidebarFadeStyle = useScrollFade(sidebarScrollRef, 24);

  // Local presentational copy of pinnedItems for drop-animation stability.
  // Follows TQ at rest; frozen during a drag gesture so a mid-drag cache
  // write (our own optimistic update, or a WS refetch) cannot reorder the
  // DOM under dnd-kit while its drop animation is still interpolating.
  const [localPinned, setLocalPinned] = useState<PinnedItem[]>(pinnedItems);
  const isDraggingRef = useRef(false);
  useEffect(() => {
    if (!isDraggingRef.current) {
      setLocalPinned(pinnedItems);
    }
  }, [pinnedItems]);

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
          {/* Brand mark leads the rail; project identity follows in the switcher */}
          <SidebarMenu>
            <SidebarMenuItem>
              <BrandMark />
            </SidebarMenuItem>
          </SidebarMenu>
          {/* Recent projects — navigation between known workspaces only */}
          <SidebarMenu>
            <SidebarMenuItem>
              <DropdownMenu>
                <DropdownMenuTrigger
                  render={
                    <SidebarMenuButton>
                      <WorkspaceAvatar name={workspace?.name ?? "M"} size="sm" />
                      <span className="flex-1 truncate font-medium">
                        {workspace?.name ?? "Multica"}
                      </span>
                      <ChevronDown className="size-3 text-muted-foreground" />
                    </SidebarMenuButton>
                  }
                />
                <DropdownMenuContent
                  className="w-auto min-w-56"
                  align="start"
                  side="bottom"
                  sideOffset={4}
                >
                  <DropdownMenuGroup>
                    <DropdownMenuLabel className="text-xs text-muted-foreground">
                      {t(($) => $.sidebar.workspaces_label)}
                    </DropdownMenuLabel>
                    {workspaces.map((ws) => (
                      <DropdownMenuItem
                        key={ws.id}
                        render={
                          <AppLink href={paths.workspace(ws.slug).issues()} />
                        }
                      >
                        <WorkspaceAvatar name={ws.name} size="sm" />
                        <span className="flex-1 truncate">{ws.name}</span>
                        {ws.id === workspace?.id && (
                          <Check className="h-3.5 w-3.5 text-primary" />
                        )}
                      </DropdownMenuItem>
                    ))}
                  </DropdownMenuGroup>
                </DropdownMenuContent>
              </DropdownMenu>
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

          {localPinned.length > 0 && (
            <Collapsible defaultOpen>
              <SidebarGroup className="group/pinned">
                <SidebarGroupLabel
                  render={<CollapsibleTrigger />}
                  className="group/trigger cursor-pointer hover:bg-sidebar-accent/70 hover:text-sidebar-accent-foreground"
                >
                  <span>{t(($) => $.sidebar.pinned_label)}</span>
                  <ChevronRight className="!size-3 ml-1 stroke-[2.5] transition-transform duration-200 group-data-[panel-open]/trigger:rotate-90" />
                  <span className="ml-auto text-[10px] text-muted-foreground opacity-0 transition-opacity group-hover/pinned:opacity-100">{localPinned.length}</span>
                </SidebarGroupLabel>
                <CollapsibleContent>
                  <SidebarGroupContent>
                    <DndContext sensors={sensors} collisionDetection={closestCenter} onDragStart={handleDragStart} onDragEnd={handleDragEnd}>
                      <SortableContext items={localPinned.map((p) => p.id)} strategy={verticalListSortingStrategy}>
                        <SidebarMenu className="gap-0.5">
                          {localPinned.map((pin: PinnedItem) => (
                            <PinRow
                              key={pin.id}
                              pin={pin}
                              href={pin.item_type === "issue" ? p.issueDetail(pin.item_id) : p.featureDetail(pin.item_id)}
                              pathname={pathname}
                              onUnpin={() => deletePin.mutate({ itemType: pin.item_type, itemId: pin.item_id })}
                              wsId={wsId ?? ""}
                            />
                          ))}
                        </SidebarMenu>
                      </SortableContext>
                    </DndContext>
                  </SidebarGroupContent>
                </CollapsibleContent>
              </SidebarGroup>
            </Collapsible>
          )}

          <SidebarGroup>
            <SidebarGroupContent>
              <SidebarMenu className="gap-0.5">
                {secondaryNav.map((item) => (
                  <NavItem key={item.key} item={item} />
                ))}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        </SidebarContent>

        <SidebarFooter className="p-2">
          <div className="flex justify-end">
            <HelpLauncher />
          </div>
        </SidebarFooter>
        <SidebarRail />
      </Sidebar>
  );
}
