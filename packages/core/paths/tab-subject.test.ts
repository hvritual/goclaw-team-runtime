import { describe, expect, it } from "vitest";
import { parseTabSubject, tabSubjectKey } from "./tab-subject";

describe("tab subjects", () => {
  it.each([
    ["/acme/issues", { kind: "page", page: "issues" }],
    ["/acme/projects/p1", { kind: "project", id: "p1" }],
    ["/acme/tasks/t1", { kind: "task", id: "t1" }],
    ["/acme/members/u1", { kind: "actor", actorType: "member", id: "u1" }],
    ["/acme/skills/s1", { kind: "skill", id: "s1" }],
    ["/acme/settings?tab=members", { kind: "page", page: "settings" }],
  ] as const)("parses %s", (url, expected) => {
    expect(parseTabSubject(url)).toEqual(expected);
  });

  it("uses stable keys for retained domains", () => {
    expect(tabSubjectKey({ kind: "task", id: "t1" })).toBe("task:t1");
    expect(
      tabSubjectKey({ kind: "actor", actorType: "member", id: "u1" }),
    ).toBe("actor:member:u1");
  });
});
