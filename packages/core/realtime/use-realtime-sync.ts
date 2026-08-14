"use client";

import { useEffect, useRef } from "react";
import { useQueryClient } from "@tanstack/react-query";
import type { StoreApi, UseBoundStore } from "zustand";
import type { WSClient } from "../api/ws-client";
import type { AuthState } from "../auth/store";
import { issueKeys } from "../issues/queries";
import { projectKeys } from "../projects/queries";
import { taskKeys } from "../tasks/queries";
import { labelKeys } from "../labels/queries";
import { propertyKeys } from "../properties/queries";
import { pinKeys } from "../pins/queries";
import { implementationKnowledgeKeys } from "../implementation-knowledge/queries";
import { workspaceKeys } from "../workspace/queries";
import { getCurrentWsId } from "../platform/workspace-storage";

export interface RealtimeSyncStores {
  authStore: UseBoundStore<StoreApi<AuthState>>;
}

/**
 * React Query owns every server-backed domain. WebSocket messages are
 * therefore cache invalidation signals; the next fetch remains authoritative.
 */
export function useRealtimeSync(
  ws: WSClient | null,
  stores: RealtimeSyncStores,
  _onToast?: (message: string, type?: "info" | "error") => void,
) {
  const qc = useQueryClient();
  const previousWsRef = useRef<WSClient | null | undefined>(undefined);

  useEffect(() => {
    const previousWs = previousWsRef.current;
    previousWsRef.current = ws;
    if (!ws) return;

    const refreshWorkspaceData = () => {
      const workspaceId = getCurrentWsId();
      if (!workspaceId) return;
      void qc.invalidateQueries({ queryKey: workspaceKeys.all(workspaceId) });
      void qc.invalidateQueries({ queryKey: issueKeys.all(workspaceId) });
      // These detail caches intentionally do not carry a workspace id. They
      // therefore sit outside `issueKeys.all(workspaceId)` and must be
      // invalidated explicitly after a missed-event window or socket swap.
      void qc.invalidateQueries({ queryKey: issueKeys.timelineAll() });
      void qc.invalidateQueries({ queryKey: issueKeys.reactionsAll() });
      void qc.invalidateQueries({ queryKey: issueKeys.subscribersAll() });
      void qc.invalidateQueries({ queryKey: issueKeys.attachmentsAll() });
      void qc.invalidateQueries({ queryKey: projectKeys.all(workspaceId) });
      void qc.invalidateQueries({ queryKey: taskKeys.all(workspaceId) });
      void qc.invalidateQueries({ queryKey: workspaceKeys.skills(workspaceId) });
      void qc.invalidateQueries({ queryKey: labelKeys.all(workspaceId) });
      void qc.invalidateQueries({ queryKey: propertyKeys.all(workspaceId) });
      void qc.invalidateQueries({ queryKey: implementationKnowledgeKeys.all(workspaceId) });
      const userId = stores.authStore.getState().user?.id;
      if (userId) {
        void qc.invalidateQueries({ queryKey: pinKeys.all(workspaceId, userId) });
      }
    };

    let timer: ReturnType<typeof setTimeout> | undefined;
    const scheduleRefresh = () => {
      if (timer) clearTimeout(timer);
      timer = setTimeout(refreshWorkspaceData, 75);
    };

    const unsubscribeAny = ws.onAny((message) => {
      const domain = message.type.split(":")[0];
      if (
        domain === "workspace" ||
        domain === "member" ||
        domain === "issue" ||
        domain === "issue_labels" ||
        domain === "issue_metadata" ||
        domain === "issue_properties" ||
        domain === "comment" ||
        domain === "activity" ||
        domain === "reaction" ||
        domain === "issue_reaction" ||
        domain === "subscriber" ||
        domain === "project" ||
        domain === "task" ||
        domain === "skill" ||
        domain === "label" ||
        domain === "property" ||
        domain === "pin"
      ) {
        scheduleRefresh();
      }
    });
    const unsubscribeReconnect = ws.onReconnect(refreshWorkspaceData);
    if (previousWs !== undefined && previousWs !== ws) {
      refreshWorkspaceData();
    }

    return () => {
      if (timer) clearTimeout(timer);
      unsubscribeAny();
      unsubscribeReconnect();
    };
  }, [qc, stores.authStore, ws]);
}
