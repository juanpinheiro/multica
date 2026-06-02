import { queryOptions } from "@tanstack/react-query";
import { api } from "../api";

export const decisionLogKeys = {
  all: (featureId: string) => ["decision-log", featureId] as const,
  list: (featureId: string) => [...decisionLogKeys.all(featureId), "list"] as const,
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
