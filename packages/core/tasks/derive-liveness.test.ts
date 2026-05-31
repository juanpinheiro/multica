import { describe, expect, it } from "vitest";
import type { AgentTask } from "../types";
import { deriveLiveness, QUIET_THRESHOLD_MS } from "./derive-liveness";

const NOW = new Date("2026-05-31T12:00:00Z").getTime();
const STARTED = new Date("2026-05-31T11:55:00Z").getTime(); // 5 min before NOW

function makeTask(overrides: Partial<AgentTask> = {}): AgentTask {
  return {
    id: "task-1",
    agent_id: "agent-1",
    runtime_id: "rt-1",
    issue_id: "issue-1",
    status: "running",
    priority: 0,
    dispatched_at: "2026-05-31T11:54:00Z",
    started_at: new Date(STARTED).toISOString(),
    completed_at: null,
    result: null,
    error: null,
    created_at: "2026-05-31T11:54:00Z",
    ...overrides,
  };
}

describe("deriveLiveness", () => {
  describe("active flag", () => {
    it("is true for running tasks", () => {
      expect(deriveLiveness(makeTask({ status: "running" }), NOW).active).toBe(true);
    });

    it("is true for waiting_local_directory tasks", () => {
      expect(deriveLiveness(makeTask({ status: "waiting_local_directory" }), NOW).active).toBe(true);
    });

    it("is false for completed tasks", () => {
      expect(deriveLiveness(makeTask({ status: "completed" }), NOW).active).toBe(false);
    });

    it("is false for failed tasks", () => {
      expect(deriveLiveness(makeTask({ status: "failed" }), NOW).active).toBe(false);
    });

    it("is false for cancelled tasks", () => {
      expect(deriveLiveness(makeTask({ status: "cancelled" }), NOW).active).toBe(false);
    });

    it("is false for queued tasks", () => {
      expect(deriveLiveness(makeTask({ status: "queued" }), NOW).active).toBe(false);
    });

    it("is false for dispatched tasks", () => {
      expect(deriveLiveness(makeTask({ status: "dispatched" }), NOW).active).toBe(false);
    });
  });

  describe("phase", () => {
    it("is run for running tasks", () => {
      expect(deriveLiveness(makeTask({ status: "running" }), NOW).phase).toBe("run");
    });

    it("is claim for queued tasks", () => {
      expect(deriveLiveness(makeTask({ status: "queued" }), NOW).phase).toBe("claim");
    });

    it("is claim for dispatched tasks", () => {
      expect(deriveLiveness(makeTask({ status: "dispatched" }), NOW).phase).toBe("claim");
    });

    it("is claim for waiting_local_directory tasks (blocked at claim)", () => {
      expect(deriveLiveness(makeTask({ status: "waiting_local_directory" }), NOW).phase).toBe("claim");
    });

    it("is pr for running task when issueStatus is in_review", () => {
      expect(
        deriveLiveness(makeTask({ status: "running" }), NOW, { issueStatus: "in_review" }).phase,
      ).toBe("pr");
    });

    it("is pr for running task when hasPr is true", () => {
      expect(
        deriveLiveness(makeTask({ status: "running" }), NOW, { hasPr: true }).phase,
      ).toBe("pr");
    });

    it("is run for running task with non-review issueStatus", () => {
      expect(
        deriveLiveness(makeTask({ status: "running" }), NOW, { issueStatus: "in_progress" }).phase,
      ).toBe("run");
    });

    it("is claim for waiting_local_directory regardless of issueStatus", () => {
      expect(
        deriveLiveness(makeTask({ status: "waiting_local_directory" }), NOW, { issueStatus: "in_review" }).phase,
      ).toBe("claim");
    });

    it("is pr for queued task when hasPr is true", () => {
      expect(
        deriveLiveness(makeTask({ status: "queued" }), NOW, { hasPr: true }).phase,
      ).toBe("pr");
    });
  });

  describe("elapsedMs", () => {
    it("is computed from started_at when present", () => {
      const elapsed = deriveLiveness(makeTask({ started_at: new Date(STARTED).toISOString() }), NOW).elapsedMs;
      expect(elapsed).toBe(NOW - STARTED);
    });

    it("is 0 when started_at is null", () => {
      expect(deriveLiveness(makeTask({ started_at: null }), NOW).elapsedMs).toBe(0);
    });

    it("is never negative", () => {
      const futureStart = new Date(NOW + 1000).toISOString();
      expect(deriveLiveness(makeTask({ started_at: futureStart }), NOW).elapsedMs).toBeGreaterThanOrEqual(0);
    });
  });

  describe("heartbeat", () => {
    const iso = (ms: number) => new Date(ms).toISOString();

    it("is fresh when last_activity_at is within the threshold", () => {
      const recent = iso(NOW - 2_000);
      const result = deriveLiveness(makeTask({ last_activity_at: recent }), NOW);
      expect(result.heartbeat).toBe("fresh");
    });

    it("is quiet when last_activity_at is beyond the threshold", () => {
      const stale = iso(NOW - (QUIET_THRESHOLD_MS + 5_000));
      const result = deriveLiveness(makeTask({ last_activity_at: stale }), NOW);
      expect(result.heartbeat).toBe("quiet");
    });

    it("computes quietMs as the gap since last_activity_at", () => {
      const last = iso(NOW - 30_000);
      expect(deriveLiveness(makeTask({ last_activity_at: last }), NOW).quietMs).toBe(30_000);
    });

    it("grows quietMs as now advances with no new activity", () => {
      const last = iso(NOW - 5_000);
      const task = makeTask({ last_activity_at: last });
      const earlier = deriveLiveness(task, NOW).quietMs;
      const later = deriveLiveness(task, NOW + 7_000).quietMs;
      expect(later).toBeGreaterThan(earlier);
      expect(later).toBe(earlier + 7_000);
    });

    it("falls back to started_at when last_activity_at is null", () => {
      const startedRecently = iso(NOW - 2_000);
      const result = deriveLiveness(
        makeTask({ last_activity_at: null, started_at: startedRecently }),
        NOW,
      );
      expect(result.heartbeat).toBe("fresh");
      expect(result.quietMs).toBe(2_000);
    });

    it("is fresh with quietMs 0 when both last_activity_at and started_at are null", () => {
      const result = deriveLiveness(
        makeTask({ last_activity_at: null, started_at: null }),
        NOW,
      );
      expect(result.heartbeat).toBe("fresh");
      expect(result.quietMs).toBe(0);
    });

    it("never reports negative quietMs under clock skew", () => {
      const future = iso(NOW + 5_000);
      expect(
        deriveLiveness(makeTask({ last_activity_at: future }), NOW).quietMs,
      ).toBeGreaterThanOrEqual(0);
    });
  });

  describe("terminal tasks return inactive liveness", () => {
    it.each(["completed", "failed", "cancelled"] as const)(
      "%s → active: false",
      (status) => {
        const result = deriveLiveness(makeTask({ status }), NOW);
        expect(result.active).toBe(false);
      },
    );
  });

  describe("waiting field", () => {
    it("is null for a running task", () => {
      expect(deriveLiveness(makeTask({ status: "running" }), NOW).waiting).toBeNull();
    });

    it("is null for a completed task", () => {
      expect(deriveLiveness(makeTask({ status: "completed" }), NOW).waiting).toBeNull();
    });

    it("is populated for waiting_local_directory with wait_reason", () => {
      const reason = "umbrella directory /code/project is in use by task abc-123";
      const result = deriveLiveness(makeTask({ status: "waiting_local_directory", wait_reason: reason }), NOW);
      expect(result.waiting).not.toBeNull();
      expect(result.waiting!.reason).toBe(reason);
      expect(result.waiting!.holderKey).toBeNull();
    });

    it("carries holderKey from ctx when provided", () => {
      const reason = "umbrella directory /code/project is in use by task abc-123";
      const result = deriveLiveness(
        makeTask({ status: "waiting_local_directory", wait_reason: reason }),
        NOW,
        { holderKey: "MUL-42" },
      );
      expect(result.waiting!.holderKey).toBe("MUL-42");
    });

    it("falls back to a default reason when wait_reason is null", () => {
      const result = deriveLiveness(
        makeTask({ status: "waiting_local_directory", wait_reason: null }),
        NOW,
      );
      expect(result.waiting).not.toBeNull();
      expect(result.waiting!.reason).toBeTruthy();
      expect(result.waiting!.holderKey).toBeNull();
    });

    it("active is true for a waiting_local_directory task", () => {
      const result = deriveLiveness(
        makeTask({ status: "waiting_local_directory", wait_reason: "some reason" }),
        NOW,
      );
      expect(result.active).toBe(true);
      expect(result.waiting).not.toBeNull();
    });
  });
});
