import { queryOptions, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api";
import { projectKeys } from "./queries";
import type {
  CreateProjectResourceRequest,
  UpdateProjectResourceRequest,
} from "../types";

export const projectResourceKeys = {
  all: (wsId: string, projectId: string) =>
    [...projectKeys.detail(wsId, projectId), "resources"] as const,
  list: (wsId: string, projectId: string, includeArchived: boolean) =>
    [...projectResourceKeys.all(wsId, projectId), { includeArchived }] as const,
};

export function projectResourcesOptions(
  wsId: string,
  projectId: string,
  includeArchived = true,
) {
  return queryOptions({
    queryKey: projectResourceKeys.list(wsId, projectId, includeArchived),
    queryFn: () => api.listProjectResources(projectId, { includeArchived }),
  });
}

function invalidateProjectResourceProjection(
  qc: ReturnType<typeof useQueryClient>,
  wsId: string,
  projectId: string,
) {
  qc.invalidateQueries({ queryKey: projectResourceKeys.all(wsId, projectId) });
  qc.invalidateQueries({ queryKey: projectKeys.detail(wsId, projectId) });
  qc.invalidateQueries({ queryKey: projectKeys.list(wsId) });
}

export function useCreateProjectResource(wsId: string, projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateProjectResourceRequest) =>
      api.createProjectResource(projectId, data),
    onSettled: () => {
      invalidateProjectResourceProjection(qc, wsId, projectId);
    },
  });
}

export function useUpdateProjectResource(wsId: string, projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      resourceId,
      data,
    }: {
      resourceId: string;
      data: UpdateProjectResourceRequest;
    }) => api.updateProjectResource(projectId, resourceId, data),
    onSettled: () => {
      invalidateProjectResourceProjection(qc, wsId, projectId);
    },
  });
}

export function useDeleteProjectResource(wsId: string, projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      resourceId,
      expectedRevision,
    }: {
      resourceId: string;
      expectedRevision: number;
    }) => api.deleteProjectResource(projectId, resourceId, expectedRevision),
    onSettled: () => {
      invalidateProjectResourceProjection(qc, wsId, projectId);
    },
  });
}
