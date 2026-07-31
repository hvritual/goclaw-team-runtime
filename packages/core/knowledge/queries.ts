import { queryOptions } from "@tanstack/react-query";
import { api } from "../api";

export const knowledgeKeys = {
  all: (workspaceId: string) => ["knowledge", workspaceId] as const,
  list: (workspaceId: string, query: string) =>
    [...knowledgeKeys.all(workspaceId), "list", query] as const,
  candidates: (workspaceId: string) =>
    [...knowledgeKeys.all(workspaceId), "candidates"] as const,
};

export function knowledgeListOptions(workspaceId: string, query = "") {
  return queryOptions({
    queryKey: knowledgeKeys.list(workspaceId, query),
    queryFn: () => api.listKnowledge({ query }),
    enabled: Boolean(workspaceId),
  });
}

export function knowledgeCandidateListOptions(
  workspaceId: string,
  enabled: boolean,
) {
  return queryOptions({
    queryKey: knowledgeKeys.candidates(workspaceId),
    queryFn: () => api.listKnowledgeCandidates(),
    enabled: Boolean(workspaceId) && enabled,
  });
}
