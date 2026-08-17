import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiClient } from "./client";

const TASK = {
  id: "task-1",
  workspace_id: "workspace-1",
  project_id: null,
  issue_id: null,
  title: "Ship S02A",
  description: "",
  status: "todo",
  priority: "none",
  assignee_type: null,
  assignee_id: null,
  creator_type: "member",
  creator_id: "member-1",
  position: 10,
  revision: 1,
  start_date: null,
  due_date: "2026-08-20T00:00:00Z",
  completed_at: null,
  archived_at: null,
  restore_status: "",
  created_at: "2026-08-18T00:00:00Z",
  updated_at: "2026-08-18T00:00:00Z",
};

const ISSUE = {
  id: "issue-1",
  workspace_id: "workspace-1",
  number: 42,
  identifier: "ONE-42",
  title: "Ship S02A",
  description: "",
  status: "todo",
  priority: "none",
  assignee_type: null,
  assignee_id: null,
  creator_type: "member",
  creator_id: "member-1",
  parent_issue_id: null,
  project_id: null,
  position: 10,
  stage: null,
  start_date: null,
  due_date: "2026-08-20T00:00:00Z",
  metadata: {},
  properties: {},
  created_at: "2026-08-18T00:00:00Z",
  updated_at: "2026-08-18T00:00:00Z",
};

describe("task API boundary", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("parses list and mutation responses and sends revisions", async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ tasks: [TASK], total: 1, next_cursor: "opaque-next" }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ ...TASK, revision: 2, status: "in_progress" }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ tasks: [{ ...TASK, revision: 2, position: 20 }] }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("http://localhost:8000");

    await expect(client.listTasks({ status: "todo", limit: 50, cursor: "opaque-current" })).resolves.toEqual({ tasks: [TASK], total: 1, next_cursor: "opaque-next" });
    await expect(client.updateTask("task-1", { status: "in_progress", expected_revision: 1 })).resolves.toMatchObject({ revision: 2 });
    await expect(client.reorderTasks({ items: [{ id: "task-1", position: 20, expected_revision: 1 }] })).resolves.toHaveLength(1);
    expect(fetchMock.mock.calls[1]?.[1]?.body).toBe(JSON.stringify({ status: "in_progress", expected_revision: 1 }));
    expect(new Headers(fetchMock.mock.calls[2]?.[1]?.headers).get("Idempotency-Key")).toBeTruthy();
    expect(fetchMock.mock.calls[0]?.[0]).toContain("limit=50");
    expect(fetchMock.mock.calls[0]?.[0]).toContain("cursor=opaque-current");
  });

  it("fails closed for malformed mutation responses and degrades malformed lists", async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ tasks: [{ ...TASK, revision: "1" }], total: 1 }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ ...TASK, revision: "2" }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("http://localhost:8000");

    await expect(client.listTasks()).resolves.toEqual({ tasks: [], total: 0, next_cursor: null });
    await expect(client.updateTask("task-1", { title: "Changed", expected_revision: 1 })).rejects.toThrow("Invalid task response");
  });

  it("adds an idempotency key to Task creation", async () => {
    const fetchMock = vi.fn().mockResolvedValueOnce(new Response(JSON.stringify(TASK), { status: 201 }));
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("http://localhost:8000");

    await expect(client.createTask({ title: "Ship S02A" })).resolves.toMatchObject({ id: "task-1" });
    expect(new Headers(fetchMock.mock.calls[0]?.[1]?.headers).get("Idempotency-Key")).toBeTruthy();
  });

  it("promotes a Task with an idempotency key and rejects malformed responses", async () => {
    const response = { task: { ...TASK, issue_id: "issue-1", revision: 2 }, issue: ISSUE, source_task_id: "task-1" };
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify(response), { status: 201 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ ...response, issue: { ...ISSUE, unexpected: "value" } }), { status: 201 }));
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("http://localhost:8000");

    await expect(client.promoteTask("task-1", { expected_revision: 1, complete_task: true, idempotency_key: "promotion-retry-key" })).resolves.toEqual(response);
    expect(fetchMock.mock.calls[0]?.[0]).toBe("http://localhost:8000/api/tasks/task-1/promote");
    expect(fetchMock.mock.calls[0]?.[1]?.body).toBe(JSON.stringify({ expected_revision: 1, complete_task: true }));
    expect(new Headers(fetchMock.mock.calls[0]?.[1]?.headers).get("Idempotency-Key")).toBe("promotion-retry-key");

    await expect(client.promoteTask("task-1", { expected_revision: 1 })).rejects.toThrow("Invalid task promotion response");
  });
});
