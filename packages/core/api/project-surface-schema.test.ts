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
  order_revision: 1,
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

  it("requires one consistent positive Pin order revision and sends the exact reorder contract", async () => {
    const { order_revision: _revision, ...missingRevision } = PIN;
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify([missingRevision]), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify([PIN, { ...PIN, id: "pin-2", order_revision: 2 }]), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify([{ ...PIN, unexpected: true }]), { status: 200 }))
      .mockResolvedValueOnce(new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("http://localhost:8000");

    await expect(client.listPins()).resolves.toEqual([]);
    await expect(client.listPins()).resolves.toEqual([]);
    await expect(client.listPins()).resolves.toEqual([]);
    await expect(client.reorderPins({ items: [{ id: "pin-2" }, { id: "pin-1" }], expected_revision: 7 })).resolves.toBeUndefined();
    expect(fetchMock.mock.calls[3]?.[0]).toBe("http://localhost:8000/api/pins/reorder");
    expect(fetchMock.mock.calls[3]?.[1]).toMatchObject({
      method: "PUT",
      body: JSON.stringify({ items: [{ id: "pin-2" }, { id: "pin-1" }], expected_revision: 7 }),
    });
  });
});
