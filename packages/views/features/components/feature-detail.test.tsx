import { forwardRef, useImperativeHandle, useState } from "react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import type { Feature, FeatureIssuesResponse } from "@multica/core/types";
import { renderWithI18n } from "../../test/i18n";

// ---------------------------------------------------------------------------
// Mocks — hoisted so they can be referenced in the implementations below
// ---------------------------------------------------------------------------

const mockUpdateFeature = vi.hoisted(() => vi.fn());

vi.mock("@multica/core/hooks", () => ({
  useWorkspaceId: () => "ws-1",
}));

vi.mock("@multica/core/auth", () => ({
  useAuthStore: Object.assign(
    (sel?: (s: { user: { id: string } }) => unknown) => {
      const state = { user: { id: "user-1" } };
      return sel ? sel(state) : state;
    },
    { getState: () => ({ user: { id: "user-1" } }) },
  ),
}));

vi.mock("@multica/core/paths", () => ({
  useCurrentWorkspace: () => ({ id: "ws-1", name: "Test WS", slug: "test" }),
  useWorkspacePaths: () => ({
    features: () => "/test/features",
    featureDetail: (id: string) => `/test/features/${id}`,
  }),
}));

vi.mock("@multica/core/workspace/queries", () => ({
  memberListOptions: () => ({ queryKey: ["members"], queryFn: () => Promise.resolve([]) }),
  agentListOptions: () => ({ queryKey: ["agents"], queryFn: () => Promise.resolve([]) }),
}));

vi.mock("@multica/core/workspace/hooks", () => ({
  useActorName: () => ({ getActorName: vi.fn() }),
}));

vi.mock("@multica/core/pins", () => ({
  pinListOptions: () => ({ queryKey: ["pins"], queryFn: () => Promise.resolve([]) }),
  useCreatePin: () => ({ mutate: vi.fn() }),
  useDeletePin: () => ({ mutate: vi.fn() }),
}));

vi.mock("@multica/core/features/mutations", () => ({
  useUpdateFeature: () => ({
    mutate: mockUpdateFeature,
    isPending: false,
  }),
  useDeleteFeature: () => ({ mutate: vi.fn() }),
}));

vi.mock("@multica/ui/hooks/use-mobile", () => ({
  useIsMobile: () => false,
}));

// react-resizable-panels: replace with simple pass-through layout
vi.mock("react-resizable-panels", () => ({
  useDefaultLayout: () => ({ defaultLayout: undefined, onLayoutChanged: vi.fn() }),
  usePanelRef: () => ({ current: null }),
}));

vi.mock("@multica/ui/components/ui/resizable", () => ({
  ResizablePanelGroup: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  ResizablePanel: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  ResizableHandle: () => null,
}));

