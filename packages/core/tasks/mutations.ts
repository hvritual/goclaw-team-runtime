import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useRef } from "react";
import type { CreateTaskRequest, PromoteTaskRequest, ReorderTasksRequest, UpdateTaskRequest } from "../types/task";
import { api } from "../api";
import { useWorkspaceId } from "../hooks";
import { issueKeys } from "../issues/queries";
import { taskKeys } from "./queries";
import { createSafeId } from "../utils";

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

export function usePromoteTask() {
  const queryClient = useQueryClient();
  const workspaceId = useWorkspaceId();
  const retryKeys = useRef(new Map<string, string>());
  const commandSignature = (request: { id: string } & PromoteTaskRequest) =>
    JSON.stringify([request.id, request.expected_revision, request.complete_task ?? false]);
  return useMutation({
    mutationFn: ({ id, ...request }: { id: string } & PromoteTaskRequest) => {
      const signature = commandSignature({ id, ...request });
      const idempotencyKey = request.idempotency_key?.trim() || retryKeys.current.get(signature) || createSafeId();
      retryKeys.current.set(signature, idempotencyKey);
      return api.promoteTask(id, { ...request, idempotency_key: idempotencyKey });
    },
    onSuccess: (_data, variables) => {
      retryKeys.current.delete(commandSignature(variables));
    },
    onSettled: (_data, _error, variables) => {
      queryClient.invalidateQueries({
        queryKey: taskKeys.detail(workspaceId, variables.id),
      });
      queryClient.invalidateQueries({ queryKey: taskKeys.all(workspaceId) });
      queryClient.invalidateQueries({ queryKey: issueKeys.all(workspaceId) });
    },
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
