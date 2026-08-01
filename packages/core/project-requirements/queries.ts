import { queryOptions, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api";
import { issueKeys } from "../issues/queries";
import type { ProjectRequirementCreateIssueRequest, ProjectRequirementLinkRequest, ProjectRequirementTransitionRequest, SaveProjectRequirementDraftRequest } from "../types";

export const projectRequirementKeys = {
  detail: (wsId: string, projectId: string) => ["project-requirements", wsId, projectId] as const,
  coverage: (wsId: string, projectId: string) => ["project-requirements", wsId, projectId, "coverage"] as const,
  issues: (wsId: string, projectId: string) => ["project-requirements", wsId, projectId, "issues"] as const,
};

export function projectRequirementBaselineOptions(wsId: string, projectId: string) {
  return queryOptions({ queryKey: projectRequirementKeys.detail(wsId, projectId), queryFn: () => api.getProjectRequirementBaseline(projectId) });
}

export function projectRequirementCoverageOptions(wsId: string, projectId: string) {
  return queryOptions({
    queryKey: projectRequirementKeys.coverage(wsId, projectId),
    queryFn: () => api.getProjectRequirementCoverage(projectId),
  });
}

export function projectRequirementIssuesOptions(wsId: string, projectId: string) {
  return queryOptions({
    queryKey: projectRequirementKeys.issues(wsId, projectId),
    queryFn: () => api.listIssues({ project_id: projectId }),
  });
}

function useInvalidateBaseline(wsId: string, projectId: string) {
  const queryClient = useQueryClient();
  return () => Promise.all([
    queryClient.invalidateQueries({ queryKey: projectRequirementKeys.detail(wsId, projectId) }),
    queryClient.invalidateQueries({ queryKey: projectRequirementKeys.coverage(wsId, projectId) }),
  ]);
}
function useInvalidateCoverage(wsId: string, projectId: string) {
  const queryClient = useQueryClient();
  return () => queryClient.invalidateQueries({ queryKey: projectRequirementKeys.coverage(wsId, projectId) });
}
function useInvalidateCreatedIssue(wsId: string, projectId: string) {
  const queryClient = useQueryClient();
  return () => Promise.all([
    queryClient.invalidateQueries({ queryKey: projectRequirementKeys.coverage(wsId, projectId) }),
    queryClient.invalidateQueries({ queryKey: projectRequirementKeys.issues(wsId, projectId) }),
    queryClient.invalidateQueries({ queryKey: issueKeys.all(wsId) }),
  ]);
}

export function useLinkProjectRequirementIssue(wsId: string, projectId: string) { const invalidate = useInvalidateCoverage(wsId, projectId); return useMutation({ mutationFn: (input: ProjectRequirementLinkRequest) => api.linkProjectRequirementIssue(projectId, input), onSettled: invalidate }); }
export function useUnlinkProjectRequirementIssue(wsId: string, projectId: string) { const invalidate = useInvalidateCoverage(wsId, projectId); return useMutation({ mutationFn: (input: ProjectRequirementLinkRequest) => api.unlinkProjectRequirementIssue(projectId, input), onSettled: invalidate }); }
export function useCreateIssueForProjectRequirement(wsId: string, projectId: string) { const invalidate = useInvalidateCreatedIssue(wsId, projectId); return useMutation({ mutationFn: ({ requirementKey, input }: { requirementKey: string; input: ProjectRequirementCreateIssueRequest }) => api.createIssueForProjectRequirement(projectId, requirementKey, input), onSettled: invalidate }); }

export function useSaveProjectRequirementDraft(wsId: string, projectId: string) {
  const invalidate = useInvalidateBaseline(wsId, projectId);
  return useMutation({ mutationFn: (input: SaveProjectRequirementDraftRequest) => api.saveProjectRequirementDraft(projectId, input), onSettled: invalidate });
}

export function useSubmitProjectRequirementReview(wsId: string, projectId: string) {
  const invalidate = useInvalidateBaseline(wsId, projectId);
  return useMutation({ mutationFn: (input: ProjectRequirementTransitionRequest) => api.submitProjectRequirementReview(projectId, input), onSettled: invalidate });
}

export function useApproveProjectRequirement(wsId: string, projectId: string) {
  const invalidate = useInvalidateBaseline(wsId, projectId);
  return useMutation({ mutationFn: (input: ProjectRequirementTransitionRequest) => api.approveProjectRequirement(projectId, input), onSettled: invalidate });
}

export function useWithdrawProjectRequirementReview(wsId: string, projectId: string) {
  const invalidate = useInvalidateBaseline(wsId, projectId);
  return useMutation({ mutationFn: (input: ProjectRequirementTransitionRequest) => api.withdrawProjectRequirementReview(projectId, input), onSettled: invalidate });
}
