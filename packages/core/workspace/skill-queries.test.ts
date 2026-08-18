import { describe, expect, it } from "vitest";
import { skillDetailOptions } from "./queries";

describe("Skill query keys", () => {
  it("scope exact versions by Workspace, Skill, and version", () => {
    expect(
      skillDetailOptions("workspace-1", "skill-1", "version-1").queryKey
    ).toEqual([
      "workspaces",
      "workspace-1",
      "skills",
      "skill-1",
      "versions",
      "version-1",
    ]);
    expect(
      skillDetailOptions("workspace-2", "skill-1", "version-1").queryKey
    ).not.toEqual(
      skillDetailOptions("workspace-1", "skill-1", "version-1").queryKey
    );
  });
});
