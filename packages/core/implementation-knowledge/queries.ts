import { infiniteQueryOptions, queryOptions } from "@tanstack/react-query";
import { api } from "../api";
import type { ProjectRetrospectiveListParams } from "../types/implementation-knowledge";

export const implementationKnowledgeKeys = {
  all: (workspaceId: string) => ["implementation-knowledge", workspaceId] as const,
  acceptanceConclusions: (workspaceId: string, issueId: string) =>
    [...implementationKnowledgeKeys.all(workspaceId), "issues", issueId, "acceptance-conclusions"] as const,
  retrospectiveProject: (workspaceId: string, projectId: string) =>
    [...implementationKnowledgeKeys.all(workspaceId), "projects", projectId, "retrospectives"] as const,
  retrospectiveLists: (workspaceId: string, projectId: string) =>
    [...implementationKnowledgeKeys.retrospectiveProject(workspaceId, projectId), "list"] as const,
  retrospectiveList: (
    workspaceId: string,
    projectId: string,
    params?: ProjectRetrospectiveListParams,
  ) => [...implementationKnowledgeKeys.retrospectiveLists(workspaceId, projectId), params ?? {}] as const,
  retrospectiveDetail: (workspaceId: string, projectId: string, retrospectiveId: string) =>
    [...implementationKnowledgeKeys.retrospectiveProject(workspaceId, projectId), "detail", retrospectiveId] as const,
  /** Backward-compatible prefix for callers that invalidate the whole Project surface. */
  retrospectives: (workspaceId: string, projectId: string) =>
    implementationKnowledgeKeys.retrospectiveProject(workspaceId, projectId),
};

export function acceptanceConclusionListOptions(workspaceId: string, issueId: string) {
  return queryOptions({
    queryKey: implementationKnowledgeKeys.acceptanceConclusions(workspaceId, issueId),
    queryFn: () => api.listIssueAcceptanceConclusions(issueId),
    enabled: Boolean(workspaceId && issueId),
  });
}

export function projectRetrospectiveListOptions(
  workspaceId: string,
  projectId: string,
  params?: ProjectRetrospectiveListParams,
) {
  return queryOptions({
    queryKey: implementationKnowledgeKeys.retrospectiveList(workspaceId, projectId, params),
    queryFn: () => api.listProjectRetrospectives(projectId, params),
    enabled: Boolean(workspaceId && projectId),
    retry: false,
  });
}

export function projectRetrospectiveDetailOptions(
  workspaceId: string,
  projectId: string,
  retrospectiveId: string,
) {
  return queryOptions({
    queryKey: implementationKnowledgeKeys.retrospectiveDetail(workspaceId, projectId, retrospectiveId),
    queryFn: () => api.getProjectRetrospective(projectId, retrospectiveId),
    enabled: Boolean(workspaceId && projectId && retrospectiveId),
    retry: false,
  });
}

export function projectRetrospectiveInfiniteListOptions(
  workspaceId: string,
  projectId: string,
  includeArchived = false,
) {
  return infiniteQueryOptions({
    queryKey: implementationKnowledgeKeys.retrospectiveList(workspaceId, projectId, {
      includeArchived,
    }),
    initialPageParam: null as string | null,
    queryFn: ({ pageParam }) => api.listProjectRetrospectives(projectId, {
      includeArchived,
      cursor: pageParam ?? undefined,
    }),
    getNextPageParam: (lastPage) => lastPage.nextCursor,
    enabled: Boolean(workspaceId && projectId),
    retry: false,
    select: (result) => ({
      ...result,
      retrospectives: result.pages.flatMap((page) => page.retrospectives),
    }),
  });
}
