import { useMutation, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { useRef } from "react";
import { api } from "../api";
import { useWorkspaceId } from "../hooks";
import { issueKeys } from "../issues/queries";
import { knowledgeKeys } from "../knowledge/queries";
import { projectKeys } from "../projects/queries";
import { taskKeys } from "../tasks/queries";
import type {
  AcceptanceConclusionInput,
  ProjectRetrospectiveInput,
  ProjectRetrospectiveTargetKind,
  ProjectRetrospectiveUpdateInput,
} from "../types/implementation-knowledge";
import { createSafeId } from "../utils";
import { implementationKnowledgeKeys } from "./queries";

export function useCreateAcceptanceConclusion(issueId: string) {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: AcceptanceConclusionInput) =>
      api.createIssueAcceptanceConclusion(issueId, input),
    onSettled: () => {
      queryClient.invalidateQueries({
        queryKey: implementationKnowledgeKeys.acceptanceConclusions(workspaceId, issueId),
      });
      queryClient.invalidateQueries({ queryKey: issueKeys.detail(workspaceId, issueId) });
      queryClient.invalidateQueries({ queryKey: knowledgeKeys.all(workspaceId) });
    },
  });
}

export function useCompleteIssueWithAcceptance(issueId: string) {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: AcceptanceConclusionInput | null) =>
      api.updateIssue(issueId, {
        status: "done",
        ...(input ? { acceptanceConclusion: input } : {}),
      }),
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: issueKeys.all(workspaceId) });
      queryClient.invalidateQueries({
        queryKey: implementationKnowledgeKeys.acceptanceConclusions(workspaceId, issueId),
      });
      queryClient.invalidateQueries({ queryKey: knowledgeKeys.all(workspaceId) });
    },
  });
}

function createSignature(projectId: string, input: ProjectRetrospectiveInput) {
  return JSON.stringify([projectId, input.content, input.participants]);
}

function targetSignature(
  projectId: string,
  input: {
    retrospectiveId: string;
    actionItemId: string;
    targetKind?: ProjectRetrospectiveTargetKind;
  },
) {
  return JSON.stringify([
    projectId,
    input.retrospectiveId,
    input.actionItemId,
    input.targetKind ?? "task",
  ]);
}

async function invalidateRetrospective(
  queryClient: QueryClient,
  workspaceId: string,
  projectId: string,
  retrospectiveId?: string,
) {
  const invalidations = [
    queryClient.invalidateQueries({
      queryKey: implementationKnowledgeKeys.retrospectiveLists(workspaceId, projectId),
    }),
    queryClient.invalidateQueries({ queryKey: projectKeys.detail(workspaceId, projectId) }),
  ];
  if (retrospectiveId) {
    invalidations.push(queryClient.invalidateQueries({
      queryKey: implementationKnowledgeKeys.retrospectiveDetail(
        workspaceId,
        projectId,
        retrospectiveId,
      ),
    }));
  }
  await Promise.all(invalidations);
}

export function useCreateProjectRetrospective(projectId: string) {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  const retryKeys = useRef(new Map<string, string>());
  return useMutation({
    mutationFn: (input: ProjectRetrospectiveInput) => {
      const signature = createSignature(projectId, input);
      const idempotencyKey =
        input.idempotencyKey?.trim() ||
        retryKeys.current.get(signature) ||
        createSafeId();
      retryKeys.current.set(signature, idempotencyKey);
      return api.createProjectRetrospective(projectId, { ...input, idempotencyKey });
    },
    onSuccess: async (data, variables) => {
      retryKeys.current.delete(createSignature(projectId, variables));
      await invalidateRetrospective(queryClient, workspaceId, projectId, data.id);
    },
  });
}

export function useUpdateProjectRetrospective(projectId: string) {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      retrospectiveId,
      ...input
    }: { retrospectiveId: string } & ProjectRetrospectiveUpdateInput) =>
      api.updateProjectRetrospective(projectId, retrospectiveId, input),
    onSuccess: (_data, variables) =>
      invalidateRetrospective(
        queryClient,
        workspaceId,
        projectId,
        variables.retrospectiveId,
      ),
  });
}

export function useArchiveProjectRetrospective(projectId: string) {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      retrospectiveId,
      expectedRevision,
    }: {
      retrospectiveId: string;
      expectedRevision: number;
    }) => api.archiveProjectRetrospective(projectId, retrospectiveId, expectedRevision),
    onSuccess: (_data, variables) =>
      invalidateRetrospective(
        queryClient,
        workspaceId,
        projectId,
        variables.retrospectiveId,
      ),
  });
}

export function useCreateProjectRetrospectiveTarget(projectId: string) {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  const retryKeys = useRef(new Map<string, string>());
  return useMutation({
    mutationFn: (input: {
      retrospectiveId: string;
      actionItemId: string;
      targetKind?: ProjectRetrospectiveTargetKind;
      idempotencyKey?: string;
    }) => {
      const signature = targetSignature(projectId, input);
      const idempotencyKey =
        input.idempotencyKey?.trim() ||
        retryKeys.current.get(signature) ||
        createSafeId();
      retryKeys.current.set(signature, idempotencyKey);
      return api.createProjectRetrospectiveTarget(
        projectId,
        input.retrospectiveId,
        input.actionItemId,
        { targetKind: input.targetKind, idempotencyKey },
      );
    },
    onSuccess: async (_data, variables) => {
      retryKeys.current.delete(targetSignature(projectId, variables));
      await Promise.all([
        invalidateRetrospective(
          queryClient,
          workspaceId,
          projectId,
          variables.retrospectiveId,
        ),
        queryClient.invalidateQueries({ queryKey: taskKeys.all(workspaceId) }),
        queryClient.invalidateQueries({ queryKey: issueKeys.all(workspaceId) }),
      ]);
    },
  });
}
