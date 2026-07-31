import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api";
import type {
  ProposeKnowledgeRequest,
  ReviewKnowledgeRequest,
} from "../types";
import { knowledgeKeys } from "./queries";
import { useWorkspaceId } from "../hooks";

export function useProposeCommentDecision() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (commentId: string) => api.proposeCommentDecision(commentId),
    onSettled: () =>
      queryClient.invalidateQueries({ queryKey: knowledgeKeys.all(workspaceId) }),
  });
}

export function useProposeKnowledge(workspaceId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (request: ProposeKnowledgeRequest) =>
      api.proposeKnowledge(request),
    onSettled: () =>
      queryClient.invalidateQueries({ queryKey: knowledgeKeys.all(workspaceId) }),
  });
}

export function useReviewKnowledge(workspaceId: string) {
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
