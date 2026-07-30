import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { CreateTaskRequest, UpdateTaskRequest } from "../types";
import { api } from "../api";
import { useWorkspaceId } from "../hooks";
import { taskKeys } from "./queries";

export function useCreateTask() {
  const queryClient = useQueryClient();
  const workspaceId = useWorkspaceId();
  return useMutation({
    mutationFn: (request: CreateTaskRequest) => api.createTask(request),
    onSettled: () =>
      queryClient.invalidateQueries({ queryKey: taskKeys.list(workspaceId) }),
  });
}

export function useUpdateTask() {
  const queryClient = useQueryClient();
  const workspaceId = useWorkspaceId();
  return useMutation({
    mutationFn: ({
      id,
      ...request
    }: { id: string } & UpdateTaskRequest) => api.updateTask(id, request),
    onSettled: (_data, _error, variables) => {
      queryClient.invalidateQueries({
        queryKey: taskKeys.detail(workspaceId, variables.id),
      });
      queryClient.invalidateQueries({ queryKey: taskKeys.list(workspaceId) });
    },
  });
}

export function useDeleteTask() {
  const queryClient = useQueryClient();
  const workspaceId = useWorkspaceId();
  return useMutation({
    mutationFn: (id: string) => api.deleteTask(id),
    onSettled: () =>
      queryClient.invalidateQueries({ queryKey: taskKeys.list(workspaceId) }),
  });
}
