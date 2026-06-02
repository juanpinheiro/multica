import { queryOptions } from "@tanstack/react-query";
import { api } from "../api";

export const milestoneKeys = {
  all: (wsId: string) => ["milestones", wsId] as const,
  list: (wsId: string) => [...milestoneKeys.all(wsId), "list"] as const,
  feature: (wsId: string, featureId: string) =>
    [...milestoneKeys.all(wsId), "feature", featureId] as const,
};

export function milestoneListOptions(wsId: string) {
  return queryOptions({
    queryKey: milestoneKeys.list(wsId),
    queryFn: () => api.listMilestones(),
    select: (data) => data.milestones,
  });
}

export function featureMilestonesOptions(wsId: string, featureId: string) {
  return queryOptions({
    queryKey: milestoneKeys.feature(wsId, featureId),
    queryFn: () => api.listMilestones({ featureId }),
    select: (data) => [...data.milestones].sort((a, b) => a.position - b.position),
  });
}
