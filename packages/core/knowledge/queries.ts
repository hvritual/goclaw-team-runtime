import { queryOptions } from "@tanstack/react-query";
import { api } from "../api";
import type { KnowledgeQueryFilters } from "../types";

export const knowledgeKeys = {
  all: (workspaceId: string) => ["knowledge", workspaceId] as const,
  list: (workspaceId: string, filters: KnowledgeQueryFilters) =>
    [...knowledgeKeys.all(workspaceId), "list", filters] as const,
  candidates: (workspaceId: string) =>
    [...knowledgeKeys.all(workspaceId), "candidates"] as const,
  detail: (workspaceId: string, knowledgeId: string) =>
    [...knowledgeKeys.all(workspaceId), "detail", knowledgeId] as const,
};

export function knowledgeListOptions(workspaceId: string, filters: KnowledgeQueryFilters = {}, enabled = true) {
  return queryOptions({
    queryKey: knowledgeKeys.list(workspaceId, filters),
    queryFn: () => api.listKnowledge(filters),
    enabled: Boolean(workspaceId) && enabled,
  });
}

export function knowledgeDetailOptions(
  workspaceId: string,
  knowledgeId: string,
  enabled = true,
) {
  return queryOptions({
    queryKey: knowledgeKeys.detail(workspaceId, knowledgeId),
    queryFn: () => api.getKnowledge(knowledgeId),
    enabled: Boolean(workspaceId) && Boolean(knowledgeId) && enabled,
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
