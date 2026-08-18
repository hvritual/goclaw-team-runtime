import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiClient } from "./client";

const PROJECT_HIT = {
  id: "project-1",
  workspace_id: "workspace-1",
  title: "修复咖啡机搜索",
  description: "Project search",
  icon: null,
  status: "planned",
  priority: "none",
  lead_type: null,
  lead_id: null,
  start_date: null,
  due_date: null,
  created_at: "2026-08-18T00:00:00Z",
  updated_at: "2026-08-18T00:00:00Z",
  issue_count: 0,
  done_count: 0,
  resource_count: 0,
  match_source: "title",
};

describe("Project search API boundary", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("forwards the exact query contract and AbortSignal", async () => {
    const fetchMock = vi.fn().mockResolvedValueOnce(
      new Response(JSON.stringify({ projects: [PROJECT_HIT], total: 1 }), { status: 200 }),
    );
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("http://localhost:8000");
    const controller = new AbortController();

    await expect(client.searchProjects({
      q: "咖啡机", limit: 50, offset: 2, include_closed: true, signal: controller.signal,
    })).resolves.toEqual({ projects: [PROJECT_HIT], total: 1 });

    expect(fetchMock.mock.calls[0]?.[0]).toBe(
      "http://localhost:8000/api/projects/search?q=%E5%92%96%E5%95%A1%E6%9C%BA&limit=50&offset=2&include_closed=true",
    );
    expect(fetchMock.mock.calls[0]?.[1]?.signal).toBe(controller.signal);
  });

  it("fails closed on unknown fields, invalid sources, or malformed totals", async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ projects: [{ ...PROJECT_HIT, unexpected: true }], total: 1 }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ projects: [{ ...PROJECT_HIT, match_source: "name" }], total: 1 }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ projects: [PROJECT_HIT], total: "1" }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("http://localhost:8000");

    await expect(client.searchProjects({ q: "project" })).resolves.toEqual({ projects: [], total: 0 });
    await expect(client.searchProjects({ q: "project" })).resolves.toEqual({ projects: [], total: 0 });
    await expect(client.searchProjects({ q: "project" })).resolves.toEqual({ projects: [], total: 0 });
  });
});
