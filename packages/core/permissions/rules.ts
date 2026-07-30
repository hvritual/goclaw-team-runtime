import type { Comment, MemberRole, Skill } from "../types";
import { ALLOW, deny, type Decision, type PermissionContext } from "./types";

const isAdminLike = (role: MemberRole | null) =>
  role === "owner" || role === "admin";

export function canEditSkill(skill: Skill, ctx: PermissionContext): Decision {
  if (ctx.userId === null) {
    return deny("not_authenticated", "Sign in to edit this skill.");
  }
  if (isAdminLike(ctx.role) || skill.created_by === ctx.userId) return ALLOW;
  return deny(
    "not_resource_owner",
    "Only the creator and workspace admins can edit this skill.",
  );
}

export function canDeleteSkill(skill: Skill, ctx: PermissionContext): Decision {
  return canEditSkill(skill, ctx);
}

export function canEditComment(
  comment: Comment,
  ctx: PermissionContext,
): Decision {
  if (ctx.userId === null) {
    return deny("not_authenticated", "Sign in to edit comments.");
  }
  if (comment.author_type !== "member") {
    return deny("not_resource_owner", "System comments cannot be edited.");
  }
  if (comment.author_id === ctx.userId || isAdminLike(ctx.role)) return ALLOW;
  return deny(
    "not_resource_owner",
    "Only the author and workspace admins can edit this comment.",
  );
}

export function canDeleteComment(
  comment: Comment,
  ctx: PermissionContext,
): Decision {
  if (ctx.userId === null) {
    return deny("not_authenticated", "Sign in to delete comments.");
  }
  if (
    (comment.author_type === "member" && comment.author_id === ctx.userId) ||
    isAdminLike(ctx.role)
  ) {
    return ALLOW;
  }
  return deny(
    "not_resource_owner",
    "Only the author and workspace admins can delete this comment.",
  );
}

export function canManageMembers(ctx: PermissionContext): Decision {
  if (isAdminLike(ctx.role)) return ALLOW;
  return deny(
    "not_admin_role",
    "Only workspace owners and admins can manage members.",
  );
}

export function canViewPermissionManagement(
  ctx: PermissionContext,
): Decision {
  if (isAdminLike(ctx.role)) return ALLOW;
  return deny(
    "not_admin_role",
    "Only workspace owners and admins can view permission management.",
  );
}
