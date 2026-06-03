import { queryOptions } from "@tanstack/react-query";
import { api } from "../api";

export const decisionLogKeys = {
  all: (featureId: string) => ["decision-log", featureId] as const,
  list: (featureId: string) => [...decisionLogKeys.all(featureId), "list"] as const,
  workspaceAll: (wsId: string) => ["decision-log", "workspace", wsId] as const,
  workspaceList: (wsId: string) =>
    [...decisionLogKeys.workspaceAll(wsId), "list"] as const,
};

// The Decision Log of an Initiative: the architectural decisions a retrospective
// Run recorded at the Initiative boundary, newest first.
export function decisionLogOptions(featureId: string) {
  return queryOptions({
    queryKey: decisionLogKeys.list(featureId),
    queryFn: () => api.listDecisionLog(featureId),
    select: (data) => data.decisions,
    enabled: !!featureId,
  });
}

// The workspace-wide Decision Log across every Initiative, newest first.
// Backs the cross-Initiative Decisions surface at /{slug}/decisions.
export function workspaceDecisionsOptions(wsId: string) {
  return queryOptions({
    queryKey: decisionLogKeys.workspaceList(wsId),
    queryFn: () => api.listWorkspaceDecisions(),
    select: (data) => data.decisions,
    enabled: !!wsId,
  });
}
