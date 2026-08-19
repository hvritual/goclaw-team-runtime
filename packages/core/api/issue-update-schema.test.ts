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

const VALID_RETROSPECTIVE_WIRE = {
  id: "retro-1",
  workspace_id: "workspace-1",
  project_id: "project-1",
  status: "published",
  current_revision: 1,
  published_revision: 1,
  created_by: "member-1",
  created_at: "2026-08-01T00:00:00Z",
  updated_at: "2026-08-01T00:00:00Z",
  current: {
    revision: 1,
    status: "published",
    action: "publish",
    content: {
      summary: "Delivery",
      successes: [],
      problems: [],
      lessons: ["Review sooner"],
      action_items: [{ id: "action-1", title: "Schedule review" }],
    },
    participants: [{ member_id: "member-1", role: "participant" }],
    actor_id: "member-1",
    created_at: "2026-08-01T00:00:00Z",
  },
  history: [{
    revision: 1,
    status: "published",
    action: "publish",
    content: {
      summary: "Delivery",
      successes: [],
      problems: [],
      lessons: ["Review sooner"],
      action_items: [{ id: "action-1", title: "Schedule review" }],
    },
    participants: [{ member_id: "member-1", role: "participant" }],
    actor_id: "member-1",
    created_at: "2026-08-01T00:00:00Z",
  }],
  action_links: [{
    retrospective_id: "retro-1",
    action_item_id: "action-1",
    source_revision: 1,
    state: "linked",
    target_kind: "task",
    target_id: "task-1",
    created_by: "member-1",
    created_at: "2026-08-01T01:00:00Z",
  }],
  access: { can_edit: false, can_publish: true, can_archive: true },
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
      content: {
        summary: "Delivery", successes: [], problems: [], lessons: ["Review sooner"], actionItems: [],
      },
      participants: [],
      idempotencyKey: "retro-key",
    })).rejects.toThrow("Invalid project retrospective response");
  });

  it("serializes the exact Retrospective route family and strict command bodies", async () => {
    const linked = VALID_RETROSPECTIVE_WIRE.action_links[0];
    const responses = [
      { retrospectives: [VALID_RETROSPECTIVE_WIRE], next_cursor: "opaque.cursor" },
      VALID_RETROSPECTIVE_WIRE,
      VALID_RETROSPECTIVE_WIRE,
      VALID_RETROSPECTIVE_WIRE,
      VALID_RETROSPECTIVE_WIRE,
      linked,
      linked,
    ];
    const fetchMock = vi.fn().mockImplementation(async () => new Response(
      JSON.stringify(responses.shift()),
      { status: 200, headers: { "Content-Type": "application/json" } },
    ));
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("http://localhost:3000");
    const content = {
      summary: "Delivery",
      successes: ["Small batches"],
      problems: [],
      lessons: ["Review sooner"],
      actionItems: [{ title: "Schedule review", assigneeId: "member-2", dueDate: "2026-08-10" }],
    };
    const participants = [{ memberId: "member-2", role: "facilitator" as const }];

    await client.listProjectRetrospectives("project-1", {
      limit: 25,
      cursor: "opaque.cursor",
      includeArchived: true,
    });
    await client.getProjectRetrospective("project-1", "retro-1");
    await client.createProjectRetrospective("project-1", {
      content,
      participants,
      idempotencyKey: "create-key",
    });
    await client.updateProjectRetrospective("project-1", "retro-1", {
      expectedRevision: 1,
      action: "save_draft",
      content: {
        ...content,
        actionItems: [{
          id: "action-1",
          title: "Schedule review",
          assigneeId: "member-2",
          dueDate: "2026-08-10",
        }],
      },
      participants,
    });
    await client.archiveProjectRetrospective("project-1", "retro-1", 2);
    await client.createProjectRetrospectiveTarget(
      "project-1", "retro-1", "action-1", { idempotencyKey: "task-key" },
    );
    await client.createProjectRetrospectiveTarget(
      "project-1", "retro-1", "action-1", { targetKind: "issue", idempotencyKey: "issue-key" },
    );

    expect(fetchMock.mock.calls.map((call) => String(call[0]))).toEqual([
      "http://localhost:3000/api/projects/project-1/retrospectives?limit=25&cursor=opaque.cursor&include_archived=true",
      "http://localhost:3000/api/projects/project-1/retrospectives/retro-1",
      "http://localhost:3000/api/projects/project-1/retrospectives",
      "http://localhost:3000/api/projects/project-1/retrospectives/retro-1",
      "http://localhost:3000/api/projects/project-1/retrospectives/retro-1?expected_revision=2",
      "http://localhost:3000/api/projects/project-1/retrospectives/retro-1/action-items/action-1/target",
      "http://localhost:3000/api/projects/project-1/retrospectives/retro-1/action-items/action-1/target",
    ]);
    const requests = fetchMock.mock.calls.map((call) => call[1] as RequestInit | undefined);
    expect(JSON.parse(requests[2]?.body as string)).toEqual({
      content: {
        summary: "Delivery",
        successes: ["Small batches"],
        problems: [],
        lessons: ["Review sooner"],
        action_items: [{
          title: "Schedule review",
          assignee_id: "member-2",
          due_date: "2026-08-10",
        }],
      },
      participants: [{ member_id: "member-2", role: "facilitator" }],
    });
    expect(new Headers(requests[2]?.headers).get("Idempotency-Key")).toBe("create-key");
    expect(JSON.parse(requests[3]?.body as string)).toMatchObject({
      expected_revision: 1,
      action: "save_draft",
      content: { action_items: [{ id: "action-1", title: "Schedule review" }] },
    });
    expect(JSON.parse(requests[5]?.body as string)).toEqual({});
    expect(JSON.parse(requests[6]?.body as string)).toEqual({ target_kind: "issue" });
    expect(new Headers(requests[5]?.headers).get("Idempotency-Key")).toBe("task-key");
    expect(new Headers(requests[6]?.headers).get("Idempotency-Key")).toBe("issue-key");
  });

  it("does not turn a malformed Retrospective list into an empty success", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ retrospectives: [{ id: "retro-1" }] }),
      { status: 200, headers: { "Content-Type": "application/json" } },
    )));
    const client = new ApiClient("http://localhost:3000");

    await expect(client.listProjectRetrospectives("project-1")).rejects.toThrow(
      "Invalid project retrospective list response",
    );
  });
});
