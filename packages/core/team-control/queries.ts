import { queryOptions } from "@tanstack/react-query";
import {
  getTeamControlProjection,
  getTeamControlWorkspace,
  listTeamControlMembers,
} from "./client";

export const teamControlKeys = {
  all: (workspaceId: string) => ["team-control", workspaceId] as const,
  workspace: (workspaceId: string) => [
    ...teamControlKeys.all(workspaceId),
    "workspace",
  ] as const,
  members: (workspaceId: string) => [
    ...teamControlKeys.all(workspaceId),
    "members",
  ] as const,
  project: (workspaceId: string, projectId: string) => [
    ...teamControlKeys.all(workspaceId),
    "project",
    projectId,
  ] as const,
  projection: (workspaceId: string, projectId: string) => [
    ...teamControlKeys.project(workspaceId, projectId),
    "projection",
  ] as const,
};

export function teamControlWorkspaceOptions(workspaceId: string) {
  return queryOptions({
    queryKey: teamControlKeys.workspace(workspaceId),
    queryFn: () => getTeamControlWorkspace(workspaceId),
  });
}

export function teamControlMembersOptions(workspaceId: string) {
  return queryOptions({
    queryKey: teamControlKeys.members(workspaceId),
    queryFn: () => listTeamControlMembers(workspaceId),
  });
}

export function teamControlProjectionOptions(
  workspaceId: string,
  projectId: string,
) {
  return queryOptions({
    queryKey: teamControlKeys.projection(workspaceId, projectId),
    queryFn: () => getTeamControlProjection(workspaceId, projectId),
  });
}
