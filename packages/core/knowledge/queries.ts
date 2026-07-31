import { queryOptions } from "@tanstack/react-query";
import { api } from "../api";

export const knowledgeKeys = {
  all: (workspaceId: string) => ["knowledge", workspaceId] as const,
  list: (workspaceId: string, query: string) =>
    [...knowledgeKeys.all(workspaceId), "list", query] as const,
  candidates: (workspaceId: string) =>
    [...knowledgeKeys.all(workspaceId), "candidates"] as const,
  detail: (workspaceId: string, knowledgeId: string) =>
    [...knowledgeKeys.all(workspaceId), "detail", knowledgeId] as const,
};

export function knowledgeListOptions(workspaceId: string, query = "") {
  return queryOptions({
    queryKey: knowledgeKeys.list(workspaceId, query),
    queryFn: () => api.listKnowledge({ query }),
    enabled: Boolean(workspaceId),
  });
}

export function knowledgeDetailOptions(
  workspaceId: string,
  knowledgeId: string,
) {
  return queryOptions({
    queryKey: knowledgeKeys.detail(workspaceId, knowledgeId),
    queryFn: () => api.getKnowledge(knowledgeId),
    enabled: Boolean(workspaceId) && Boolean(knowledgeId),
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
