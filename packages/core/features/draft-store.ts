import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";
import type { FeatureStatus, FeaturePriority } from "../types";
import { createWorkspaceAwareStorage, registerForWorkspaceRehydration } from "../platform/workspace-storage";
import { defaultStorage } from "../platform/storage";

interface FeatureDraft {
  title: string;
  description: string;
  status: FeatureStatus;
  priority: FeaturePriority;
  leadType?: "member" | "agent";
  leadId?: string;
  icon?: string;
}

const EMPTY_DRAFT: FeatureDraft = {
  title: "",
  description: "",
  status: "planned",
  priority: "none",
  leadType: undefined,
  leadId: undefined,
  icon: undefined,
};

interface FeatureDraftStore {
  draft: FeatureDraft;
  setDraft: (patch: Partial<FeatureDraft>) => void;
  clearDraft: () => void;
  hasDraft: () => boolean;
}

export const useFeatureDraftStore = create<FeatureDraftStore>()(
  persist(
    (set, get) => ({
      draft: { ...EMPTY_DRAFT },
      setDraft: (patch) =>
        set((s) => ({ draft: { ...s.draft, ...patch } })),
      clearDraft: () =>
        set({ draft: { ...EMPTY_DRAFT } }),
      hasDraft: () => {
        const { draft } = get();
        return !!(draft.title || draft.description);
      },
    }),
    {
      name: "multica_project_draft",
      storage: createJSONStorage(() => createWorkspaceAwareStorage(defaultStorage)),
    },
  ),
);

registerForWorkspaceRehydration(() => useFeatureDraftStore.persist.rehydrate());
