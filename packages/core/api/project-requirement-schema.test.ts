import { afterEach, describe, expect, it, vi } from "vitest";
import { noopLogger } from "../logger";
import { setSchemaLogger } from "./schema";
import { ApiClient } from "./client";

const CONTENT = {
  problem_statement: "Define scope",
  goals: [{ key: "goal-1", text: "Ship safely" }],
  in_scope: [],
  out_of_scope: [],
  constraints: [],
  acceptance_criteria: [],
  dependencies: [],
};

const VALID_RESPONSE = {
  baseline: {
    id: "baseline-1",
    workspace_id: "workspace-1",
    project_id: "project-1",
    status: "changed",
    current_revision: 8,
    approved_revision: 6,
    effective_revision: 6,
    submitted_by: null,
    submitted_at: null,
    approved_by: "owner-1",
    approved_at: "2026-08-01T00:00:00Z",
    frozen_by: "owner-1",
    frozen_at: "2026-08-01T01:00:00Z",
    retired_by: null,
    retired_at: null,
    created_at: "2026-08-01T00:00:00Z",
    updated_at: "2026-08-01T02:00:00Z",
  },
  current_content: CONTENT,
  effective_content: { ...CONTENT, problem_statement: "Effective scope" },
  history: [
    {
      baseline_id: "baseline-1",
      revision: 8,
      content: CONTENT,
      state: "changed",
      action: "material_change",
      change_summary: "Expand scope",
      actor_id: "member-1",
      submitted_by: null,
      submitted_at: null,
      approved_by: "owner-1",
      approved_at: "2026-08-01T00:00:00Z",
      frozen_by: "owner-1",
      frozen_at: "2026-08-01T01:00:00Z",
      created_at: "2026-08-01T02:00:00Z",
    },
  ],
  issue_links: [
    {
      requirement_key: "goal-1",
      issue_id: "issue-1",
      identifier: "MUL-1",
      title: "Ship",
      status: "started",
      linked_revision: 5,
      review_required: true,
      linked_by: "member-1",
      linked_at: "2026-08-01T00:30:00Z",
      unlinked_at: null,
    },
  ],
  outline_links: [
    {
      requirement_key: "goal-1",
      node_id: "node-1",
      node_title: "Delivery",
      linked_revision: 7,
      linked_by: "member-1",
      linked_at: "2026-08-01T01:30:00Z",
      unlinked_at: null,
    },
  ],
  access: {
    can_edit: true,
    can_approve: false,
    can_manage_access: false,
    can_manage_outline: true,
  },
};

const VALID_COVERAGE_RESPONSE = {
  baseline_status: "retired",
  current: {
    revision: 8,
    state: "retired",
    total: 2,
    linked: 1,
    implemented: 1,
    accepted: 1,
    unlinked: 1,
    items: [
      {
        requirement_key: "goal-1",
        section: "goals",
        text: "Ship safely",
        stage: "accepted",
        issues: [
          {
            id: "issue-1",
            identifier: "MUL-1",
            title: "Ship",
            status: "done",
            acceptance_result: "accepted",
          },
        ],
      },
      {
        requirement_key: "scope-1",
        section: "in_scope",
        text: "Document scope",
        stage: "unlinked",
        issues: [],
      },
    ],
  },
  effective: {
    revision: 6,
    state: "frozen",
    total: 1,
    linked: 1,
    implemented: 0,
    accepted: 0,
    unlinked: 0,
    items: [
      {
        requirement_key: "goal-1",
        section: "goals",
        text: "Ship safely",
        stage: "linked",
        issues: [
          {
            id: "issue-1",
            identifier: "MUL-1",
            title: "Ship",
            status: "in_progress",
            acceptance_result: null,
          },
        ],
      },
    ],
  },
};

