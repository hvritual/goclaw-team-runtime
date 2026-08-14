import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiClient } from "./client";

const VALID_ISSUE = {
  id: "issue-1",
  workspace_id: "workspace-1",
  number: 1,
  identifier: "MUL-1",
  title: "Acceptance boundary",
  description: null,
  status: "done",
  priority: "none",
  assignee_type: null,
  assignee_id: null,
  creator_type: "member",
  creator_id: "member-1",
  parent_issue_id: null,
  project_id: null,
  position: 0,
  stage: null,
  start_date: null,
  due_date: null,
  metadata: {},
  properties: {},
  created_at: "2026-08-01T00:00:00Z",
  updated_at: "2026-08-01T01:00:00Z",
};

describe("issue update API boundary", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("rejects a malformed update response", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ id: 42, status: null }),
      { status: 200, headers: { "Content-Type": "application/json" } },
    )));
    const client = new ApiClient("http://localhost:3000");

    await expect(client.updateIssue("issue-1", { status: "done" })).rejects.toThrow(
      "Invalid issue update response",
    );
  });

  it("rejects a malformed move response", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ id: "issue-1", position: "between" }),
      { status: 200, headers: { "Content-Type": "application/json" } },
    )));
    const client = new ApiClient("http://localhost:3000");

    await expect(client.moveIssue("issue-1", {
      before_id: null,
      after_id: null,
    })).rejects.toThrow("Invalid issue move response");
  });

  it("rejects malformed hierarchy and batch responses", async () => {
    const responses = [
      { issues: [{ ...VALID_ISSUE, status: "invented" }] },
      {},
      { progress: [{ parent_issue_id: "parent", total: -1, done: 2 }] },
      { updated: "2" },
      { deleted: -1 },
    ];
    vi.stubGlobal("fetch", vi.fn().mockImplementation(async () => new Response(
      JSON.stringify(responses.shift()),
      { status: 200, headers: { "Content-Type": "application/json" } },
    )));
    const client = new ApiClient("http://localhost:3000");

    await expect(client.listChildIssues("parent")).rejects.toThrow("Invalid child issues response");
    await expect(client.listChildrenByParents(["parent"])).rejects.toThrow("Invalid child issues response");
    await expect(client.getChildIssueProgress()).rejects.toThrow("Invalid child issue progress response");
    await expect(client.batchUpdateIssues(["issue-1"], { status: "done" })).rejects.toThrow("Invalid batch issue update response");
    await expect(client.batchDeleteIssues(["issue-1"])).rejects.toThrow("Invalid batch issue delete response");
  });

  it("accepts and serializes exact hierarchy and batch contracts", async () => {
    const child = { ...VALID_ISSUE, id: "child-1", parent_issue_id: "parent-1" };
    const responses = [
      { issues: [child] },
      { issues: [child] },
      { progress: [{ parent_issue_id: "parent-1", total: 2, done: 1 }] },
      { updated: 2 },
      { deleted: 2 },
    ];
    const fetchMock = vi.fn().mockImplementation(async () => new Response(
      JSON.stringify(responses.shift()),
      { status: 200, headers: { "Content-Type": "application/json" } },
    ));
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("http://localhost:3000");

    await expect(client.listChildIssues("parent-1")).resolves.toEqual({ issues: [child] });
    await expect(client.listChildrenByParents(["parent-1", "parent-2"])).resolves.toEqual({ issues: [child] });
    await expect(client.getChildIssueProgress()).resolves.toEqual({
      progress: [{ parent_issue_id: "parent-1", total: 2, done: 1 }],
    });
    await expect(client.batchUpdateIssues(["issue-1", "issue-2"], {
      status: "done",
      parent_issue_id: null,
    })).resolves.toEqual({ updated: 2 });
    await expect(client.batchDeleteIssues(["issue-1", "issue-2"])).resolves.toEqual({ deleted: 2 });

    expect(fetchMock.mock.calls.map((call) => String(call[0]))).toEqual([
      "http://localhost:3000/api/issues/parent-1/children",
      "http://localhost:3000/api/issues/children?parent_ids=parent-1,parent-2",
      "http://localhost:3000/api/issues/child-progress",
      "http://localhost:3000/api/issues/batch-update",
      "http://localhost:3000/api/issues/batch-delete",
    ]);
    expect(JSON.parse((fetchMock.mock.calls[3]?.[1] as RequestInit).body as string)).toEqual({
      issue_ids: ["issue-1", "issue-2"],
      updates: { status: "done", parent_issue_id: null },
    });
    expect(JSON.parse((fetchMock.mock.calls[4]?.[1] as RequestInit).body as string)).toEqual({
      issue_ids: ["issue-1", "issue-2"],
    });
  });

  it("serializes acceptance input only at the HTTP boundary", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(
      JSON.stringify(VALID_ISSUE),
      { status: 200, headers: { "Content-Type": "application/json" } },
    ));
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("http://localhost:3000");

    await client.updateIssue("issue-1", {
      status: "done",
      acceptanceConclusion: {
        result: "accepted",
        rationale: "All checks passed.",
        evidenceRefs: ["artifact://report/v2"],
      },
    });

    const request = fetchMock.mock.calls[0]?.[1] as RequestInit;
    expect(JSON.parse(request.body as string)).toMatchObject({
      status: "done",
      acceptance_conclusion: {
        result: "accepted",
        rationale: "All checks passed.",
        evidence_refs: ["artifact://report/v2"],
      },
    });
  });

  it("rejects malformed acceptance and retrospective mutation responses", async () => {
    vi.stubGlobal("fetch", vi.fn().mockImplementation(async () => new Response(
      JSON.stringify({ id: 42, lessons: "none" }),
      { status: 200, headers: { "Content-Type": "application/json" } },
    )));
    const client = new ApiClient("http://localhost:3000");

    await expect(client.createIssueAcceptanceConclusion("issue-1", {
      result: "accepted", rationale: "Passed", evidenceRefs: [],
    })).rejects.toThrow("Invalid acceptance conclusion response");
    await expect(client.createProjectRetrospective("project-1", {
      summary: "Delivery", successes: [], problems: [], lessons: ["Review sooner"], followUpRefs: [],
    })).rejects.toThrow("Invalid project retrospective response");
  });
});
