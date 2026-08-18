import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiClient } from "./client";

const ISSUE_HIT = {
  id: "issue-1",
  workspace_id: "workspace-1",
  number: 41,
  identifier: "WSP-41",
  title: "修复咖啡机搜索",
  description: "Issue search",
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
  metadata: {},
  properties: {},
  created_at: "2026-08-18T00:00:00Z",
  updated_at: "2026-08-18T00:00:00Z",
  match_source: "identifier",
};

describe("Issue search API boundary", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("forwards the exact query contract and AbortSignal", async () => {
    const fetchMock = vi.fn().mockResolvedValueOnce(
      new Response(JSON.stringify({ issues: [ISSUE_HIT], total: 1 }), { status: 200 }),
    );
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("http://localhost:8000");
    const controller = new AbortController();

    await expect(client.searchIssues({
      q: "咖啡机", limit: 50, offset: 2, include_closed: true, signal: controller.signal,
    })).resolves.toEqual({ issues: [ISSUE_HIT], total: 1 });

    expect(fetchMock.mock.calls[0]?.[0]).toBe(
      "http://localhost:8000/api/issues/search?q=%E5%92%96%E5%95%A1%E6%9C%BA&limit=50&offset=2&include_closed=true",
    );
    expect(fetchMock.mock.calls[0]?.[1]?.signal).toBe(controller.signal);
  });

  it("fails closed on unknown fields, invalid sources, or malformed totals", async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ issues: [{ ...ISSUE_HIT, unexpected: true }], total: 1 }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ issues: [{ ...ISSUE_HIT, match_source: "comment" }], total: 1 }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ issues: [ISSUE_HIT], total: "1" }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("http://localhost:8000");

    await expect(client.searchIssues({ q: "issue" })).resolves.toEqual({ issues: [], total: 0 });
    await expect(client.searchIssues({ q: "issue" })).resolves.toEqual({ issues: [], total: 0 });
    await expect(client.searchIssues({ q: "issue" })).resolves.toEqual({ issues: [], total: 0 });
  });
});
