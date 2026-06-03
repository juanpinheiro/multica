import { describe, it, expect } from "vitest";
import type { AgentTask, InboxItem, Issue } from "../types";
import { buildLiveEvents } from "./build-live-events";

const NOW = new Date("2026-06-03T12:00:00Z").getTime();
const iso = (ms: number) => new Date(ms).toISOString();

function makeTask(overrides: Partial<AgentTask> = {}): AgentTask {
  return {
    id: "task-1",
    agent_id: "agent-1",
    runtime_id: "rt-1",
    issue_id: "issue-1",
    status: "running",
    priority: 0,
    dispatched_at: iso(NOW - 60_000),
    started_at: iso(NOW - 30_000),
    completed_at: null,
    result: null,
    error: null,
    created_at: iso(NOW - 60_000),
    ...overrides,
  };
}

function makeIssue(overrides: Partial<Issue> = {}): Issue {
  return {
    id: "issue-1",
    workspace_id: "ws-1",
    number: 1,
    identifier: "MUL-1",
    title: "Issue",
    description: null,
    status: "in_progress",
    priority: "none",
    assignee_type: null,
    assignee_id: null,
    creator_type: "member",
    creator_id: "u-1",
    parent_issue_id: null,
    feature_id: "feature-1",
    position: 0,
    start_date: null,
    due_date: null,
    metadata: {},
    created_at: iso(NOW - 86400_000),
    updated_at: iso(NOW - 1_000),
    ...overrides,
  };
}

function makeInbox(overrides: Partial<InboxItem> = {}): InboxItem {
  return {
    id: "inbox-1",
    workspace_id: "ws-1",
    recipient_type: "member",
    recipient_id: "u-1",
    actor_type: "system",
    actor_id: null,
    type: "task_completed",
    severity: "info",
    issue_id: "issue-1",
    title: "Task completed",
    body: null,
    issue_status: null,
    read: false,
    archived: false,
    created_at: iso(NOW - 5_000),
    details: null,
    ...overrides,
  };
}

