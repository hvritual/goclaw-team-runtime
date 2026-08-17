import { queryOptions } from "@tanstack/react-query";
import { api } from "../api";
import type { TaskStatus } from "../types/task";

export interface TaskListFilters {
  project_id?: string;
  issue_id?: string;
  status?: TaskStatus;
}

export const taskKeys = {
  all: (workspaceId: string) => ["tasks", workspaceId] as const,
  list: (workspaceId: string, filters?: TaskListFilters) =>
    filters
      ? ([...taskKeys.all(workspaceId), "list", filters] as const)
      : ([...taskKeys.all(workspaceId), "list"] as const),
  detail: (workspaceId: string, id: string) =>
    [...taskKeys.all(workspaceId), "detail", id] as const,
};

export function taskListOptions(workspaceId: string, filters?: TaskListFilters) {
  return queryOptions({
    queryKey: taskKeys.list(workspaceId, filters),
    queryFn: () => api.listTasks(filters),
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
