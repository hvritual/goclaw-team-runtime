import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { CreateTaskRequest, ReorderTasksRequest, UpdateTaskRequest } from "../types/task";
import { api } from "../api";
import { useWorkspaceId } from "../hooks";
import { taskKeys } from "./queries";

export function useCreateTask() {
  const queryClient = useQueryClient();
  const workspaceId = useWorkspaceId();
  return useMutation({
    mutationFn: (request: CreateTaskRequest) => api.createTask(request),
    onSettled: () =>
      queryClient.invalidateQueries({ queryKey: taskKeys.all(workspaceId) }),
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
      queryClient.invalidateQueries({ queryKey: taskKeys.all(workspaceId) });
    },
  });
}

export function useDeleteTask() {
  const queryClient = useQueryClient();
  const workspaceId = useWorkspaceId();
  return useMutation({
    mutationFn: ({ id, expectedRevision }: { id: string; expectedRevision: number }) =>
      api.deleteTask(id, expectedRevision),
    onSettled: () =>
      queryClient.invalidateQueries({ queryKey: taskKeys.all(workspaceId) }),
  });
}

export function useRestoreTask() {
  const queryClient = useQueryClient();
  const workspaceId = useWorkspaceId();
  return useMutation({
    mutationFn: ({ id, expectedRevision }: { id: string; expectedRevision: number }) =>
      api.restoreTask(id, expectedRevision),
    onSettled: (_data, _error, variables) => {
      queryClient.invalidateQueries({ queryKey: taskKeys.detail(workspaceId, variables.id) });
      queryClient.invalidateQueries({ queryKey: taskKeys.all(workspaceId) });
    },
  });
}

export function useReorderTasks() {
  const queryClient = useQueryClient();
  const workspaceId = useWorkspaceId();
  return useMutation({
    mutationFn: (request: ReorderTasksRequest) => api.reorderTasks(request),
    onSettled: () => queryClient.invalidateQueries({ queryKey: taskKeys.all(workspaceId) }),
  });
}
