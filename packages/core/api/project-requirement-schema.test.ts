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
});
