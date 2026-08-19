import {
  queryOptions,
  useMutation,
  useQueryClient,
} from "@tanstack/react-query";
import { useRef } from "react";
import { api } from "../api";
import { createSafeId } from "../utils";
import type {
  CreateProjectOutlineNodeRequest,
  ProjectRequirementLinkRequest,
  ProjectRequirementOutlineLinkRequest,
  ProjectRequirementTransitionRequest,
  ReplaceProjectRequirementAccessRequest,
  SaveProjectRequirementDraftRequest,
} from "../types";

export const projectRequirementKeys = {
  detail: (wsId: string, projectId: string) =>
    ["project-requirements", wsId, projectId] as const,
  coverageAll: (wsId: string) =>
    ["project-requirement-coverage", wsId] as const,
  coverage: (wsId: string, projectId: string) =>
    [...projectRequirementKeys.coverageAll(wsId), projectId] as const,
  issues: (wsId: string, projectId: string) =>
    ["project-requirements", wsId, projectId, "issues"] as const,
  access: (wsId: string, projectId: string) =>
    ["project-requirements", wsId, projectId, "access"] as const,
  outline: (wsId: string, projectId: string) =>
    ["project-requirements", wsId, projectId, "outline"] as const,
};

export function projectRequirementBaselineOptions(
  wsId: string,
  projectId: string
) {
  return queryOptions({
    queryKey: projectRequirementKeys.detail(wsId, projectId),
    queryFn: () => api.getProjectRequirementBaseline(projectId),
  });
}

// S07C owns coverage semantics. The compatibility query remains uninstalled in
// the S07B view and must not be interpreted as acceptance coverage.
export function projectRequirementCoverageOptions(
  wsId: string,
  projectId: string
) {
  return queryOptions({
    queryKey: projectRequirementKeys.coverage(wsId, projectId),
    queryFn: () => api.getProjectRequirementCoverage(projectId),
  });
}

export function projectRequirementIssuesOptions(
  wsId: string,
  projectId: string
) {
  return queryOptions({
    queryKey: projectRequirementKeys.issues(wsId, projectId),
    queryFn: () => api.listIssues({ project_id: projectId }),
  });
}

export function projectRequirementAccessOptions(
  wsId: string,
  projectId: string,
  enabled = true
) {
  return queryOptions({
    queryKey: projectRequirementKeys.access(wsId, projectId),
    queryFn: () => api.getProjectRequirementAccess(projectId),
    enabled,
  });
}

export function projectOutlineOptions(wsId: string, projectId: string) {
  return queryOptions({
    queryKey: projectRequirementKeys.outline(wsId, projectId),
    queryFn: () => api.getProjectOutline(projectId),
  });
}

function useInvalidateBaseline(wsId: string, projectId: string) {
  const queryClient = useQueryClient();
  return () =>
    Promise.all([
      queryClient.invalidateQueries({
        queryKey: projectRequirementKeys.detail(wsId, projectId),
      }),
      queryClient.invalidateQueries({
        queryKey: projectRequirementKeys.coverage(wsId, projectId),
      }),
    ]);
}

function saveSignature(
  projectId: string,
  input: SaveProjectRequirementDraftRequest
): string {
  return JSON.stringify([
    projectId,
    input.expectedRevision,
    input.content,
    input.changeSummary,
    input.materialChange,
  ]);
}

function outlineCreateSignature(
  projectId: string,
  input: CreateProjectOutlineNodeRequest
): string {
  return JSON.stringify([projectId, input.expectedRevision, input.title]);
}

export function useSaveProjectRequirementDraft(
  wsId: string,
  projectId: string
) {
  const invalidate = useInvalidateBaseline(wsId, projectId);
  const retryKeys = useRef(new Map<string, string>());
  return useMutation({
    mutationFn: (input: SaveProjectRequirementDraftRequest) => {
      if (input.expectedRevision !== 0) {
        return api.saveProjectRequirementDraft(projectId, input);
      }
      const signature = saveSignature(projectId, input);
      const idempotencyKey =
        input.idempotencyKey?.trim() ||
        retryKeys.current.get(signature) ||
        createSafeId();
      retryKeys.current.set(signature, idempotencyKey);
      return api.saveProjectRequirementDraft(projectId, {
        ...input,
        idempotencyKey,
      });
    },
    onSuccess: (_data, variables) => {
      if (variables.expectedRevision === 0) {
        retryKeys.current.delete(saveSignature(projectId, variables));
      }
    },
    onSettled: invalidate,
  });
}

