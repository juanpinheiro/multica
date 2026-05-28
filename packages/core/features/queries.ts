import { queryOptions } from "@tanstack/react-query";
import { api } from "../api";

export const featureKeys = {
  all: (wsId: string) => ["features", wsId] as const,
  list: (wsId: string) => [...featureKeys.all(wsId), "list"] as const,
  detail: (wsId: string, id: string) =>
    [...featureKeys.all(wsId), "detail", id] as const,
  issues: (wsId: string, id: string) =>
    [...featureKeys.all(wsId), "issues", id] as const,
};

export function featureListOptions(wsId: string) {
  return queryOptions({
    queryKey: featureKeys.list(wsId),
    queryFn: () => api.listFeatures(),
    select: (data) => data.features,
  });
}

export function featureDetailOptions(wsId: string, id: string) {
  return queryOptions({
    queryKey: featureKeys.detail(wsId, id),
    queryFn: () => api.getFeature(id),
  });
}

export function featureIssuesOptions(wsId: string, id: string) {
  return queryOptions({
    queryKey: featureKeys.issues(wsId, id),
    queryFn: () => api.getFeatureIssues(id),
  });
}
