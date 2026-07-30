import { describe, expect, it } from "vitest";
import { resolveTabPresentation } from "./tab-presentation";

describe("tab presentation", () => {
  it("presents task details", () => {
    expect(
      resolveTabPresentation(
        { kind: "task", id: "t1" },
        { task: { title: "Prepare release" } },
      ),
    ).toEqual({
      visual: { kind: "icon", icon: "ListChecks" },
      title: { kind: "text", text: "Prepare release" },
    });
  });

  it("presents member details", () => {
    expect(
      resolveTabPresentation(
        { kind: "actor", actorType: "member", id: "u1" },
        { actorName: "Ada" },
      ),
    ).toEqual({
      visual: { kind: "actor", actorType: "member", id: "u1" },
      title: { kind: "text", text: "Ada" },
    });
  });
});
