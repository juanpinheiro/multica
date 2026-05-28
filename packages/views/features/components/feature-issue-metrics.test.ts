import { describe, expect, it } from "vitest";
import { getFeatureIssueMetrics } from "./feature-issue-metrics";

describe("getFeatureIssueMetrics", () => {
  it("surfaces project-level totals from the feature record", () => {
    const metrics = getFeatureIssueMetrics({ issue_count: 9, done_count: 5 });

    expect(metrics).toEqual({
      totalCount: 9,
      completedCount: 5,
    });
  });
});
