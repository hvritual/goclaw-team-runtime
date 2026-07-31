import { queryOptions } from "@tanstack/react-query";
import { api } from "../api";

export const projectKeys = {
  all: (wsId: string) => ["projects", wsId] as const,
  list: (wsId: string) => [...projectKeys.all(wsId), "list"] as const,
  detail: (wsId: string, id: string) =>
    [...projectKeys.all(wsId), "detail", id] as const,
};

export function projectListOptions(wsId: string) {
  return queryOptions({
    queryKey: projectKeys.list(wsId),
    queryFn: () => api.listProjects(),
    select: (data) => data.projects,
  });
}

export interface ProjectLeadership {
  projectId: string;
  leadType: "member" | null;
  leadId: string | null;
}

/**
 * Keep wire-format project fields at the query boundary for consumers that
 * only need leadership data.
 */
export function projectLeadershipListOptions(wsId: string) {
  return queryOptions({
    queryKey: projectKeys.list(wsId),
    queryFn: () => api.listProjects(),
    select: (data): ProjectLeadership[] =>
      data.projects.map((project) => ({
        projectId: project.id,
        leadType: project.lead_type,
        leadId: project.lead_id,
      })),
  });
}

export function projectDetailOptions(wsId: string, id: string) {
  return queryOptions({
    queryKey: projectKeys.detail(wsId, id),
    queryFn: () => api.getProject(id),
  });
}
