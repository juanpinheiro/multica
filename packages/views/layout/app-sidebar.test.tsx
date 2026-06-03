import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import layout from "../locales/en/layout.json";
import { AppSidebar } from "./app-sidebar";

vi.mock("../i18n", () => ({
  useT: () => ({ t: (selector: (m: typeof layout) => string) => selector(layout) }),
}));

const { tasks, pins } = vi.hoisted(() => ({
  tasks: { current: [] as Array<{ id: string; status: string }> },
  pins: { current: [] as Array<Record<string, unknown>> },
}));

vi.mock("@dnd-kit/core", () => ({
  DndContext: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  PointerSensor: vi.fn(),
  closestCenter: vi.fn(),
  useSensor: vi.fn(),
  useSensors: vi.fn(),
}));
vi.mock("@dnd-kit/sortable", () => ({
  SortableContext: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  useSortable: () => ({ attributes: {}, listeners: {}, setNodeRef: vi.fn() }),
  verticalListSortingStrategy: vi.fn(),
  arrayMove: <T,>(arr: T[]) => arr,
}));
vi.mock("@dnd-kit/utilities", () => ({ CSS: { Transform: { toString: () => undefined } } }));
vi.mock("@multica/ui/components/ui/sidebar", () => ({
  Sidebar: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  SidebarContent: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  SidebarFooter: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  SidebarGroup: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  SidebarGroupContent: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  SidebarGroupLabel: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  SidebarHeader: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  SidebarMenu: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  SidebarMenuButton: ({ children }: { children: React.ReactNode }) => <button type="button">{children}</button>,
  SidebarMenuItem: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  SidebarRail: () => null,
}));
vi.mock("@multica/ui/components/ui/dropdown-menu", () => ({
  DropdownMenu: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  DropdownMenuContent: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  DropdownMenuGroup: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  DropdownMenuItem: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  DropdownMenuLabel: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  DropdownMenuSeparator: () => null,
  DropdownMenuTrigger: ({ render }: { render: React.ReactNode }) => <>{render}</>,
}));
vi.mock("@multica/ui/components/ui/collapsible", () => ({
  Collapsible: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  CollapsibleContent: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  CollapsibleTrigger: () => <button type="button" />,
}));
vi.mock("@multica/ui/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  TooltipContent: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  TooltipTrigger: ({ children, render }: { children?: React.ReactNode; render?: React.ReactNode }) =>
    render ? <>{render}</> : <button type="button">{children}</button>,
}));
vi.mock("./help-launcher", () => ({ HelpLauncher: () => null }));
vi.mock("../navigation", () => ({
  AppLink: ({ children, href }: { children: React.ReactNode; href: string }) => <a href={href}>{children}</a>,
  useNavigation: () => ({ pathname: "/acme/live", push: vi.fn() }),
}));
vi.mock("../features/components/feature-icon", () => ({ FeatureIcon: () => <span /> }));

