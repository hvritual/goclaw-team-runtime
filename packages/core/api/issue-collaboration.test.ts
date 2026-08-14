import { afterEach, describe, expect, it, vi } from "vitest";
import { setCurrentWorkspace } from "../platform/workspace-storage";
import { ApiClient } from "./client";

const timestamp = "2026-08-14T01:02:03Z";
const comment = {
  id: "comment-1",
  issue_id: "issue-1",
  author_type: "member",
  author_id: "user-1",
  content: "Decision",
  type: "comment",
  parent_id: null,
  reactions: [],
  attachments: [],
  created_at: timestamp,
  updated_at: timestamp,
  resolved_at: null,
  resolved_by_type: null,
  resolved_by_id: null,
};

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

describe("Issue collaboration API boundary", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    setCurrentWorkspace(null, null);
  });

  it("accepts the exact collaboration response families and dedicated reactions GET", async () => {
    const issueReaction = {
      id: "issue-reaction-1",
      issue_id: "issue-1",
      actor_type: "member",
      actor_id: "user-1",
      emoji: "thumbs-up",
      created_at: timestamp,
    };
    const subscriber = {
      issue_id: "issue-1",
      user_type: "member",
      user_id: "user-1",
      reason: "manual",
      created_at: timestamp,
    };
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(jsonResponse([comment]))
      .mockResolvedValueOnce(jsonResponse([{ type: "comment", id: comment.id, actor_type: comment.author_type, actor_id: comment.author_id, created_at: timestamp, content: comment.content, parent_id: null, updated_at: timestamp, comment_type: "comment", reactions: [], attachments: [], resolved_at: null, resolved_by_type: null, resolved_by_id: null }]))
      .mockResolvedValueOnce(jsonResponse([issueReaction]))
      .mockResolvedValueOnce(jsonResponse([subscriber]))
      .mockResolvedValueOnce(jsonResponse({ subscribed: true }));
    vi.stubGlobal("fetch", fetchMock);
    setCurrentWorkspace("acme", "workspace-1");
    const client = new ApiClient("http://localhost:3000");

    await expect(client.listComments("issue-1")).resolves.toEqual([comment]);
    await expect(client.listTimeline("issue-1")).resolves.toHaveLength(1);
    await expect(client.listIssueReactions("issue-1")).resolves.toEqual([issueReaction]);
    await expect(client.listIssueSubscribers("issue-1")).resolves.toEqual([subscriber]);
    await expect(client.subscribeToIssue("issue-1")).resolves.toBeUndefined();

    expect(fetchMock.mock.calls[2]?.[0]).toBe("http://localhost:3000/api/issues/issue-1/reactions");
    expect(JSON.parse((fetchMock.mock.calls[4]?.[1] as RequestInit).body as string)).toEqual({});
  });

  it.each([
    ["comments", (client: ApiClient) => client.listComments("issue-1"), [{ ...comment, resolved_at: undefined }]],
    ["timeline", (client: ApiClient) => client.listTimeline("issue-1"), [{ type: "activity", id: "activity-1", actor_type: "member", actor_id: "user-1", created_at: timestamp, action: "status_changed" }]],
    ["Issue reactions", (client: ApiClient) => client.listIssueReactions("issue-1"), [{ id: "reaction-1", issue_id: "issue-1", actor_type: "service", actor_id: "service-1", emoji: "x", created_at: timestamp }]],
    ["subscribers", (client: ApiClient) => client.listIssueSubscribers("issue-1"), [{ issue_id: "issue-1", user_type: "agent", user_id: "agent-1", reason: "manual", created_at: timestamp }]],
  ])("rejects malformed %s instead of fabricating an empty success", async (_name, call, payload) => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse(payload)));
    await expect(call(new ApiClient("http://localhost:3000"))).rejects.toThrow(/Invalid .* response/);
  });

  it("validates mutation responses before returning them", async () => {
    vi.stubGlobal("fetch", vi.fn()
      .mockResolvedValueOnce(jsonResponse({ ...comment, id: "" }, 201))
      .mockResolvedValueOnce(jsonResponse({ id: "reaction-1", comment_id: "comment-1", actor_type: "service", actor_id: "service-1", emoji: "x", created_at: timestamp }, 201))
      .mockResolvedValueOnce(jsonResponse({ subscribed: "yes" })));
    const client = new ApiClient("http://localhost:3000");

    await expect(client.createComment("issue-1", "Decision")).rejects.toThrow("Invalid comment response");
    await expect(client.addReaction("comment-1", "x")).rejects.toThrow("Invalid comment reaction response");
    await expect(client.subscribeToIssue("issue-1")).rejects.toThrow("Invalid subscription response");
  });
});