describe("buildLiveEvents", () => {
  it("returns an empty result for empty inputs", () => {
    const result = buildLiveEvents({ tasks: [], inbox: [], issues: [], now: NOW });
    expect(result).toEqual({ events: [], runningAgents: 0, runningInitiatives: 0 });
  });

  it("emits an agent_started event for a running task", () => {
    const result = buildLiveEvents({
      tasks: [makeTask()],
      inbox: [],
      issues: [makeIssue()],
      now: NOW,
    });
    expect(result.events).toHaveLength(1);
    expect(result.events[0]).toMatchObject({
      type: "agent_started",
      initiativeId: "feature-1",
      issueId: "issue-1",
      agentId: "agent-1",
    });
  });

  it("emits both agent_started and tool_use when last_activity_at is set", () => {
    const result = buildLiveEvents({
      tasks: [makeTask({ last_activity_at: iso(NOW - 2_000) })],
      inbox: [],
      issues: [makeIssue()],
      now: NOW,
    });
    const types = result.events.map((e) => e.type);
    expect(types).toContain("agent_started");
    expect(types).toContain("tool_use");
  });

  it("orders events newest-first by their timestamp", () => {
    const result = buildLiveEvents({
      tasks: [],
      inbox: [
        makeInbox({ id: "a", created_at: iso(NOW - 60_000) }),
        makeInbox({ id: "b", created_at: iso(NOW - 1_000) }),
        makeInbox({ id: "c", created_at: iso(NOW - 30_000) }),
      ],
      issues: [makeIssue()],
      now: NOW,
    });
    expect(result.events.map((e) => e.id)).toEqual([
      "inbox:b",
      "inbox:c",
      "inbox:a",
    ]);
  });

  it("dedupes events sharing the same id", () => {
    const result = buildLiveEvents({
      tasks: [],
      inbox: [makeInbox({ id: "dup" }), makeInbox({ id: "dup" })],
      issues: [makeIssue()],
      now: NOW,
    });
    expect(result.events).toHaveLength(1);
  });

  it("threads phase and heartbeat through tool_use events via deriveLiveness", () => {
    const last = iso(NOW - 1_500);
    const result = buildLiveEvents({
      tasks: [makeTask({ status: "running", last_activity_at: last })],
      inbox: [],
      issues: [makeIssue()],
      now: NOW,
    });
    const toolUse = result.events.find((e) => e.type === "tool_use")!;
    expect(toolUse.phase).toBe("run");
    expect(toolUse.heartbeat).toBe("fresh");
  });

  it("marks heartbeat quiet when last_activity_at is beyond the threshold", () => {
    const stale = iso(NOW - 60_000);
    const result = buildLiveEvents({
      tasks: [makeTask({ status: "running", last_activity_at: stale })],
      inbox: [],
      issues: [makeIssue()],
      now: NOW,
    });
    const toolUse = result.events.find((e) => e.type === "tool_use")!;
    expect(toolUse.heartbeat).toBe("quiet");
  });

  it("uses claim phase for waiting_local_directory tasks", () => {
    const result = buildLiveEvents({
      tasks: [
        makeTask({
          status: "waiting_local_directory",
          last_activity_at: iso(NOW - 1_000),
        }),
      ],
      inbox: [],
      issues: [makeIssue()],
      now: NOW,
    });
    const toolUse = result.events.find((e) => e.type === "tool_use")!;
    expect(toolUse.phase).toBe("claim");
  });

  it("skips inbox items whose type is not in the live-feed mapping", () => {
    const result = buildLiveEvents({
      tasks: [],
      inbox: [makeInbox({ id: "x", type: "new_comment" })],
      issues: [makeIssue()],
      now: NOW,
    });
    expect(result.events).toHaveLength(0);
  });

  it("maps inbox types to their activity event types", () => {
    const result = buildLiveEvents({
      tasks: [],
      inbox: [
        makeInbox({ id: "a", type: "task_completed" }),
        makeInbox({ id: "b", type: "task_failed" }),
        makeInbox({ id: "c", type: "initiative_tripwire" }),
        makeInbox({ id: "d", type: "feature_ready_for_review" }),
      ],
      issues: [makeIssue()],
      now: NOW,
    });
    const byId = new Map(result.events.map((e) => [e.id, e.type]));
    expect(byId.get("inbox:a")).toBe("issue_done");
    expect(byId.get("inbox:b")).toBe("dod_failed");
    expect(byId.get("inbox:c")).toBe("tripwire_paused");
    expect(byId.get("inbox:d")).toBe("initiative_ready_for_review");
  });

  it("resolves initiativeId from inbox details when present", () => {
    const result = buildLiveEvents({
      tasks: [],
      inbox: [
        makeInbox({
          id: "x",
          type: "initiative_tripwire",
          issue_id: null,
          details: { feature_id: "feature-7" },
        }),
      ],
      issues: [],
      now: NOW,
    });
    expect(result.events[0]!.initiativeId).toBe("feature-7");
  });

  it("falls back to issue.feature_id when inbox details omit feature_id", () => {
    const result = buildLiveEvents({
      tasks: [],
      inbox: [makeInbox({ id: "x", issue_id: "issue-1" })],
      issues: [makeIssue({ id: "issue-1", feature_id: "feature-2" })],
      now: NOW,
    });
    expect(result.events[0]!.initiativeId).toBe("feature-2");
  });

  it("counts runningAgents from running and waiting_local_directory tasks", () => {
    const result = buildLiveEvents({
      tasks: [
        makeTask({ id: "t1", status: "running", issue_id: "issue-1" }),
        makeTask({ id: "t2", status: "waiting_local_directory", issue_id: "issue-1" }),
        makeTask({ id: "t3", status: "completed", issue_id: "issue-1" }),
      ],
      inbox: [],
      issues: [makeIssue()],
      now: NOW,
    });
    expect(result.runningAgents).toBe(2);
  });

  it("counts runningInitiatives as the distinct feature ids of running tasks", () => {
    const result = buildLiveEvents({
      tasks: [
        makeTask({ id: "t1", status: "running", issue_id: "issue-a" }),
        makeTask({ id: "t2", status: "running", issue_id: "issue-b" }),
        makeTask({ id: "t3", status: "running", issue_id: "issue-c" }),
      ],
      inbox: [],
      issues: [
        makeIssue({ id: "issue-a", feature_id: "feature-1" }),
        makeIssue({ id: "issue-b", feature_id: "feature-1" }),
        makeIssue({ id: "issue-c", feature_id: "feature-2" }),
      ],
      now: NOW,
    });
    expect(result.runningInitiatives).toBe(2);
  });

  it("agent event for an unlinked issue still surfaces with initiativeId null", () => {
    const result = buildLiveEvents({
      tasks: [makeTask({ issue_id: "orphan" })],
      inbox: [],
      issues: [],
      now: NOW,
    });
    expect(result.events[0]!.initiativeId).toBeNull();
  });
});
