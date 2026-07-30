"use client";

import type { Skill } from "../types";
import { useCurrentMember } from "./use-current-member";
import { canDeleteSkill, canEditSkill } from "./rules";
import { deny, type Decision } from "./types";

const PENDING: Decision = deny("unknown", "");

export function useSkillPermissions(
  skill: Skill | null,
  workspaceId: string,
): { canEdit: Decision; canDelete: Decision } {
  const { userId, role } = useCurrentMember(workspaceId);
  if (skill === null) {
    return { canEdit: PENDING, canDelete: PENDING };
  }
  const context = { userId, role };
  return {
    canEdit: canEditSkill(skill, context),
    canDelete: canDeleteSkill(skill, context),
  };
}
