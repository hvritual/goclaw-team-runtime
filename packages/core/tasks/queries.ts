import { queryOptions } from "@tanstack/react-query";
import { api } from "../api";

export const taskKeys = {
  all: (workspaceId: string) => ["tasks", workspaceId] as const,
  list: (workspaceId: string) =>
    [...taskKeys.all(workspaceId), "list"] as const,
  detail: (workspaceId: string, id: string) =>
    [...taskKeys.all(workspaceId), "detail", id] as const,
};

export function taskListOptions(workspaceId: string) {
  return queryOptions({
    queryKey: taskKeys.list(workspaceId),
    queryFn: () => api.listTasks(),
    select: (response) => response.tasks,
  });
}

export function taskDetailOptions(workspaceId: string, id: string) {
  return queryOptions({
    queryKey: taskKeys.detail(workspaceId, id),
    queryFn: () => api.getTask(id),
    enabled: Boolean(id),
  });
}
