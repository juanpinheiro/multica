import { queryOptions } from "@tanstack/react-query";
import { api } from "../api";

export const dodKeys = {
  all: (wsId: string) => ["dod", wsId] as const,
  milestone: (wsId: string, milestoneId: string) =>
    [...dodKeys.all(wsId), "milestone", milestoneId] as const,
  issue: (wsId: string, issueId: string) => [...dodKeys.all(wsId), "issue", issueId] as const,
};

// A Milestone's DoD assertions with their latest validation status.
export function milestoneDodOptions(wsId: string, milestoneId: string) {
  return queryOptions({
    queryKey: dodKeys.milestone(wsId, milestoneId),
    queryFn: () => api.listMilestoneDoD(milestoneId),
    select: (data) => data.assertions,
  });
}

// The per-Issue Acceptance Criteria view: the DoD assertions of its Milestone.
export function issueDodOptions(wsId: string, issueId: string) {
  return queryOptions({
    queryKey: dodKeys.issue(wsId, issueId),
    queryFn: () => api.listIssueDoD(issueId),
    select: (data) => data.assertions,
  });
}