vi.mock("@multica/core/auth", () => ({
  useAuthStore: (selector: (state: { user: { id: string } }) => unknown) => selector({ user: { id: "user-1" } }),
}));
vi.mock("@multica/core/paths", () => ({
  paths: { workspace: (slug: string) => ({ live: () => `/${slug}/live` }) },
  useCurrentWorkspace: () => ({ id: "ws-1", name: "Acme", slug: "acme", mode: "worktree" }),
  useWorkspacePaths: () => ({
    live: () => "/acme/live",
    initiatives: () => "/acme/initiatives",
    decisions: () => "/acme/decisions",
    inbox: () => "/acme/inbox",
    issues: () => "/acme/issues",
    autopilots: () => "/acme/autopilots",
    agents: () => "/acme/agents",
    usage: () => "/acme/usage",
    runtimes: () => "/acme/runtimes",
    skills: () => "/acme/skills",
    settings: () => "/acme/settings",
    issueDetail: (id: string) => `/acme/issues/${id}`,
    initiativeDetail: (id: string) => `/acme/initiatives/${id}`,
  }),
}));
vi.mock("@multica/core/api", async (importOriginal) => ({ ...(await importOriginal<typeof import("@multica/core/api")>()), api: {} }));
vi.mock("@multica/core/inbox/queries", () => ({ deduplicateInboxItems: (items: unknown[]) => items, inboxKeys: { list: () => ["inbox"] } }));
vi.mock("@multica/core/pins/mutations", () => ({ useDeletePin: () => ({ mutate: vi.fn() }), useReorderPins: () => ({ mutate: vi.fn() }) }));
vi.mock("@multica/core/pins/queries", () => ({ pinListOptions: () => ({ queryKey: ["pins"] }) }));
vi.mock("@multica/core/features/queries", () => ({ featureDetailOptions: () => ({ queryKey: ["feature"] }) }));
vi.mock("@multica/core/agents/queries", () => ({ agentTaskSnapshotOptions: () => ({ queryKey: ["task-snapshot"] }) }));
vi.mock("@tanstack/react-query", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@tanstack/react-query")>()),
  useMutation: () => ({ isPending: false, mutate: vi.fn() }),
  useQuery: ({ queryKey }: { queryKey: readonly unknown[] }) => {
    if (queryKey[0] === "pins") return { data: pins.current };
    if (queryKey[0] === "task-snapshot") return { data: tasks.current };
    return { data: [] };
  },
  useQueries: () => [],
  useQueryClient: () => ({ fetchQuery: vi.fn(), invalidateQueries: vi.fn() }),
}));

function precedes(before: HTMLElement, after: HTMLElement): boolean {
  return Boolean(before.compareDocumentPosition(after) & Node.DOCUMENT_POSITION_FOLLOWING);
}

describe("AppSidebar chrome", () => {
  it("renders the project header with workspace name, branch and mode", () => {
    tasks.current = [];
    render(<AppSidebar />);
    expect(screen.getByText("Acme")).toBeInTheDocument();
    expect(screen.getByText("acme")).toBeInTheDocument();
    expect(screen.getByText("worktree")).toBeInTheDocument();
  });

  it("renders the primary nav in order: Live, Initiatives, Decisions, Inbox", () => {
    tasks.current = [];
    render(<AppSidebar />);
    const order = ["Live", "Initiatives", "Decisions", "Inbox"];
    for (const label of order) expect(screen.getByText(label)).toBeInTheDocument();
    order.slice(1).forEach((label, i) => {
      expect(precedes(screen.getByText(order[i]!), screen.getByText(label))).toBe(true);
    });
  });

  it("renders the Workbench section with Agents, Costs, Skills, Runtimes, Autopilot", () => {
    tasks.current = [];
    render(<AppSidebar />);
    const order = ["Agents", "Costs", "Skills", "Runtimes", "Autopilot"];
    for (const label of order) expect(screen.getByText(label)).toBeInTheDocument();
    order.slice(1).forEach((label, i) => {
      expect(precedes(screen.getByText(order[i]!), screen.getByText(label))).toBe(true);
    });
    // Sanity: Workbench label is present.
    expect(screen.getByText("Workbench")).toBeInTheDocument();
  });

  it("drops the workspace switcher dropdown and other dead chrome", () => {
    tasks.current = [];
    render(<AppSidebar />);
    expect(screen.queryByText("Workspaces")).toBeNull();
    expect(screen.queryByText("My Issues")).toBeNull();
    expect(screen.queryByText("New Issue")).toBeNull();
    expect(screen.queryByText("Squads")).toBeNull();
    expect(screen.queryByText("Create workspace")).toBeNull();
  });

  it("shows the agents-active count in the footer when tasks are running", () => {
    tasks.current = [
      { id: "t1", status: "running" },
      { id: "t2", status: "running" },
      { id: "t3", status: "queued" },
    ];
    render(<AppSidebar />);
    expect(screen.getByText(/2 agents active/i)).toBeInTheDocument();
  });

  it("shows Idle in the footer when no tasks are running", () => {
    tasks.current = [];
    render(<AppSidebar />);
    expect(screen.getByText(/idle/i)).toBeInTheDocument();
  });
});
