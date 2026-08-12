"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { ApiError } from "../api/client";
import { executeTeamControlCommand } from "./client";
import { parseTeamControlProblem } from "./schemas";
import { teamControlKeys } from "./queries";
import type { TeamControlCommandInput } from "./types";

export function isTeamControlConflict(error: unknown): boolean {
  return error instanceof ApiError
    && (error.status === 409 || parseTeamControlProblem(error.body)?.code === "conflict");
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
