"use client";

import { ActorAvatar as ActorAvatarBase } from "@multica/ui/components/common/actor-avatar";
import type { AvatarSize } from "@multica/ui/lib/avatar-size";
import { useActorName } from "@multica/core/workspace/hooks";
import { useWorkspacePaths } from "@multica/core/paths";
import { MemberProfileCard } from "../members/member-profile-card";
import { useNavigation } from "../navigation";
import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from "@multica/ui/components/ui/hover-card";

interface ActorAvatarProps {
  actorType: string;
  actorId: string;
  size?: AvatarSize;
  className?: string;
  enableHoverCard?: boolean;
  profileLink?: boolean;
}

export function ActorAvatar({
  actorType,
  actorId,
  size,
  className,
  enableHoverCard,
  profileLink,
}: ActorAvatarProps) {
  const { getActorName, getActorInitials, getActorAvatarUrl } = useActorName();
  const paths = useWorkspacePaths();

  const avatar = (
    <ActorAvatarBase
      name={getActorName(actorType, actorId)}
      initials={getActorInitials(actorType, actorId)}
      avatarUrl={getActorAvatarUrl(actorType, actorId)}
      isSystem={actorType === "system"}
      size={size}
      className={className}
    />
  );

  if (actorType !== "member") return avatar;

  const linked =
    profileLink === false ? (
      avatar
    ) : (
      <MemberProfileLink href={paths.memberDetail(actorId)}>
        {avatar}
      </MemberProfileLink>
    );

  if (!enableHoverCard) return linked;

  return (
    <HoverCard>
      <HoverCardTrigger render={<span className="inline-flex cursor-pointer" />}>
        {linked}
      </HoverCardTrigger>
      <HoverCardContent align="start" className="w-72">
        <MemberProfileCard userId={actorId} />
      </HoverCardContent>
    </HoverCard>
  );
}

function MemberProfileLink({
  href,
  children,
}: {
  href: string;
  children: React.ReactNode;
}) {
  const { push, openInNewTab, getShareableUrl } = useNavigation();

  const navigate = (event: React.MouseEvent | React.KeyboardEvent) => {
    event.preventDefault();
    event.stopPropagation();
    if ("metaKey" in event && (event.metaKey || event.ctrlKey || event.shiftKey)) {
      if (openInNewTab) {
        openInNewTab(href);
      } else {
        window.open(getShareableUrl(href), "_blank", "noopener,noreferrer");
      }
      return;
    }
    push(href);
  };

  return (
    <span
      role="link"
      tabIndex={-1}
      className="inline-flex cursor-pointer rounded-full"
      onClick={navigate}
      onKeyDown={(event) => {
        if (event.key === "Enter" || event.key === " ") navigate(event);
      }}
    >
      {children}
    </span>
  );
}
