import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiClient } from "./client";

const VALID_RESPONSE = {
  baseline: {
    id: "baseline-1", workspace_id: "workspace-1", project_id: "project-1", status: "draft",
    current_revision: 1, approved_revision: null, created_at: "2026-08-01T00:00:00Z", updated_at: "2026-08-01T00:00:00Z",
  },
  current_content: { problem_statement: "Define scope", goals: [], in_scope: [], out_of_scope: [], constraints: [], acceptance_criteria: [], dependencies: [] },
  effective_content: null,
  history: [],
};

describe("project requirement API boundary", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("degrades a malformed baseline read to the empty state", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ baseline: { id: 42 } }), { status: 200, headers: { "Content-Type": "application/json" } })));
    const response = await new ApiClient("http://localhost:3000").getProjectRequirementBaseline("project-1");
    expect(response).toEqual({ baseline: null, currentContent: null, effectiveContent: null, history: [] });
  });

  it("rejects a malformed draft mutation response", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ baseline: { id: 42 } }), { status: 200, headers: { "Content-Type": "application/json" } })));
    const client = new ApiClient("http://localhost:3000");
    await expect(client.saveProjectRequirementDraft("project-1", { expectedRevision: 0, content: { problemStatement: "Define scope", goals: [], inScope: [], outOfScope: [], constraints: [], acceptanceCriteria: [], dependencies: [] }, changeSummary: "Initial" })).rejects.toThrow("Invalid project requirement draft response");
  });

  it("serializes camelCase request fields only at the HTTP boundary", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify(VALID_RESPONSE), { status: 200, headers: { "Content-Type": "application/json" } }));
    vi.stubGlobal("fetch", fetchMock);
    const response = await new ApiClient("http://localhost:3000").saveProjectRequirementDraft("project-1", { expectedRevision: 4, content: { problemStatement: "Define scope", goals: [], inScope: [], outOfScope: [], constraints: [], acceptanceCriteria: [], dependencies: [] }, changeSummary: "Clarify scope" });
    const request = fetchMock.mock.calls[0]?.[1] as RequestInit;
    expect(JSON.parse(request.body as string)).toMatchObject({ expected_revision: 4, change_summary: "Clarify scope", content: { problem_statement: "Define scope", in_scope: [], acceptance_criteria: [] } });
    expect(response.baseline?.currentRevision).toBe(1);
    expect(response.currentContent?.problemStatement).toBe("Define scope");
  });

  it("converts coverage links from snake_case and safely falls back when malformed", async () => {
    const coverage = { current: { revision: 2, total: 1, linked: 1, unlinked: 0, linked_issue_done: 1, linked_issue_blocked: 0, items: [{ requirement_key: "goal-1", section: "goals", issues: [{ id: "issue-1", identifier: "MUL-1", title: "Ship", status: "done", created_by: "member-1", created_at: "2026-08-01T00:00:00Z" }] }] }, effective: null };
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify(coverage), { status: 200, headers: { "Content-Type": "application/json" } })));
    const response = await new ApiClient("http://localhost:3000").getProjectRequirementCoverage("project-1");
    expect(response.current?.linkedIssueDone).toBe(1);
    expect(response.current?.items[0]?.requirementKey).toBe("goal-1");
    expect(response.current?.items[0]?.issues[0]).toMatchObject({ createdBy: "member-1", createdAt: "2026-08-01T00:00:00Z" });
  });

  it("keeps zero-link coverage items as empty arrays and falls back for malformed coverage", async () => {
    const zeroLinks = { current: { revision: 1, total: 1, linked: 0, unlinked: 1, linked_issue_done: 0, linked_issue_blocked: 0, items: [{ requirement_key: "goal-1", section: "goals", issues: [] }] }, effective: null };
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(new Response(JSON.stringify(zeroLinks), { status: 200, headers: { "Content-Type": "application/json" } })).mockResolvedValueOnce(new Response(JSON.stringify({ current: { revision: "bad" } }), { status: 200, headers: { "Content-Type": "application/json" } })));
    const client = new ApiClient("http://localhost:3000");
    expect((await client.getProjectRequirementCoverage("project-1")).current?.items[0]?.issues).toEqual([]);
    expect(await client.getProjectRequirementCoverage("project-1")).toEqual({ current: null, effective: null });
  });

  it("serializes tracking requests at the HTTP boundary", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetchMock);
    await new ApiClient("http://localhost:3000").linkProjectRequirementIssue("project-1", { requirementKey: "goal-1", issueId: "issue-1", revision: 2 });
    expect(JSON.parse((fetchMock.mock.calls[0]?.[1] as RequestInit).body as string)).toEqual({ requirement_key: "goal-1", issue_id: "issue-1", revision: 2 });
  });

  it("serializes unlink and requirement issue creation requests", async () => {
    const fetchMock = vi.fn().mockResolvedValueOnce(new Response(null, { status: 204 })).mockResolvedValueOnce(new Response(JSON.stringify({ id: "issue-1", workspace_id: "workspace-1", number: 1, identifier: "MUL-1", title: "Goal", description: null, status: "todo", priority: "none", assignee_type: null, assignee_id: null, creator_type: "member", creator_id: "member-1", parent_issue_id: null, project_id: "project-1", position: 0, stage: null, start_date: null, due_date: null, metadata: {}, properties: {}, created_at: "2026-08-01T00:00:00Z", updated_at: "2026-08-01T00:00:00Z" }), { status: 201, headers: { "Content-Type": "application/json" } }));
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("http://localhost:3000");
    await client.unlinkProjectRequirementIssue("project-1", { requirementKey: "goal 1", issueId: "issue-1", revision: 2 });
    await client.createIssueForProjectRequirement("project-1", "goal 1", { revision: 2 });
    expect(fetchMock.mock.calls[0]?.[0]).toContain("goal%201/issue-1?revision=2");
    expect(JSON.parse((fetchMock.mock.calls[1]?.[1] as RequestInit).body as string)).toEqual({ revision: 2 });
  });
});