export function useSubmitProjectRequirementReview(
  wsId: string,
  projectId: string
) {
  const invalidate = useInvalidateBaseline(wsId, projectId);
  return useMutation({
    mutationFn: (input: ProjectRequirementTransitionRequest) =>
      api.submitProjectRequirementReview(projectId, input),
    onSettled: invalidate,
  });
}

export function useApproveProjectRequirement(wsId: string, projectId: string) {
  const invalidate = useInvalidateBaseline(wsId, projectId);
  return useMutation({
    mutationFn: (input: ProjectRequirementTransitionRequest) =>
      api.approveProjectRequirement(projectId, input),
    onSettled: invalidate,
  });
}

export function useWithdrawProjectRequirementReview(
  wsId: string,
  projectId: string
) {
  const invalidate = useInvalidateBaseline(wsId, projectId);
  return useMutation({
    mutationFn: (input: ProjectRequirementTransitionRequest) =>
      api.withdrawProjectRequirementReview(projectId, input),
    onSettled: invalidate,
  });
}

export function useFreezeProjectRequirement(wsId: string, projectId: string) {
  const invalidate = useInvalidateBaseline(wsId, projectId);
  return useMutation({
    mutationFn: (input: ProjectRequirementTransitionRequest) =>
      api.freezeProjectRequirement(projectId, input),
    onSettled: invalidate,
  });
}

export function useRetireProjectRequirement(wsId: string, projectId: string) {
  const invalidate = useInvalidateBaseline(wsId, projectId);
  return useMutation({
    mutationFn: (input: ProjectRequirementTransitionRequest) =>
      api.retireProjectRequirement(projectId, input),
    onSettled: invalidate,
  });
}

export function useLinkProjectRequirementIssue(
  wsId: string,
  projectId: string
) {
  const invalidate = useInvalidateBaseline(wsId, projectId);
  return useMutation({
    mutationFn: (input: ProjectRequirementLinkRequest) =>
      api.linkProjectRequirementIssue(projectId, input),
    onSettled: invalidate,
  });
}

export function useUnlinkProjectRequirementIssue(
  wsId: string,
  projectId: string
) {
  const invalidate = useInvalidateBaseline(wsId, projectId);
  return useMutation({
    mutationFn: (input: ProjectRequirementLinkRequest) =>
      api.unlinkProjectRequirementIssue(projectId, input),
    onSettled: invalidate,
  });
}

export function useLinkProjectRequirementOutline(
  wsId: string,
  projectId: string
) {
  const invalidate = useInvalidateBaseline(wsId, projectId);
  return useMutation({
    mutationFn: (input: ProjectRequirementOutlineLinkRequest) =>
      api.linkProjectRequirementOutline(projectId, input),
    onSettled: invalidate,
  });
}

export function useUnlinkProjectRequirementOutline(
  wsId: string,
  projectId: string
) {
  const invalidate = useInvalidateBaseline(wsId, projectId);
  return useMutation({
    mutationFn: (input: ProjectRequirementOutlineLinkRequest) =>
      api.unlinkProjectRequirementOutline(projectId, input),
    onSettled: invalidate,
  });
}

export function useCreateProjectOutlineNode(wsId: string, projectId: string) {
  const queryClient = useQueryClient();
  const retryKeys = useRef(new Map<string, string>());
  return useMutation({
    mutationFn: (input: CreateProjectOutlineNodeRequest) => {
      const signature = outlineCreateSignature(projectId, input);
      const idempotencyKey =
        input.idempotencyKey?.trim() ||
        retryKeys.current.get(signature) ||
        createSafeId();
      retryKeys.current.set(signature, idempotencyKey);
      return api.createProjectOutlineNode(projectId, {
        ...input,
        idempotencyKey,
      });
    },
    onSuccess: (_data, variables) => {
      retryKeys.current.delete(outlineCreateSignature(projectId, variables));
    },
    onSettled: () =>
      queryClient.invalidateQueries({
        queryKey: projectRequirementKeys.outline(wsId, projectId),
      }),
  });
}

export function useReplaceProjectRequirementAccess(
  wsId: string,
  projectId: string
) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: ReplaceProjectRequirementAccessRequest) =>
      api.replaceProjectRequirementAccess(projectId, input),
    onSettled: () =>
      Promise.all([
        queryClient.invalidateQueries({
          queryKey: projectRequirementKeys.access(wsId, projectId),
        }),
        queryClient.invalidateQueries({
          queryKey: projectRequirementKeys.detail(wsId, projectId),
        }),
      ]),
  });
}