vi.mock("@multica/ui/components/ui/sheet", () => ({
  Sheet: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  SheetContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));

vi.mock("@multica/ui/components/ui/skeleton", () => ({
  Skeleton: () => <div data-testid="skeleton" />,
}));

vi.mock("@multica/ui/components/ui/alert-dialog", () => ({
  AlertDialog: ({ children }: any) => <div>{children}</div>,
  AlertDialogContent: ({ children }: any) => <div>{children}</div>,
  AlertDialogHeader: ({ children }: any) => <div>{children}</div>,
  AlertDialogTitle: ({ children }: any) => <div>{children}</div>,
  AlertDialogDescription: ({ children }: any) => <div>{children}</div>,
  AlertDialogFooter: ({ children }: any) => <div>{children}</div>,
  AlertDialogAction: ({ children, onClick }: any) => <button onClick={onClick}>{children}</button>,
  AlertDialogCancel: ({ children }: any) => <button>{children}</button>,
}));

vi.mock("@multica/ui/components/ui/dropdown-menu", () => ({
  DropdownMenu: ({ children }: any) => <>{children}</>,
  DropdownMenuTrigger: ({ render }: any) => <>{render}</>,
  DropdownMenuContent: ({ children }: any) => <div>{children}</div>,
  DropdownMenuItem: ({ children, onClick }: any) => <button onClick={onClick}>{children}</button>,
  DropdownMenuSeparator: () => null,
}));

vi.mock("@multica/ui/components/ui/popover", () => ({
  Popover: ({ children }: any) => <>{children}</>,
  PopoverTrigger: ({ render }: any) => <>{render}</>,
  PopoverContent: ({ children }: any) => <div>{children}</div>,
}));

vi.mock("@multica/ui/components/ui/tooltip", () => ({
  Tooltip: ({ children }: any) => <>{children}</>,
  TooltipTrigger: ({ render }: any) => <>{render}</>,
  TooltipContent: ({ children }: any) => <div role="tooltip">{children}</div>,
}));

vi.mock("@multica/ui/components/ui/button", () => ({
  Button: ({ children, onClick, disabled, "data-testid": testId }: any) => (
    <button onClick={onClick} disabled={disabled} data-testid={testId}>
      {children}
    </button>
  ),
}));

vi.mock("@multica/ui/components/common/emoji-picker", () => ({
  EmojiPicker: () => null,
}));

vi.mock("@multica/ui/lib/utils", () => ({
  cn: (...args: (string | false | null | undefined)[]) => args.filter(Boolean).join(" "),
}));

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

vi.mock("../../navigation", () => ({
  AppLink: ({ children, href }: any) => <a href={href}>{children}</a>,
  useNavigation: () => ({ push: vi.fn() }),
}));

vi.mock("../../layout/page-header", () => ({
  PageHeader: ({ children }: any) => <header>{children}</header>,
}));

vi.mock("../../common/actor-avatar", () => ({
  ActorAvatar: () => <span data-testid="actor-avatar" />,
}));

vi.mock("../../issues/components/priority-icon", () => ({
  PriorityIcon: ({ priority }: any) => <span data-testid={`priority-${priority}`} />,
}));

vi.mock("../../issues/components/status-icon", () => ({
  StatusIcon: ({ status }: any) => <span data-testid={`status-${status}`} />,
}));

vi.mock("./feature-resources-section", () => ({
  FeatureResourcesSection: () => <div data-testid="resources-section" />,
}));

vi.mock("./feature-issue-metrics", () => ({
  getFeatureIssueMetrics: () => ({ totalCount: 0, completedCount: 0 }),
}));

vi.mock("./labels", () => ({
  useFeatureStatusLabels: () => ({
    planned: "Planned", in_progress: "In Progress", paused: "Paused",
    completed: "Completed", cancelled: "Cancelled",
  }),
  useFeaturePriorityLabels: () => ({
    urgent: "Urgent", high: "High", medium: "Medium", low: "Low", none: "No priority",
  }),
}));

vi.mock("../../editor/extensions/pinyin-match", () => ({
  matchesPinyin: () => false,
}));

vi.mock("../../editor", () => {
  const ContentEditor = forwardRef<any, any>(({ defaultValue, placeholder, onUpdate }, ref) => {
    const [value, setValue] = useState(defaultValue || "");
    useImperativeHandle(ref, () => ({
      getMarkdown: () => value,
      clearContent: vi.fn(),
      focus: vi.fn(),
    }));
    return (
      <textarea
        data-testid="content-editor"
        value={value}
        placeholder={placeholder}
        onChange={(e) => {
          setValue(e.target.value);
          onUpdate?.(e.target.value);
        }}
      />
    );
  });
  ContentEditor.displayName = "ContentEditor";

  const TitleEditor = forwardRef<any, any>(({ defaultValue, placeholder, onBlur }, ref) => {
    const [value] = useState(defaultValue || "");
    useImperativeHandle(ref, () => ({ getText: () => value, focus: vi.fn() }));
    return <input data-testid="title-editor" defaultValue={defaultValue} placeholder={placeholder} onBlur={(e) => onBlur?.(e.target.value)} />;
  });
  TitleEditor.displayName = "TitleEditor";

  return { ContentEditor, TitleEditor };
});

// ---------------------------------------------------------------------------
// Query mock — returns data keyed by query key prefix
// ---------------------------------------------------------------------------

const mockFeature = vi.hoisted((): { value: Feature } => ({
  value: {
    id: "feature-1",
    workspace_id: "ws-1",
    title: "Auth v2",
    description: "## Overview\n\nThis feature redesigns authentication.",
    icon: "🔐",
    status: "planned",
    priority: "high",
    lead_type: null,
    lead_id: null,
    target_branch: null,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    issue_count: 2,
    done_count: 0,
    resource_count: 0,
  },
}));

const mockIssuesResponse = vi.hoisted((): { value: FeatureIssuesResponse } => ({
  value: {
    ready_now: [
      { id: "i1", identifier: "MUL-1", title: "Add login page", status: "todo", priority: "high" },
    ],
    blocked: [
      { id: "i2", identifier: "MUL-2", title: "Add OAuth provider", status: "backlog", priority: "medium", blocked_by: ["MUL-1"] },
    ],
    pull_requests: [],
  },
}));

vi.mock("@tanstack/react-query", async () => {
  const actual = await vi.importActual<typeof import("@tanstack/react-query")>("@tanstack/react-query");
  return {
    ...actual,
    useQuery: (opts: { queryKey: unknown[] }) => {
      const key = opts.queryKey as string[];
      if (key[0] === "features" && key[2] === "detail") {
        return { data: mockFeature.value, isLoading: false };
      }
      if (key[0] === "features" && key[2] === "issues") {
        return { data: mockIssuesResponse.value, isLoading: false };
      }
      return { data: undefined, isLoading: false };
    },
  };
});

// ---------------------------------------------------------------------------
// Import component after all mocks
// ---------------------------------------------------------------------------

import { FeatureDetail } from "./feature-detail";

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("FeatureDetail", () => {
  beforeEach(() => {
    mockUpdateFeature.mockClear();
  });

  it("renders description as primary content", () => {
    renderWithI18n(<FeatureDetail featureId="feature-1" />);
    const editor = screen.getByTestId("content-editor");
    expect(editor).toBeInTheDocument();
    // The editor is positioned in the main content area with the description text
    expect((editor as HTMLTextAreaElement).value).toContain("Overview");
    // Description comes before the issues section in the DOM
    const issuesHeading = screen.getByText("Issues");
    expect(editor.compareDocumentPosition(issuesHeading)).toBe(Node.DOCUMENT_POSITION_FOLLOWING);
  });

  it("shows approve button when status is planned", () => {
    renderWithI18n(<FeatureDetail featureId="feature-1" />);
    expect(screen.getByTestId("approve-button")).toBeInTheDocument();
  });

  it("hides approve button when status is in_progress", () => {
    mockFeature.value = { ...mockFeature.value, status: "in_progress" };
    renderWithI18n(<FeatureDetail featureId="feature-1" />);
    expect(screen.queryByTestId("approve-button")).not.toBeInTheDocument();
    mockFeature.value = { ...mockFeature.value, status: "planned" };
  });

  it("clicking approve fires update mutation with in_progress status", async () => {
    mockFeature.value = { ...mockFeature.value, status: "planned" };
    renderWithI18n(<FeatureDetail featureId="feature-1" />);
    fireEvent.click(screen.getByTestId("approve-button"));
    await waitFor(() => {
      expect(mockUpdateFeature).toHaveBeenCalledWith(
        expect.objectContaining({ id: "feature-1", status: "in_progress" }),
      );
    });
  });

  it("splits issues into Ready now and Blocked sections", () => {
    renderWithI18n(<FeatureDetail featureId="feature-1" />);
    expect(screen.getByText("Ready now")).toBeInTheDocument();
    expect(screen.getByText("Blocked")).toBeInTheDocument();
    expect(screen.getByText("Add login page")).toBeInTheDocument();
    expect(screen.getByText("Add OAuth provider")).toBeInTheDocument();
    expect(screen.getByText(/blocked by MUL-1/i)).toBeInTheDocument();
  });

  it("shows target branch in header when set", () => {
    mockFeature.value = { ...mockFeature.value, target_branch: "feature/auth-v2" };
    renderWithI18n(<FeatureDetail featureId="feature-1" />);
    expect(screen.getByTestId("branch-indicator")).toHaveTextContent("feature/auth-v2");
    mockFeature.value = { ...mockFeature.value, target_branch: null };
  });

  it("does not show branch indicator when target_branch is null", () => {
    mockFeature.value = { ...mockFeature.value, target_branch: null };
    renderWithI18n(<FeatureDetail featureId="feature-1" />);
    expect(screen.queryByTestId("branch-indicator")).not.toBeInTheDocument();
  });

  it("shows PR link in header when an open PR is present", () => {
    mockIssuesResponse.value = {
      ...mockIssuesResponse.value,
      pull_requests: [
        { number: 42, html_url: "https://github.com/owner/repo/pull/42", state: "open", title: "Auth v2" },
      ],
    };
    renderWithI18n(<FeatureDetail featureId="feature-1" />);
    expect(screen.getByTestId("pr-link")).toBeInTheDocument();
    expect(screen.getByTestId("pr-link")).toHaveAttribute("href", "https://github.com/owner/repo/pull/42");
    mockIssuesResponse.value = { ...mockIssuesResponse.value, pull_requests: [] };
  });

  it("shows PR count when multiple open PRs are present", () => {
    mockIssuesResponse.value = {
      ...mockIssuesResponse.value,
      pull_requests: [
        { number: 1, html_url: "https://github.com/owner/repo/pull/1", state: "open", title: "PR 1" },
        { number: 2, html_url: "https://github.com/owner/repo/pull/2", state: "open", title: "PR 2" },
      ],
    };
    renderWithI18n(<FeatureDetail featureId="feature-1" />);
    expect(screen.getByTestId("pr-count")).toBeInTheDocument();
    mockIssuesResponse.value = { ...mockIssuesResponse.value, pull_requests: [] };
  });

  it("hides PR badge when all PRs are closed", () => {
    mockIssuesResponse.value = {
      ...mockIssuesResponse.value,
      pull_requests: [
        { number: 1, html_url: "https://github.com/owner/repo/pull/1", state: "closed", title: "Old PR" },
      ],
    };
    renderWithI18n(<FeatureDetail featureId="feature-1" />);
    expect(screen.queryByTestId("pr-link")).not.toBeInTheDocument();
    expect(screen.queryByTestId("pr-count")).not.toBeInTheDocument();
    mockIssuesResponse.value = { ...mockIssuesResponse.value, pull_requests: [] };
  });


  it("does not render a New Issue button in the empty state", () => {
    mockIssuesResponse.value = { ready_now: [], blocked: [], pull_requests: [] };
    renderWithI18n(<FeatureDetail featureId="feature-1" />);
    expect(screen.queryByRole("button", { name: /new issue/i })).not.toBeInTheDocument();
    mockIssuesResponse.value = {
      ready_now: [{ id: "i1", identifier: "MUL-1", title: "Add login page", status: "todo", priority: "high" }],
      blocked: [{ id: "i2", identifier: "MUL-2", title: "Add OAuth provider", status: "backlog", priority: "medium", blocked_by: ["MUL-1"] }],
      pull_requests: [],
    };
  });

  it("shows in_progress issues with running indicator", () => {
    mockIssuesResponse.value = {
      ready_now: [{ id: "i1", identifier: "MUL-1", title: "Running task", status: "in_progress", priority: "high" }],
      blocked: [],
      pull_requests: [],
    };
    renderWithI18n(<FeatureDetail featureId="feature-1" />);
    expect(screen.getByText("running")).toBeInTheDocument();
    mockIssuesResponse.value = {
      ready_now: [{ id: "i1", identifier: "MUL-1", title: "Add login page", status: "todo", priority: "high" }],
      blocked: [{ id: "i2", identifier: "MUL-2", title: "Add OAuth provider", status: "backlog", priority: "medium", blocked_by: ["MUL-1"] }],
      pull_requests: [],
    };
  });
});
