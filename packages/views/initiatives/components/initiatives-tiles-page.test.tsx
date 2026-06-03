import { render, screen, fireEvent } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ActivityEvent, BuildLiveEventsResult } from "@multica/core/tasks/build-live-events";
import type { AgentTask, Feature, Issue, Milestone } from "@multica/core/types";
import layout from "../../locales/en/layout.json";
import { InitiativesTilesPage } from "./initiatives-tiles-page";

const { liveResult, features, issues, milestones, tasks, navPush } = vi.hoisted(() => ({
  liveResult: { current: null as BuildLiveEventsResult | null },
  features: { current: [] as Feature[] },
  issues: { current: [] as Issue[] },
  milestones: { current: [] as Milestone[] },
  tasks: { current: [] as AgentTask[] },
  navPush: vi.fn(),
}));

function interpolate(template: string, params: Record<string, unknown> = {}): string {
  return template.replace(/\{\{(\w+)\}\}/g, (_, key) =>
    params[key] !== undefined ? String(params[key]) : `{{${key}}}`,
  );
}

vi.mock("../../i18n", () => ({
  useT: () => ({
    t: (selector: (m: typeof layout) => string, params?: Record<string, unknown>) =>
      interpolate(selector(layout), params),
  }),
  useTimeAgo: () => () => "just now",
}));

vi.mock("@multica/core/hooks", () => ({
  useWorkspaceId: () => "ws-1",
}));

vi.mock("@multica/core/paths", () => ({
  useWorkspacePaths: () => ({
    initiativeDetail: (id: string) => `/acme/initiatives/${id}`,
    initiativeIssue: (fid: string, iid: string) =>
      `/acme/initiatives/${fid}/issues/${iid}`,
    issueDetail: (id: string) => `/acme/issues/${id}`,
  }),
}));

vi.mock("../../navigation", () => ({
  AppLink: ({ href, children, onClick, ...rest }: React.AnchorHTMLAttributes<HTMLAnchorElement> & { href: string }) => (
    <a
      href={href}
      onClick={(e) => {
        e.preventDefault();
        navPush(href);
        onClick?.(e);
      }}
      {...rest}
    >
      {children}
    </a>
  ),
}));

vi.mock("@multica/core/tasks/use-live-events", () => ({
  useLiveEvents: () => liveResult.current!,
}));

vi.mock("@multica/core/features/queries", () => ({
  featureListOptions: () => ({ queryKey: ["features"] }),
}));

vi.mock("@multica/core/issues/queries", () => ({
  issueListOptions: () => ({ queryKey: ["issues"] }),
}));

vi.mock("@multica/core/milestones/queries", () => ({
  milestoneListOptions: () => ({ queryKey: ["milestones"] }),
}));

vi.mock("@multica/core/agents/queries", () => ({
  agentTaskSnapshotOptions: () => ({ queryKey: ["task-snapshot"] }),
}));

vi.mock("@tanstack/react-query", () => ({
  useQuery: ({ queryKey }: { queryKey: readonly unknown[] }) => {
    if (queryKey[0] === "features") return { data: features.current };
    if (queryKey[0] === "issues") return { data: issues.current };
    if (queryKey[0] === "milestones") return { data: milestones.current };
    if (queryKey[0] === "task-snapshot") return { data: tasks.current };
    return { data: [] };
  },
}));

function makeFeature(overrides: Partial<Feature> = {}): Feature {
  return {
    id: "feature-1",
    workspace_id: "ws-1",
    title: "Refactor checkout",
    description: null,
    icon: null,
    status: "running",
    priority: "medium",
    lead_type: null,
    lead_id: null,
    branch_slug: "refactor-checkout",
    mode: "hitl",
    budget_tokens: 0,
    budget_runs: 0,
    budget_seconds: 0,
    failure_tolerance: 0,
    created_at: "2026-06-01T10:00:00Z",
    updated_at: "2026-06-03T11:00:00Z",
    issue_count: 6,
    done_count: 2,
    resource_count: 0,
    ...overrides,
  };
}

function makeIssue(overrides: Partial<Issue> = {}): Issue {
  return {
    id: "issue-1",
    workspace_id: "ws-1",
    number: 1,
    identifier: "MUL-1",
    title: "Wire setup intent",
    description: null,
    status: "in_progress",
    priority: "medium",
    assignee_type: "agent",
    assignee_id: "agent-1",
    creator_type: "member",
    creator_id: "user-1",
    parent_issue_id: null,
    feature_id: "feature-1",
    position: 1,
    start_date: null,
    due_date: null,
    metadata: {},
    created_at: "2026-06-01T10:00:00Z",
    updated_at: "2026-06-03T11:00:00Z",
    ...overrides,
  };
}

function makeMilestone(overrides: Partial<Milestone> = {}): Milestone {
  return {
    id: "ms-1",
    workspace_id: "ws-1",
    feature_id: "feature-1",
    title: "M1",
    position: 1,
    validation_status: "pending",
    created_at: "",
    updated_at: "",
    ...overrides,
  };
}

function makeEvent(overrides: Partial<ActivityEvent> = {}): ActivityEvent {
  return {
    id: "e-1",
    at: "2026-06-03T11:59:00Z",
    type: "agent_started",
    initiativeId: "feature-1",
    issueId: "issue-1",
    agentId: "agent-1",
    message: "agent claimed the task",
    ...overrides,
  };
}

