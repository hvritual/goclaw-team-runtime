"use client";

import { useEffect, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { ApiError } from "../api";
import { streamTeamControlEvents } from "./client";
import { teamControlKeys } from "./queries";
import type { TeamControlConnectionState } from "./types";

const RECONNECT_BASE_MS = 1_000;
const RECONNECT_MAX_MS = 30_000;

export function useTeamControlEvents(
  workspaceId: string,
  projectId: string,
  options: { enabled?: boolean; initialCursor?: number } = {},
): TeamControlConnectionState {
  const queryClient = useQueryClient();
  const [state, setState] = useState<TeamControlConnectionState>("connecting");

  useEffect(() => {
    if (options.enabled === false) {
      setState("offline");
      return;
    }
    const controller = new AbortController();
    const projectionKey = teamControlKeys.projection(workspaceId, projectId);
    let cursor = options.initialCursor ?? 0;
    let reconnectTimer: ReturnType<typeof setTimeout> | undefined;
    let reconnectAttempt = 0;

    const scheduleReconnect = () => {
      const base = Math.min(RECONNECT_MAX_MS, RECONNECT_BASE_MS * 2 ** reconnectAttempt);
      reconnectAttempt++;
      const jitter = Math.floor(Math.random() * Math.min(250, base / 4));
      reconnectTimer = setTimeout(() => void connect(true), base + jitter);
    };

    const connect = async (reconnecting: boolean) => {
      setState(reconnecting ? "reconnecting" : "connecting");
      try {
        cursor = await streamTeamControlEvents(workspaceId, projectId, {
          after: cursor,
          signal: controller.signal,
          onOpen: () => { reconnectAttempt = 0; setState("connected"); },
          onCursor: (sequence) => { cursor = sequence; },
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
          scheduleReconnect();
        }
      } catch (error) {
        if (controller.signal.aborted) return;
        if (error instanceof ApiError && [401, 403, 404].includes(error.status)) {
          setState("offline");
          return;
        }
        setState(typeof navigator !== "undefined" && navigator.onLine === false
          ? "offline"
          : "reconnecting");
        scheduleReconnect();
      }
    };

    void connect(false);
    return () => {
      controller.abort();
      if (reconnectTimer) clearTimeout(reconnectTimer);
    };
  }, [options.enabled, options.initialCursor, projectId, queryClient, workspaceId]);

  return state;
}
