import { describe, expect, it } from "vitest";
import { deriveActivityCounters } from "./derive-activity-counters";

const NOW = new Date("2026-05-31T12:00:00Z").getTime();
const STARTED = new Date("2026-05-31T11:55:00Z").getTime();
const startedAt = new Date(STARTED).toISOString();

describe("deriveActivityCounters", () => {
  describe("activityCount", () => {
    it("counts tool_use events", () => {
      const msgs = [{ type: "tool_use" }, { type: "tool_use" }, { type: "text" }] as const;
      expect(deriveActivityCounters(msgs, startedAt, NOW).activityCount).toBe(2);
    });

    it("ignores text events", () => {
      expect(deriveActivityCounters([{ type: "text" }], startedAt, NOW).activityCount).toBe(0);
    });

    it("ignores thinking events", () => {
      expect(deriveActivityCounters([{ type: "thinking" }], startedAt, NOW).activityCount).toBe(0);
    });

    it("ignores tool_result events", () => {
      expect(deriveActivityCounters([{ type: "tool_result" }], startedAt, NOW).activityCount).toBe(0);
    });

    it("ignores error events", () => {
      expect(deriveActivityCounters([{ type: "error" }], startedAt, NOW).activityCount).toBe(0);
    });

    it("returns 0 for an empty timeline", () => {
      expect(deriveActivityCounters([], startedAt, NOW).activityCount).toBe(0);
    });

    it("grows monotonically as tool_use events accumulate", () => {
      const fewer = deriveActivityCounters([{ type: "tool_use" }], startedAt, NOW);
      const more = deriveActivityCounters(
        [{ type: "tool_use" }, { type: "tool_use" }],
        startedAt,
        NOW,
      );
      expect(more.activityCount).toBeGreaterThan(fewer.activityCount);
    });
  });

  describe("elapsedMs", () => {
    it("is computed from started_at to now", () => {
      expect(deriveActivityCounters([], startedAt, NOW).elapsedMs).toBe(NOW - STARTED);
    });

    it("is 0 when started_at is null", () => {
      expect(deriveActivityCounters([], null, NOW).elapsedMs).toBe(0);
    });

    it("is never negative", () => {
      const futureStart = new Date(NOW + 1000).toISOString();
      expect(
        deriveActivityCounters([], futureStart, NOW).elapsedMs,
      ).toBeGreaterThanOrEqual(0);
    });

    it("grows as now advances", () => {
      const earlier = deriveActivityCounters([], startedAt, NOW);
      const later = deriveActivityCounters([], startedAt, NOW + 5_000);
      expect(later.elapsedMs).toBeGreaterThan(earlier.elapsedMs);
      expect(later.elapsedMs - earlier.elapsedMs).toBe(5_000);
    });
  });
});
