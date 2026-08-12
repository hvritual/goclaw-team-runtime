"use client";

import { useEffect, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { streamTeamControlEvents } from "./client";
import { teamControlKeys } from "./queries";
import type { TeamControlConnectionState } from "./types";

const RECONNECT_DELAY_MS = 1_000;

export function useTeamControlEvents(
  workspaceId: string,
  projectId: string,
): TeamControlConnectionState {
  const queryClient = useQueryClient();
  const [state, setState] = useState<TeamControlConnectionState>("connecting");

  useEffect(() => {
    const controller = new AbortController();
    const projectionKey = teamControlKeys.projection(workspaceId, projectId);
    let cursor = 0;
    let reconnectTimer: ReturnType<typeof setTimeout> | undefined;

    const connect = async (reconnecting: boolean) => {
      setState(reconnecting ? "reconnecting" : "connecting");
      try {
        cursor = await streamTeamControlEvents(workspaceId, projectId, {
          after: cursor,
          signal: controller.signal,
          onOpen: () => setState("connected"),
          onEvent: (event) => {
            if (
              event.workspace_id === workspaceId
              && event.project_id === projectId
            ) {
              void queryClient.invalidateQueries({ queryKey: projectionKey });
            }
          },
        });
        if (!controller.signal.aborted) {
          reconnectTimer = setTimeout(() => void connect(true), RECONNECT_DELAY_MS);
        }
      } catch {
        if (controller.signal.aborted) return;
        setState(typeof navigator !== "undefined" && navigator.onLine === false
          ? "offline"
          : "reconnecting");
        reconnectTimer = setTimeout(() => void connect(true), RECONNECT_DELAY_MS);
      }
    };

    void connect(false);
    return () => {
      controller.abort();
      if (reconnectTimer) clearTimeout(reconnectTimer);
    };
  }, [projectId, queryClient, workspaceId]);

  return state;
}
