"use client";

import { useCallback, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import type { MemberWithUser } from "../types";
import { useWorkspaceId } from "../hooks";
import { memberListOptions } from "./queries";
import { resolvePublicFileUrl } from "./avatar-url";

const EMPTY_MEMBERS: MemberWithUser[] = [];

export function buildActorNameResolver(directories: {
  members: readonly { user_id: string; name: string }[];
}) {
  const memberNames = new Map(
    directories.members.map((member) => [member.user_id, member.name]),
  );
  return (type: string, id: string) => {
    if (type === "member") return memberNames.get(id) ?? "Unknown";
    if (type === "system") return "Multica";
    return "System";
  };
}

export function useActorName() {
  const workspaceId = useWorkspaceId();
  const { data: members = EMPTY_MEMBERS } = useQuery(
    memberListOptions(workspaceId),
  );
  const getMemberName = useCallback(
    (userId: string) =>
      members.find((member) => member.user_id === userId)?.name ?? "Unknown",
    [members],
  );
  const getActorName = useMemo(
    () => buildActorNameResolver({ members }),
    [members],
  );
  const getActorInitials = useCallback(
    (type: string, id: string) =>
      getActorName(type, id)
        .split(" ")
        .map((part) => part[0])
        .join("")
        .toUpperCase()
        .slice(0, 2),
    [getActorName],
  );
  const getActorAvatarUrl = useCallback(
    (type: string, id: string): string | null =>
      type === "member"
        ? resolvePublicFileUrl(
            members.find((member) => member.user_id === id)?.avatar_url,
          )
        : null,
    [members],
  );

  return useMemo(
    () => ({
      getMemberName,
      getActorName,
      getActorInitials,
      getActorAvatarUrl,
    }),
    [getActorAvatarUrl, getActorInitials, getActorName, getMemberName],
  );
}