function seedResult(events: ActivityEvent[]) {
  liveResult.current = { events, runningAgents: 0, runningInitiatives: 0 };
}

describe("InitiativesTilesPage", () => {
  beforeEach(() => {
    navPush.mockClear();
    seedResult([]);
    features.current = [];
    issues.current = [];
    milestones.current = [];
    tasks.current = [];
  });

  it("shows an empty state when there are no initiatives", () => {
    render(<InitiativesTilesPage />);
    expect(screen.getByTestId("initiatives-empty")).toBeInTheDocument();
  });

  it("renders one tile per feature", () => {
    features.current = [
      makeFeature({ id: "a", title: "Alpha" }),
      makeFeature({ id: "b", title: "Beta", status: "ready" }),
    ];
    render(<InitiativesTilesPage />);
    expect(screen.getAllByTestId(/^initiative-tile-/)).toHaveLength(2);
    expect(screen.getByText("Alpha")).toBeInTheDocument();
    expect(screen.getByText("Beta")).toBeInTheDocument();
  });

  it("sorts tiles by status: running, in_review, ready, blocked, done", () => {
    features.current = [
      makeFeature({ id: "d", title: "Done one", status: "done" }),
      makeFeature({ id: "r", title: "Ready one", status: "ready" }),
      makeFeature({ id: "ir", title: "Review one", status: "in_review" }),
      makeFeature({ id: "ru", title: "Running one", status: "running" }),
      makeFeature({ id: "bl", title: "Blocked one", status: "blocked" }),
    ];
    render(<InitiativesTilesPage />);
    const titles = screen
      .getAllByTestId(/^initiative-tile-/)
      .map((el) => el.getAttribute("data-testid"));
    expect(titles).toEqual([
      "initiative-tile-ru",
      "initiative-tile-ir",
      "initiative-tile-r",
      "initiative-tile-bl",
      "initiative-tile-d",
    ]);
  });

  it("clicking a tile navigates to the initiative detail path", () => {
    features.current = [makeFeature({ id: "alpha", title: "Alpha" })];
    render(<InitiativesTilesPage />);
    fireEvent.click(screen.getByTestId("initiative-tile-alpha"));
    expect(navPush).toHaveBeenCalledWith("/acme/initiatives/alpha");
  });

  it("renders milestone progress when milestones exist for the feature", () => {
    features.current = [makeFeature({ id: "f1", issue_count: 4, done_count: 1 })];
    milestones.current = [
      makeMilestone({ id: "m1", feature_id: "f1", validation_status: "passed" }),
      makeMilestone({ id: "m2", feature_id: "f1", validation_status: "pending" }),
      makeMilestone({ id: "m3", feature_id: "f1", validation_status: "pending" }),
    ];
    render(<InitiativesTilesPage />);
    const tile = screen.getByTestId("initiative-tile-f1");
    expect(tile).toHaveTextContent("1/3 milestones");
    expect(tile).toHaveTextContent("1/4 issues");
  });

  it("renders a running-agents row only when an agent is on an issue of the feature", () => {
    features.current = [
      makeFeature({ id: "active", title: "Active" }),
      makeFeature({ id: "idle", title: "Idle" }),
    ];
    issues.current = [
      makeIssue({ id: "issue-a", feature_id: "active" }),
    ];
    tasks.current = [
      {
        id: "task-a",
        agent_id: "agent-1",
        runtime_id: "rt-1",
        issue_id: "issue-a",
        status: "running",
        priority: 0,
        dispatched_at: null,
        started_at: "2026-06-03T11:55:00Z",
        completed_at: null,
        result: null,
        error: null,
        created_at: "2026-06-03T11:54:00Z",
      },
    ];
    render(<InitiativesTilesPage />);
    expect(
      screen.getByTestId("initiative-tile-active").querySelector("[data-testid='running-agents-row']"),
    ).toBeInTheDocument();
    expect(
      screen.getByTestId("initiative-tile-idle").querySelector("[data-testid='running-agents-row']"),
    ).toBeNull();
  });

  it("renders up to 3 recent mini-feed events for the feature", () => {
    features.current = [makeFeature({ id: "f1" })];
    seedResult([
      makeEvent({ id: "e1", initiativeId: "f1", at: "2026-06-03T11:59:50Z" }),
      makeEvent({ id: "e2", initiativeId: "f1", at: "2026-06-03T11:59:40Z" }),
      makeEvent({ id: "e3", initiativeId: "f1", at: "2026-06-03T11:59:30Z" }),
      makeEvent({ id: "e4", initiativeId: "f1", at: "2026-06-03T11:59:20Z" }),
      makeEvent({ id: "other", initiativeId: "other-feature", at: "2026-06-03T11:59:55Z" }),
    ]);
    render(<InitiativesTilesPage />);
    const rows = screen
      .getByTestId("initiative-tile-f1")
      .querySelectorAll("[data-testid='initiative-mini-feed-row']");
    expect(rows).toHaveLength(3);
  });

  it("shows the blocked indicator for blocked features", () => {
    features.current = [makeFeature({ id: "b1", status: "blocked" })];
    render(<InitiativesTilesPage />);
    expect(
      screen.getByTestId("initiative-tile-b1").querySelector("[data-testid='blocked-indicator']"),
    ).toBeInTheDocument();
  });
});
