import { queryOptions, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api";
import type { ProjectRequirementTransitionRequest, SaveProjectRequirementDraftRequest } from "../types";

export const projectRequirementKeys = {
  detail: (wsId: string, projectId: string) => ["project-requirements", wsId, projectId] as const,
};

export function projectRequirementBaselineOptions(wsId: string, projectId: string) {
  return queryOptions({ queryKey: projectRequirementKeys.detail(wsId, projectId), queryFn: () => api.getProjectRequirementBaseline(projectId) });
}

function useInvalidate(wsId: string, projectId: string) {
  const queryClient = useQueryClient();
  return () => queryClient.invalidateQueries({ queryKey: projectRequirementKeys.detail(wsId, projectId) });
}

export function useSaveProjectRequirementDraft(wsId: string, projectId: string) {
  const invalidate = useInvalidate(wsId, projectId);
  return useMutation({ mutationFn: (input: SaveProjectRequirementDraftRequest) => api.saveProjectRequirementDraft(projectId, input), onSettled: invalidate });
}

export function useSubmitProjectRequirementReview(wsId: string, projectId: string) {
  const invalidate = useInvalidate(wsId, projectId);
  return useMutation({ mutationFn: (input: ProjectRequirementTransitionRequest) => api.submitProjectRequirementReview(projectId, input), onSettled: invalidate });
}

export function useApproveProjectRequirement(wsId: string, projectId: string) {
  const invalidate = useInvalidate(wsId, projectId);
  return useMutation({ mutationFn: (input: ProjectRequirementTransitionRequest) => api.approveProjectRequirement(projectId, input), onSettled: invalidate });
}

export function useWithdrawProjectRequirementReview(wsId: string, projectId: string) {
  const invalidate = useInvalidate(wsId, projectId);
  return useMutation({ mutationFn: (input: ProjectRequirementTransitionRequest) => api.withdrawProjectRequirementReview(projectId, input), onSettled: invalidate });
}
