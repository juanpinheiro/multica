import { queryOptions, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api";
import { featureKeys } from "./queries";
import type {
  CreateFeatureResourceRequest,
  ListFeatureResourcesResponse,
  FeatureResource,
} from "../types";

export const featureResourceKeys = {
  list: (wsId: string, featureId: string) =>
    [...featureKeys.detail(wsId, featureId), "resources"] as const,
};

export function featureResourcesOptions(wsId: string, featureId: string) {
  return queryOptions({
    queryKey: featureResourceKeys.list(wsId, featureId),
    queryFn: () => api.listFeatureResources(featureId),
    select: (data) => data.resources,
  });
}

export function useCreateFeatureResource(wsId: string, featureId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateFeatureResourceRequest) =>
      api.createFeatureResource(featureId, data),
    onSuccess: (created) => {
      qc.setQueryData<ListFeatureResourcesResponse>(
        featureResourceKeys.list(wsId, featureId),
        (old: ListFeatureResourcesResponse | undefined) =>
          old && !old.resources.some((r: FeatureResource) => r.id === created.id)
            ? {
                ...old,
                resources: [...old.resources, created],
                total: old.total + 1,
              }
            : old,
      );
    },
    onSettled: () => {
      qc.invalidateQueries({
        queryKey: featureResourceKeys.list(wsId, featureId),
      });
    },
  });
}

export function useDeleteFeatureResource(wsId: string, featureId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (resourceId: string) =>
      api.deleteFeatureResource(featureId, resourceId),
    onMutate: async (resourceId) => {
      await qc.cancelQueries({
        queryKey: featureResourceKeys.list(wsId, featureId),
      });
      const prev = qc.getQueryData<ListFeatureResourcesResponse>(
        featureResourceKeys.list(wsId, featureId),
      );
      qc.setQueryData<ListFeatureResourcesResponse>(
        featureResourceKeys.list(wsId, featureId),
        (old: ListFeatureResourcesResponse | undefined) =>
          old
            ? {
                ...old,
                resources: old.resources.filter(
                  (r: FeatureResource) => r.id !== resourceId,
                ),
                total: old.total - 1,
              }
            : old,
      );
      return { prev };
    },
    onError: (_err, _id, ctx) => {
      if (ctx?.prev) {
        qc.setQueryData(featureResourceKeys.list(wsId, featureId), ctx.prev);
      }
    },
    onSettled: () => {
      qc.invalidateQueries({
        queryKey: featureResourceKeys.list(wsId, featureId),
      });
    },
  });
}
