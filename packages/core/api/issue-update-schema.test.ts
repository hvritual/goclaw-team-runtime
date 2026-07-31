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
