"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { ApiError, errorCode } from "../api/client";
import { executeTeamControlCommand } from "./client";
import { teamControlKeys } from "./queries";
import type { TeamControlCommandInput } from "./types";

export function isTeamControlConflict(error: unknown): boolean {
  return error instanceof ApiError
    && (error.status === 409 || errorCode(error) === "conflict");
}

export function useTeamControlCommand(
  workspaceId: string,
  projectId: string,
) {
  const queryClient = useQueryClient();
  const projectionKey = teamControlKeys.projection(workspaceId, projectId);

  return useMutation({
    mutationFn: (command: TeamControlCommandInput) =>
      executeTeamControlCommand(workspaceId, projectId, command),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: projectionKey });
    },
    onError: async (error) => {
      if (isTeamControlConflict(error)) {
        await queryClient.invalidateQueries({ queryKey: projectionKey });
      }
    },
  });
}
