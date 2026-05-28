"use client";

import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";
import { createWorkspaceAwareStorage, registerForWorkspaceRehydration } from "../../platform/workspace-storage";
import { defaultStorage } from "../../platform/storage";

export type FeatureViewMode = "compact" | "comfortable";

export interface FeatureViewState {
  viewMode: FeatureViewMode;
  setViewMode: (mode: FeatureViewMode) => void;
}

export const useFeatureViewStore = create<FeatureViewState>()(
  persist(
    (set) => ({
      viewMode: "compact",
      setViewMode: (mode) => set({ viewMode: mode }),
    }),
    {
      name: "multica_projects_view",
      storage: createJSONStorage(() => createWorkspaceAwareStorage(defaultStorage)),
      partialize: (state) => ({ viewMode: state.viewMode }),
      merge: (persisted, current) => {
        if (!persisted) return { ...current, viewMode: "compact" };
        return { ...current, ...(persisted as Partial<FeatureViewState>) };
      },
    }
  )
);

registerForWorkspaceRehydration(() => useFeatureViewStore.persist.rehydrate());