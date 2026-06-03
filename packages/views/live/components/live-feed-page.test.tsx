import { render, screen, fireEvent } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type {
  ActivityEvent,
  BuildLiveEventsResult,
} from "@multica/core/tasks/build-live-events";
import layout from "../../locales/en/layout.json";
import { LiveFeedPage } from "./live-feed-page";

const { liveResult, tasks, issues, features, navPush } = vi.hoisted(() => ({
  liveResult: { current: null as BuildLiveEventsResult | null },
  tasks: { current: [] as Array<Record<string, unknown>> },
  issues: { current: [] as Array<Record<string, unknown>> },
  features: { current: [] as Array<Record<string, unknown>> },
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

vi.mock("@multica/core/agents/queries", () => ({
  agentTaskSnapshotOptions: () => ({ queryKey: ["task-snapshot"] }),
}));

vi.mock("@multica/core/issues/queries", () => ({
  issueListOptions: () => ({ queryKey: ["issues"] }),
}));

vi.mock("@multica/core/features/queries", () => ({
  featureListOptions: () => ({ queryKey: ["features"] }),
}));

vi.mock("@tanstack/react-query", () => ({
  useQuery: ({ queryKey }: { queryKey: readonly unknown[] }) => {
    if (queryKey[0] === "task-snapshot") return { data: tasks.current };
    if (queryKey[0] === "issues") return { data: issues.current };
    if (queryKey[0] === "features") return { data: features.current };
    return { data: [] };
  },
}));

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

function seedResult(events: ActivityEvent[], runningAgents = 0, runningInitiatives = 0) {
  liveResult.current = { events, runningAgents, runningInitiatives };
}

const FEATURE = { id: "feature-1", title: "Refactor checkout" };
const ISSUE = {
  id: "issue-1",
  identifier: "MUL-12",
  title: "Wire SetupIntent",
  feature_id: "feature-1",
};

describe("LiveFeedPage", () => {
  beforeEach(() => {
    navPush.mockClear();
    features.current = [FEATURE];
    issues.current = [ISSUE];
    tasks.current = [];
  });

  it("renders the headline with the running-agents count", () => {
    seedResult([], 3, 2);
    render(<LiveFeedPage />);
    expect(screen.getByTestId("live-headline")).toHaveTextContent("3 agents at work");
  });

  it("shows the empty state when there are no events", () => {
    seedResult([], 0, 0);
    render(<LiveFeedPage />);
    expect(screen.getByTestId("live-empty")).toBeInTheDocument();
  });

  it("renders events newest-first in the order returned by the aggregator", () => {
    seedResult([
      makeEvent({ id: "a", message: "first", at: "2026-06-03T11:59:50Z" }),
      makeEvent({ id: "b", message: "second", at: "2026-06-03T11:59:40Z" }),
      makeEvent({ id: "c", message: "third", at: "2026-06-03T11:59:30Z" }),
    ]);
    render(<LiveFeedPage />);
    const rows = screen.getAllByTestId("live-event-row");
    expect(rows.map((r) => r.textContent)).toEqual([
      expect.stringContaining("first"),
      expect.stringContaining("second"),
      expect.stringContaining("third"),
    ]);
  });

  it("Failures filter hides non-failure events", () => {
    seedResult([
      makeEvent({ id: "ok", type: "agent_started", message: "running" }),
      makeEvent({ id: "fail", type: "dod_failed", message: "DoD failed" }),
    ]);
    render(<LiveFeedPage />);
    fireEvent.click(screen.getByTestId("live-filter-failures"));
    const rows = screen.getAllByTestId("live-event-row");
    expect(rows).toHaveLength(1);
    expect(rows[0]!.dataset.eventType).toBe("dod_failed");
  });

  it("Live now filter shows only running event types", () => {
    seedResult([
      makeEvent({ id: "a", type: "agent_started" }),
      makeEvent({ id: "b", type: "tool_use" }),
      makeEvent({ id: "c", type: "issue_done" }),
      makeEvent({ id: "d", type: "tripwire_paused" }),
    ]);
    render(<LiveFeedPage />);
    fireEvent.click(screen.getByTestId("live-filter-running"));
    const types = screen.getAllByTestId("live-event-row").map((r) => r.dataset.eventType);
    expect(types).toEqual(expect.arrayContaining(["agent_started", "tool_use"]));
    expect(types).not.toContain("issue_done");
    expect(types).not.toContain("tripwire_paused");
  });

  it("clicking an initiative chip navigates to the initiative detail path", () => {
    seedResult([makeEvent({ id: "x" })]);
    render(<LiveFeedPage />);
    fireEvent.click(screen.getByTestId("live-event-initiative-chip"));
    expect(navPush).toHaveBeenCalledWith("/acme/initiatives/feature-1");
  });

  it("renders a live-now chip per running task with an issue", () => {
    tasks.current = [
      {
        id: "t1",
        status: "running",
        issue_id: "issue-1",
        agent_id: "a1",
        runtime_id: "r1",
      },
    ];
    seedResult([], 1, 1);
    render(<LiveFeedPage />);
    expect(screen.getByTestId("live-now-chip")).toHaveTextContent("MUL-12");
  });
});
