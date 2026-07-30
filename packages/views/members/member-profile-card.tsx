"use client";

import { useQuery } from "@tanstack/react-query";
import type { MemberRole } from "@multica/core/types";
import { useWorkspaceId } from "@multica/core";
import { memberListOptions } from "@multica/core/workspace/queries";
import { resolvePublicFileUrl } from "@multica/core/workspace/avatar-url";
import { ActorAvatar } from "@multica/ui/components/common/actor-avatar";
import { Skeleton } from "@multica/ui/components/ui/skeleton";
import { useT } from "../i18n";

export function MemberProfileCard({ userId }: { userId: string }) {
  const { t } = useT("members");
  const wsId = useWorkspaceId();
  const { data: members = [], isLoading } = useQuery(memberListOptions(wsId));
  const member = members.find((candidate) => candidate.user_id === userId);

  if (isLoading && !member) {
    return (
      <div className="flex items-center gap-3">
        <Skeleton className="h-10 w-10 rounded-full" />
        <div className="flex-1 space-y-1.5">
          <Skeleton className="h-4 w-28" />
          <Skeleton className="h-3 w-20" />
        </div>
      </div>
    );
  }

  if (!member) {
    return (
      <div className="text-xs text-muted-foreground">
        {t(($) => $.card.unavailable)}
      </div>
    );
  }

  const initials = member.name
    .split(" ")
    .map((word) => word[0])
    .join("")
    .toUpperCase()
    .slice(0, 2);

  return (
    <div className="flex items-start gap-3 text-left">
      <ActorAvatar
        name={member.name}
        initials={initials}
        avatarUrl={resolvePublicFileUrl(member.avatar_url)}
        size="xl"
      />
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-1.5">
          <p className="truncate text-sm font-semibold">{member.name}</p>
          <RoleBadge role={member.role} />
        </div>
        <p className="mt-0.5 truncate text-xs text-muted-foreground">
          {member.email}
        </p>
      </div>
    </div>
  );
}

function RoleBadge({ role }: { role: MemberRole }) {
  const { t } = useT("members");
  return (
    <span className="rounded-md bg-muted px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground">
      {role === "owner"
        ? t(($) => $.role.owner)
        : role === "admin"
          ? t(($) => $.role.admin)
          : t(($) => $.role.member)}
    </span>
  );
}
