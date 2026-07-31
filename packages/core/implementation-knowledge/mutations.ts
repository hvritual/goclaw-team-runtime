import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api";
import { useWorkspaceId } from "../hooks";
import { issueKeys } from "../issues/queries";
import { projectKeys } from "../projects/queries";
import { knowledgeKeys } from "../knowledge/queries";
import type { AcceptanceConclusionInput, ProjectRetrospectiveInput } from "../types";
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

export function useCreateProjectRetrospective(projectId: string) {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: ProjectRetrospectiveInput) =>
      api.createProjectRetrospective(projectId, input),
    onSettled: () => {
      queryClient.invalidateQueries({
        queryKey: implementationKnowledgeKeys.retrospectives(workspaceId, projectId),
      });
      queryClient.invalidateQueries({ queryKey: projectKeys.detail(workspaceId, projectId) });
      queryClient.invalidateQueries({ queryKey: knowledgeKeys.all(workspaceId) });
    },
  });
}
