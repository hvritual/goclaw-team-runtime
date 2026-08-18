import { describe, expect, it } from "vitest";
import type { Skill } from "../types";
import { canDeleteSkill, canEditSkill } from "./rules";

const SKILL: Skill = {
  id: "skill-1",
  workspace_id: "workspace-1",
  version_id: "version-1",
  version: "1",
  name: "Release helper",
  description: "",
  config: {},
  status: "draft",
  revision: 1,
  created_by: "creator-1",
  created_at: "2026-08-18T00:00:00Z",
  updated_at: "2026-08-18T00:00:00Z",
  archived: false,
  content: "",
  files: [],
};

describe("Skill administration permissions", () => {
  it("denies the creator unless they are a workspace owner or admin", () => {
    const creator = { userId: "creator-1", role: "member" as const };
    expect(canEditSkill(SKILL, creator).allowed).toBe(false);
    expect(canDeleteSkill(SKILL, creator).allowed).toBe(false);
    expect(
      canEditSkill(SKILL, { userId: "admin-1", role: "admin" }).allowed
    ).toBe(true);
  });
});
