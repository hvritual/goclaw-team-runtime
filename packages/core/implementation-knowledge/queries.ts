import { queryOptions } from "@tanstack/react-query";
import { api } from "../api";

export const implementationKnowledgeKeys = {
	all: (workspaceId: string) => ["implementation-knowledge", workspaceId] as const,
  acceptanceConclusions: (workspaceId: string, issueId: string) =>
    [...implementationKnowledgeKeys.all(workspaceId), "issues", issueId, "acceptance-conclusions"] as const,
  retrospectives: (workspaceId: string, projectId: string) =>
    [...implementationKnowledgeKeys.all(workspaceId), "projects", projectId, "retrospectives"] as const,
};

export function acceptanceConclusionListOptions(workspaceId: string, issueId: string) {
  return queryOptions({
    queryKey: implementationKnowledgeKeys.acceptanceConclusions(workspaceId, issueId),
    queryFn: () => api.listIssueAcceptanceConclusions(issueId),
    enabled: Boolean(workspaceId && issueId),
  });
}

export function projectRetrospectiveListOptions(workspaceId: string, projectId: string) {
  return queryOptions({
    queryKey: implementationKnowledgeKeys.retrospectives(workspaceId, projectId),
    queryFn: () => api.listProjectRetrospectives(projectId),
    enabled: Boolean(workspaceId && projectId),
  });
}
