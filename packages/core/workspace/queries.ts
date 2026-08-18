import { queryOptions } from "@tanstack/react-query";
import { api } from "../api";
import type { Workspace } from "../types";

export const workspaceKeys = {
  all: (wsId: string) => ["workspaces", wsId] as const,
  list: () => ["workspaces", "list"] as const,
  members: (wsId: string) => ["workspaces", wsId, "members"] as const,
  permissions: (wsId: string) => ["workspaces", wsId, "permissions"] as const,
  invitations: (wsId: string) => ["workspaces", wsId, "invitations"] as const,
  myInvitations: () => ["invitations", "mine"] as const,
  skills: (wsId: string) => ["workspaces", wsId, "skills"] as const,
};

export function workspaceListOptions() {
  return queryOptions({
    queryKey: workspaceKeys.list(),
    queryFn: () => api.listWorkspaces(),
  });
}

/** Resolves the workspace whose slug matches, from the cached workspace list. */
export function workspaceBySlugOptions(slug: string) {
  return queryOptions({
    ...workspaceListOptions(),
    select: (list: Workspace[]) => list.find((w) => w.slug === slug) ?? null,
  });
}

export function memberListOptions(wsId: string) {
  return queryOptions({
    queryKey: workspaceKeys.members(wsId),
    queryFn: () => api.listMembers(wsId),
  });
}

export function workspacePermissionOptions(wsId: string, enabled = true) {
  return queryOptions({
    queryKey: workspaceKeys.permissions(wsId),
    queryFn: () => api.getWorkspacePermissions(wsId),
    enabled: !!wsId && enabled,
  });
}

export function skillListOptions(wsId: string) {
  return queryOptions({
    queryKey: workspaceKeys.skills(wsId),
    queryFn: () => api.listSkills(),
  });
}

export function skillDetailOptions(
  wsId: string,
  skillId: string,
  versionId?: string
) {
  return queryOptions({
    queryKey: versionId
      ? ([
          ...workspaceKeys.skills(wsId),
          skillId,
          "versions",
          versionId,
        ] as const)
      : ([...workspaceKeys.skills(wsId), skillId] as const),
    queryFn: () => api.getSkill(skillId, versionId),
    enabled: !!skillId,
  });
}

export function invitationListOptions(wsId: string, enabled = true) {
  return queryOptions({
    queryKey: workspaceKeys.invitations(wsId),
    queryFn: () => api.listWorkspaceInvitations(wsId),
    enabled: !!wsId && enabled,
  });
}

export function myInvitationListOptions() {
  return queryOptions({
    queryKey: workspaceKeys.myInvitations(),
    queryFn: () => api.listMyInvitations(),
  });
}
