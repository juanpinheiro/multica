import { describe, it, expect } from "vitest";
import { addIssueToBuckets, getBucket } from "./cache-helpers";
import type { Issue, ListIssuesCache } from "../types";

function makeIssue(id: string, status: Issue["status"], position = 0): Issue {
  return {
    id,
    workspace_id: "ws-1",
    number: 1,
    identifier: `TST-${id}`,
    title: `Issue ${id}`,
    description: null,
    status,
    priority: "medium",
    assignee_type: null,
    assignee_id: null,
    creator_type: "member",
    creator_id: "user-1",
    parent_issue_id: null,
    feature_id: null,
    position,
    start_date: null,
    due_date: null,
    metadata: {},
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
  };
}

function makeCache(issues: Issue[]): ListIssuesCache {
  const byStatus: ListIssuesCache["byStatus"] = {};
  for (const issue of issues) {
    const existing = byStatus[issue.status];
    if (existing) {
      existing.issues.push(issue);
      existing.total++;
    } else {
      byStatus[issue.status] = { issues: [issue], total: 1 };
    }
  }
  return { byStatus };
}

describe("addIssueToBuckets", () => {
  it("prepends the new issue at the start of its status bucket", () => {
    const existing = makeIssue("existing-1", "todo", 50);
    const cache = makeCache([existing]);
    const newIssue = makeIssue("new-1", "todo", 0);

    const result = addIssueToBuckets(cache, newIssue);
    const bucket = getBucket(result, "todo");

    expect(bucket.issues[0]!.id).toBe("new-1");
    expect(bucket.issues[1]!.id).toBe("existing-1");
  });

  it("increments the bucket total", () => {
    const cache = makeCache([makeIssue("old-1", "backlog", 10)]);
    const result = addIssueToBuckets(cache, makeIssue("new-1", "backlog", 0));
    expect(getBucket(result, "backlog").total).toBe(2);
  });

  it("is a no-op when the issue is already present", () => {
    const issue = makeIssue("dup", "todo", 0);
    const cache = makeCache([issue]);
    const result = addIssueToBuckets(cache, issue);
    expect(result).toBe(cache);
  });

  it("creates the bucket when the status is not yet present", () => {
    const cache = makeCache([]);
    const result = addIssueToBuckets(cache, makeIssue("first", "in_progress", 0));
    const bucket = getBucket(result, "in_progress");
    expect(bucket.issues).toHaveLength(1);
    expect(bucket.total).toBe(1);
  });
});
