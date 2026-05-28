import type { Feature } from "@multica/core/types";

export function getFeatureIssueMetrics(
  feature: Pick<Feature, "issue_count" | "done_count">,
) {
  return {
    totalCount: feature.issue_count,
    completedCount: feature.done_count,
  };
}