function jsonResponse(value: unknown, status = 200): Response {
  return new Response(JSON.stringify(value), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

function draftInput(expectedRevision = 8) {
  return {
    expectedRevision,
    content: {
      problemStatement: "Define scope",
      goals: [{ key: "goal-1", text: "Ship safely" }],
      inScope: [],
      outOfScope: [],
      constraints: [],
      acceptanceCriteria: [],
      dependencies: [],
    },
    changeSummary: "Clarify scope",
    materialChange: true,
    idempotencyKey: "requirement-create-key",
  };
}

describe("project requirement API boundary", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    setSchemaLogger(noopLogger);
  });

  it("throws and emits only safe diagnostics for a malformed baseline read", async () => {
    const warn = vi.fn();
    setSchemaLogger({ ...noopLogger, warn });
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValue(
          jsonResponse({ baseline: { id: 42, secret: "do-not-log" } })
        )
    );

    await expect(
      new ApiClient("http://localhost:3000").getProjectRequirementBaseline(
        "project-1"
      )
    ).rejects.toThrow("Invalid project requirement baseline response");

    expect(warn).toHaveBeenCalledWith(
      expect.stringContaining("GET /api/projects/:id/requirement-baseline"),
      expect.objectContaining({
        endpoint: "GET /api/projects/:id/requirement-baseline",
        issues: expect.any(Array),
        receivedShape: { kind: "object", fieldCount: 1 },
      })
    );
    expect(JSON.stringify(warn.mock.calls)).not.toContain("do-not-log");
  });

  it("strictly maps the complete lifecycle, history, links, impact, and access projection", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(jsonResponse(VALID_RESPONSE))
    );

    const response = await new ApiClient(
      "http://localhost:3000"
    ).getProjectRequirementBaseline("project-1");

    expect(response.baseline).toMatchObject({
      status: "changed",
      currentRevision: 8,
      effectiveRevision: 6,
      frozenBy: "owner-1",
      retiredAt: null,
    });
    expect(response.currentContent?.problemStatement).toBe("Define scope");
    expect(response.effectiveContent?.problemStatement).toBe("Effective scope");
    expect(response.history[0]).toMatchObject({
      action: "material_change",
      state: "changed",
      frozenBy: "owner-1",
    });
    expect(response.issueLinks[0]).toMatchObject({
      requirementKey: "goal-1",
      issueId: "issue-1",
      reviewRequired: true,
    });
    expect(response.outlineLinks[0]).toMatchObject({
      requirementKey: "goal-1",
      nodeId: "node-1",
      nodeTitle: "Delivery",
    });
    expect(response.access).toEqual({
      canEdit: true,
      canApprove: false,
      canManageAccess: false,
      canManageOutline: true,
    });
  });

  it("rejects unknown lifecycle values and missing response fields", async () => {
    const client = new ApiClient("http://localhost:3000");
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        jsonResponse({
          ...VALID_RESPONSE,
          baseline: { ...VALID_RESPONSE.baseline, status: "future" },
        })
      )
      .mockResolvedValueOnce(
        jsonResponse({ ...VALID_RESPONSE, access: undefined })
      );
    vi.stubGlobal("fetch", fetchMock);

    await expect(
      client.getProjectRequirementBaseline("project-1")
    ).rejects.toThrow("Invalid project requirement baseline response");
    await expect(
      client.getProjectRequirementBaseline("project-1")
    ).rejects.toThrow("Invalid project requirement baseline response");
  });

  it("strictly maps current and effective Requirement coverage without fallback", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(jsonResponse(VALID_COVERAGE_RESPONSE))
    );

    const coverage = await new ApiClient(
      "http://localhost:3000"
    ).getProjectRequirementCoverage("project-1");

    expect(coverage.baselineStatus).toBe("retired");
    expect(coverage.current).toMatchObject({
      revision: 8,
      state: "retired",
      total: 2,
      linked: 1,
      implemented: 1,
      accepted: 1,
      unlinked: 1,
    });
    expect(coverage.current?.items[0]).toMatchObject({
      requirementKey: "goal-1",
      section: "goals",
      text: "Ship safely",
      stage: "accepted",
    });
    expect(coverage.current?.items[0]?.issues[0]).toMatchObject({
      identifier: "MUL-1",
      status: "done",
      acceptanceResult: "accepted",
    });
    expect(coverage.effective).toMatchObject({ revision: 6, state: "frozen" });
  });

  it("rejects malformed, inconsistent, or partial coverage instead of returning empty data", async () => {
    const client = new ApiClient("http://localhost:3000");
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        jsonResponse({
          ...VALID_COVERAGE_RESPONSE,
          current: { ...VALID_COVERAGE_RESPONSE.current, accepted: 2 },
        })
      )
      .mockResolvedValueOnce(
        jsonResponse({
          ...VALID_COVERAGE_RESPONSE,
          current: {
            ...VALID_COVERAGE_RESPONSE.current,
            items: [
              {
                ...VALID_COVERAGE_RESPONSE.current.items[0],
                stage: "future",
              },
            ],
          },
        })
      )
      .mockResolvedValueOnce(jsonResponse({ current: null, effective: null }));
    vi.stubGlobal("fetch", fetchMock);

    await expect(
      client.getProjectRequirementCoverage("project-1")
    ).rejects.toThrow("Invalid project requirement coverage response");
    await expect(
      client.getProjectRequirementCoverage("project-1")
    ).rejects.toThrow("Invalid project requirement coverage response");
    await expect(
      client.getProjectRequirementCoverage("project-1")
    ).rejects.toThrow("Invalid project requirement coverage response");
  });

  it("serializes the exact draft wire contract and reusable create key", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(jsonResponse(VALID_RESPONSE, 201));
    vi.stubGlobal("fetch", fetchMock);

    const response = await new ApiClient(
      "http://localhost:3000"
    ).saveProjectRequirementDraft("project-1", draftInput(0));
    const request = fetchMock.mock.calls[0]?.[1] as RequestInit;

    expect(new Headers(request.headers).get("Idempotency-Key")).toBe(
      "requirement-create-key"
    );
    expect(JSON.parse(request.body as string)).toEqual({
      expected_revision: 0,
      content: CONTENT,
      change_summary: "Clarify scope",
      material_change: true,
    });
    expect(response.baseline?.currentRevision).toBe(8);
  });

  it("throws for malformed draft and transition responses", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockImplementation(() =>
          Promise.resolve(jsonResponse({ baseline: { id: 42 } }))
        )
    );
    const client = new ApiClient("http://localhost:3000");

    await expect(
      client.saveProjectRequirementDraft("project-1", draftInput())
    ).rejects.toThrow("Invalid project requirement draft response");
    await expect(
      client.freezeProjectRequirement("project-1", { expectedRevision: 8 })
    ).rejects.toThrow("Invalid project requirement transition response");
  });

  it("supports every explicit lifecycle route with expected_revision", async () => {
    const fetchMock = vi
      .fn()
      .mockImplementation(() => Promise.resolve(jsonResponse(VALID_RESPONSE)));
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("http://localhost:3000");

    await client.submitProjectRequirementReview("project-1", {
      expectedRevision: 1,
    });
    await client.withdrawProjectRequirementReview("project-1", {
      expectedRevision: 2,
    });
    await client.approveProjectRequirement("project-1", {
      expectedRevision: 3,
    });
    await client.freezeProjectRequirement("project-1", { expectedRevision: 4 });
    await client.retireProjectRequirement("project-1", { expectedRevision: 5 });

    expect(fetchMock.mock.calls.map(([url]) => url)).toEqual([
      "http://localhost:3000/api/projects/project-1/requirement-baseline/submit-review",
      "http://localhost:3000/api/projects/project-1/requirement-baseline/withdraw",
      "http://localhost:3000/api/projects/project-1/requirement-baseline/approve",
      "http://localhost:3000/api/projects/project-1/requirement-baseline/freeze",
      "http://localhost:3000/api/projects/project-1/requirement-baseline/retire",
    ]);
    expect(
      fetchMock.mock.calls.map(([, init]) =>
        JSON.parse((init as RequestInit).body as string)
      )
    ).toEqual([
      { expected_revision: 1 },
      { expected_revision: 2 },
      { expected_revision: 3 },
      { expected_revision: 4 },
      { expected_revision: 5 },
    ]);
  });

  it("uses exact Issue link bodies, strict POST responses, and bodyless unlink query", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse(VALID_RESPONSE))
      .mockResolvedValueOnce(new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("http://localhost:3000");

    await client.linkProjectRequirementIssue("project-1", {
      requirementKey: "goal 1",
      issueId: "issue/1",
      expectedRevision: 8,
    });
    await client.unlinkProjectRequirementIssue("project-1", {
      requirementKey: "goal 1",
      issueId: "issue/1",
      expectedRevision: 9,
    });

    expect(
      JSON.parse((fetchMock.mock.calls[0]?.[1] as RequestInit).body as string)
    ).toEqual({
      expected_revision: 8,
      requirement_key: "goal 1",
      issue_id: "issue/1",
    });
    expect(fetchMock.mock.calls[1]?.[0]).toBe(
      "http://localhost:3000/api/projects/project-1/requirement-baseline/links/goal%201/issue%2F1?expected_revision=9"
    );
    expect((fetchMock.mock.calls[1]?.[1] as RequestInit).body).toBeUndefined();
  });

  it("strictly handles minimal outline read/create/link and access-set replacement", async () => {
    const outline = {
      revision: 2,
      nodes: [
        {
          id: "node-1",
          workspace_id: "workspace-1",
          project_id: "project-1",
          title: "Delivery",
          created_by: "member-1",
          created_at: "2026-08-01T00:00:00Z",
        },
      ],
    };
    const access = {
      revision: 3,
      grants: [
        {
          member_id: "member-1",
          user_id: "user-1",
          role: "member",
          grant_kind: "project_editor",
          granted_by: "owner-1",
          granted_at: "2026-08-01T00:00:00Z",
        },
      ],
    };
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse(outline))
      .mockResolvedValueOnce(jsonResponse(outline, 201))
      .mockResolvedValueOnce(jsonResponse(VALID_RESPONSE))
      .mockResolvedValueOnce(new Response(null, { status: 204 }))
      .mockResolvedValueOnce(jsonResponse(access))
      .mockResolvedValueOnce(jsonResponse(access));
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("http://localhost:3000");

    expect((await client.getProjectOutline("project-1")).nodes[0]?.title).toBe(
      "Delivery"
    );
    await client.createProjectOutlineNode("project-1", {
      expectedRevision: 2,
      title: "Delivery",
      idempotencyKey: "outline-create-key",
    });
    await client.linkProjectRequirementOutline("project-1", {
      requirementKey: "goal-1",
      nodeId: "node-1",
      expectedRevision: 8,
    });
    await client.unlinkProjectRequirementOutline("project-1", {
      requirementKey: "goal-1",
      nodeId: "node-1",
      expectedRevision: 9,
    });
    expect(
      (await client.getProjectRequirementAccess("project-1")).revision
    ).toBe(3);
    await client.replaceProjectRequirementAccess("project-1", {
      expectedRevision: 3,
      grants: [{ memberId: "member-1", grantKind: "requirement_approver" }],
    });

    expect(
      new Headers((fetchMock.mock.calls[1]?.[1] as RequestInit).headers).get(
        "Idempotency-Key"
      )
    ).toBe("outline-create-key");
    expect(
      JSON.parse((fetchMock.mock.calls[1]?.[1] as RequestInit).body as string)
    ).toEqual({
      expected_revision: 2,
      title: "Delivery",
    });
    expect(
      JSON.parse((fetchMock.mock.calls[2]?.[1] as RequestInit).body as string)
    ).toEqual({
      expected_revision: 8,
      requirement_key: "goal-1",
      node_id: "node-1",
    });
    expect(fetchMock.mock.calls[3]?.[0]).toContain(
      "/outline-links/goal-1/node-1?expected_revision=9"
    );
    expect(
      JSON.parse((fetchMock.mock.calls[5]?.[1] as RequestInit).body as string)
    ).toEqual({
      expected_revision: 3,
      grants: [{ member_id: "member-1", grant_kind: "requirement_approver" }],
    });
  });
});
