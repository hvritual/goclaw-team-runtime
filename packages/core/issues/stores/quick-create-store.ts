"use client";

import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";
import { createWorkspaceAwareStorage, registerForWorkspaceRehydration } from "../../platform/workspace-storage";
import { defaultStorage } from "../../platform/storage";
import { registerDraftCleanup } from "../../drafts/cleanup-registry";

// Per-workspace memory for the issue create modal. Persisting the most recent
// project and keep-open preference removes repetitive setup without carrying
// any server-backed domain state in Zustand.
interface QuickCreateState {
  lastProjectId: string | null;
  setLastProjectId: (id: string | null) => void;
  keepOpen: boolean;
  setKeepOpen: (v: boolean) => void;
}

export const useQuickCreateStore = create<QuickCreateState>()(
  persist(
    (set) => ({
      lastProjectId: null,
      setLastProjectId: (id) => set({ lastProjectId: id }),
      keepOpen: false,
      setKeepOpen: (v) => set({ keepOpen: v }),
    }),
    {
      name: "multica_quick_create",
      storage: createJSONStorage(() => createWorkspaceAwareStorage(defaultStorage)),
    },
  ),
);

registerForWorkspaceRehydration(() => useQuickCreateStore.persist.rehydrate());

registerDraftCleanup({
  storageKey: "multica_quick_create",
  workspaceScoped: true,
  // Reset preferences so they do not survive into the next login on the same
  // browser tab.
  resetInMemory: () =>
    useQuickCreateStore.setState({
      lastProjectId: null,
      keepOpen: false,
    }),
});
