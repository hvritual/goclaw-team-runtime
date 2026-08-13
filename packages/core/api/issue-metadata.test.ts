import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiClient } from "./client";
import { setCurrentWorkspace } from "../platform/workspace-storage";

const ISSUE_WITHOUT_METADATA = {
  id: "issue-1",
  workspace_id: "workspace-1",
  number: 1,
  identifier: "MUL-1",
  title: "Metadata contract",
  description: null,
  status: "todo",
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
  properties: {},
  created_at: "2026-08-13T00:00:00Z",
  updated_at: "2026-08-13T00:00:00Z",
};

describe("issue metadata API boundary", () => {
  afterEach(() => { vi.unstubAllGlobals(); setCurrentWorkspace(null, null); });

  it("parses getIssue through IssueSchema and defaults missing metadata", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(
      JSON.stringify(ISSUE_WITHOUT_METADATA),
      { status: 200, headers: { "Content-Type": "application/json" } },
    )));
    const issue = await new ApiClient("http://localhost:3000").getIssue("issue-1");
    expect(issue.metadata).toEqual({});
  });

  it("uses dedicated metadata paths, encodes keys, and preserves request body shape", async () => {
    const fetchMock = vi.fn().mockImplementation(async () => new Response(
      JSON.stringify({ metadata: { count: 2, enabled: true, channel: "beta" } }),
      { status: 200, headers: { "Content-Type": "application/json" } },
    ));
    vi.stubGlobal("fetch", fetchMock);
    setCurrentWorkspace("acme", "workspace-1");
    const client = new ApiClient("http://localhost:3000");

    expect(await client.getIssueMetadata("MUL-1")).toEqual({ count: 2, enabled: true, channel: "beta" });
    expect(await client.putIssueMetadata("MUL-1", "release/channel", 2)).toEqual({ count: 2, enabled: true, channel: "beta" });
    expect(await client.deleteIssueMetadata("MUL-1", "release/channel")).toEqual({ count: 2, enabled: true, channel: "beta" });

    expect(fetchMock.mock.calls[0]?.[0]).toBe("http://localhost:3000/api/issues/MUL-1/metadata");
    expect((fetchMock.mock.calls[0]?.[1] as RequestInit).headers).toMatchObject({ "X-Workspace-Slug": "acme", "X-Workspace-ID": "workspace-1" });
    expect(fetchMock.mock.calls[1]?.[0]).toBe("http://localhost:3000/api/issues/MUL-1/metadata/release%2Fchannel");
    expect(JSON.parse((fetchMock.mock.calls[1]?.[1] as RequestInit).body as string)).toEqual({ value: 2 });
    expect(fetchMock.mock.calls[2]?.[0]).toBe("http://localhost:3000/api/issues/MUL-1/metadata/release%2Fchannel");
  });

  it("rejects malformed metadata responses instead of coercing values", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ metadata: { nested: { bad: true } } }),
      { status: 200, headers: { "Content-Type": "application/json" } },
    )));
    await expect(new ApiClient("http://localhost:3000").getIssueMetadata("issue-1"))
      .rejects.toThrow("Invalid issue metadata response");
  });

  it("rejects a metadata response with the required envelope missing", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(
      JSON.stringify({}),
      { status: 200, headers: { "Content-Type": "application/json" } },
    )));
    await expect(new ApiClient("http://localhost:3000").getIssueMetadata("issue-1"))
      .rejects.toThrow("Invalid issue metadata response");
  });
});
