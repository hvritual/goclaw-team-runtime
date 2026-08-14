import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiClient } from "./client";

const PROJECT = {
  id: "project-1",
  workspace_id: "workspace-1",
  title: "Launch",
  description: null,
  icon: null,
  status: "planned",
  priority: "none",
  lead_type: null,
  lead_id: null,
  start_date: null,
  due_date: null,
  created_at: "2026-08-14T00:00:00Z",
  updated_at: "2026-08-14T00:00:00Z",
  issue_count: 0,
  done_count: 0,
  resource_count: 0,
};

const PIN = {
  id: "pin-1",
  workspace_id: "workspace-1",
  user_id: "user-1",
  item_type: "project",
  item_id: "project-1",
  position: 0,
  created_at: "2026-08-14T00:00:00Z",
};

describe("project surface API boundary", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("accepts exact Project and Pin responses", async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ projects: [PROJECT], total: 1 }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify(PROJECT), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify(PIN), { status: 201 }))
      .mockResolvedValueOnce(new Response(JSON.stringify([PIN]), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("http://localhost:8000");

    await expect(client.listProjects()).resolves.toEqual({ projects: [PROJECT], total: 1 });
    await expect(client.getProject("project-1")).resolves.toEqual(PROJECT);
    await expect(client.createPin({ item_type: "project", item_id: "project-1" })).resolves.toEqual(PIN);
    await expect(client.listPins()).resolves.toEqual([PIN]);
  });

  it("fails closed for malformed list entries and mutation responses", async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ projects: [{ ...PROJECT, id: 7 }], total: 1 }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ ...PROJECT, status: null }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ ...PIN, item_type: "workspace" }), { status: 201 }))
      .mockResolvedValueOnce(new Response(JSON.stringify([{ ...PIN, position: -1 }]), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("http://localhost:8000");

    await expect(client.listProjects()).resolves.toEqual({ projects: [], total: 0 });
    await expect(client.getProject("project-1")).rejects.toThrow("Invalid project response");
    await expect(client.createPin({ item_type: "project", item_id: "project-1" })).rejects.toThrow("Invalid pin response");
    await expect(client.listPins()).resolves.toEqual([]);
  });

  it("requires every frozen list field and preserves legacy fractional pin positions", async () => {
    const { issue_count: _issueCount, ...missingCount } = PROJECT;
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ projects: [missingCount], total: 1 }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ projects: [PROJECT] }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify([{ ...PIN, position: 1.5 }]), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("http://localhost:8000");

    await expect(client.listProjects()).resolves.toEqual({ projects: [], total: 0 });
    await expect(client.listProjects()).resolves.toEqual({ projects: [], total: 0 });
    await expect(client.listPins()).resolves.toEqual([{ ...PIN, position: 1.5 }]);
  });
});
