import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api";
import { featureKeys } from "./queries";
import { useWorkspaceId } from "../hooks";
import type { Feature, CreateFeatureRequest, UpdateFeatureRequest, ListFeaturesResponse } from "../types";

export function useCreateFeature() {
  const qc = useQueryClient();
  const wsId = useWorkspaceId();
  return useMutation({
    mutationFn: (data: CreateFeatureRequest) => api.createFeature(data),
    onSuccess: (newFeature) => {
      qc.setQueryData<ListFeaturesResponse>(featureKeys.list(wsId), (old: ListFeaturesResponse | undefined) =>
        old && !old.features.some((p: Feature) => p.id === newFeature.id)
          ? { ...old, features: [...old.features, newFeature], total: old.total + 1 }
          : old,
      );
    },
    onSettled: () => {
      qc.invalidateQueries({ queryKey: featureKeys.list(wsId) });
    },
  });
}

export function useUpdateFeature() {
  const qc = useQueryClient();
  const wsId = useWorkspaceId();
  return useMutation({
    mutationFn: ({ id, ...data }: { id: string } & UpdateFeatureRequest) =>
      api.updateFeature(id, data),
    onMutate: ({ id, ...data }) => {
      qc.cancelQueries({ queryKey: featureKeys.list(wsId) });
      const prevList = qc.getQueryData<ListFeaturesResponse>(featureKeys.list(wsId));
      const prevDetail = qc.getQueryData<Feature>(featureKeys.detail(wsId, id));
      qc.setQueryData<ListFeaturesResponse>(featureKeys.list(wsId), (old: ListFeaturesResponse | undefined) =>
        old ? { ...old, features: old.features.map((p: Feature) => (p.id === id ? { ...p, ...data } : p)) } : old,
      );
      qc.setQueryData<Feature>(featureKeys.detail(wsId, id), (old: Feature | undefined) =>
        old ? { ...old, ...data } : old,
      );
      return { prevList, prevDetail, id };
    },
    onError: (_err, _vars, ctx) => {
      if (ctx?.prevList) qc.setQueryData(featureKeys.list(wsId), ctx.prevList);
      if (ctx?.prevDetail) qc.setQueryData(featureKeys.detail(wsId, ctx.id), ctx.prevDetail);
    },
    onSettled: (_data, _err, vars) => {
      qc.invalidateQueries({ queryKey: featureKeys.detail(wsId, vars.id) });
      qc.invalidateQueries({ queryKey: featureKeys.list(wsId) });
    },
  });
}

export function useDeleteFeature() {
  const qc = useQueryClient();
  const wsId = useWorkspaceId();
  return useMutation({
    mutationFn: (id: string) => api.deleteFeature(id),
    onMutate: async (id) => {
      await qc.cancelQueries({ queryKey: featureKeys.list(wsId) });
      const prevList = qc.getQueryData<ListFeaturesResponse>(featureKeys.list(wsId));
      qc.setQueryData<ListFeaturesResponse>(featureKeys.list(wsId), (old: ListFeaturesResponse | undefined) =>
        old ? { ...old, features: old.features.filter((p: Feature) => p.id !== id), total: old.total - 1 } : old,
      );
      qc.removeQueries({ queryKey: featureKeys.detail(wsId, id) });
      return { prevList };
    },
    onError: (_err, _id, ctx) => {
      if (ctx?.prevList) qc.setQueryData(featureKeys.list(wsId), ctx.prevList);
    },
    onSettled: () => {
      qc.invalidateQueries({ queryKey: featureKeys.list(wsId) });
    },
  });
}
