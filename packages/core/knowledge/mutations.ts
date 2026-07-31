import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api";
import { useWorkspaceId } from "../hooks";
import type {
  ProposeKnowledgeRequest,
  ReviewKnowledgeRequest,
} from "../types";
import { knowledgeKeys } from "./queries";

export function useProposeKnowledge() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (request: ProposeKnowledgeRequest) =>
      api.proposeKnowledge(request),
    onSettled: () =>
      queryClient.invalidateQueries({ queryKey: knowledgeKeys.all(workspaceId) }),
  });
}

export function useReviewKnowledge() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      candidateId,
      ...request
    }: ReviewKnowledgeRequest & { candidateId: string }) =>
      api.reviewKnowledgeCandidate(candidateId, request),
    onSettled: () =>
      queryClient.invalidateQueries({ queryKey: knowledgeKeys.all(workspaceId) }),
  });
}
