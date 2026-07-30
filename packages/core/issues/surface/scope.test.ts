import { describe, expect, it } from "vitest";
import { issueScopeKey } from "./scope";
import { buildIssueSurfaceQueryPlan } from "./query-plan";

describe("issue surface scope", () => {
  it("builds stable six-domain surface keys", () => {
    expect(issueScopeKey({ type: "workspace" })).toBe("workspace:all");
    expect(
      issueScopeKey({ type: "workspace", actorKind: "members" }),
    ).toBe("workspace:members");
    expect(
      issueScopeKey({ type: "my", relation: "assigned", userId: "u1" }),
    ).toBe("my:u1:assigned");
    expect(issueScopeKey({ type: "project", projectId: "p1" })).toBe(
      "project:p1",
    );
    expect(
      issueScopeKey({
        type: "actor",
        actorType: "member",
        actorId: "u1",
        relation: "created",
      }),
    ).toBe("actor:member:u1:created");
  });

  it("builds workspace and member query plans", () => {
    expect(buildIssueSurfaceQueryPlan({ type: "workspace" })).toMatchObject({
      kind: "workspace",
      scopeKey: "workspace:all",
      queryFilter: {},
      createDefaults: {},
    });
    expect(
      buildIssueSurfaceQueryPlan({ type: "workspace", actorKind: "members" }),
    ).toMatchObject({
      kind: "scoped",
      scopeKey: "workspace:members",
      queryFilter: { assignee_types: ["member"] },
      createDefaults: {},
    });
  });

  it("builds personal, project, and actor plans", () => {
    expect(
      buildIssueSurfaceQueryPlan({
        type: "my",
        relation: "assigned",
        userId: "u1",
      }),
    ).toMatchObject({
      queryScope: "assigned",
      queryFilter: { assignee_id: "u1" },
      createDefaults: { assignee_type: "member", assignee_id: "u1" },
    });
    expect(
      buildIssueSurfaceQueryPlan({ type: "project", projectId: "p1" }),
    ).toMatchObject({
      queryFilter: { project_id: "p1" },
      createDefaults: { project_id: "p1" },
    });
    expect(
      buildIssueSurfaceQueryPlan({
        type: "actor",
        actorType: "member",
        actorId: "u1",
        relation: "assigned",
      }),
    ).toMatchObject({
      queryFilter: { assignee_id: "u1" },
      createDefaults: { assignee_type: "member", assignee_id: "u1" },
    });
  });
});
