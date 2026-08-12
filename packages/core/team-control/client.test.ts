/**
 * @vitest-environment jsdom
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiClient, setApiInstance } from "../api";
import {
  executeTeamControlCommand,
  getTeamControlProjection,
  parseSSEFrame,
  streamTeamControlEvents,
} from "./client";

const hash = "a".repeat(64);

afterEach(() => {
  vi.unstubAllGlobals();
  document.cookie = "multica_csrf=; Max-Age=0; path=/";
});

describe("Team Control authenticated client", () => {
  it("uses the existing bearer, cookie, and CSRF transport", async () => {
    document.cookie = "multica_csrf=csrf-value; path=/";
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      schema_version: 1,
      workspace_id: "ws-1",
      project_id: "project-1",
      head: 0,
      head_hash: "",
      nodes: {},
      edges: {},
      evidence: {},
      checks: {},
      acceptances: {},
    }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("https://api.example.test");
    client.setToken("desktop-token");
    setApiInstance(client);

    await getTeamControlProjection("ws-1", "project-1");

    expect(fetchMock).toHaveBeenCalledWith(
      "https://api.example.test/control-plane/v1/workspaces/ws-1/projects/project-1/projection",
      expect.objectContaining({
        credentials: "include",
        headers: expect.objectContaining({
          Authorization: "Bearer desktop-token",
          "X-CSRF-Token": "csrf-value",
        }),
      }),
    );
  });

  it("degrades a malformed projection to a scoped empty projection", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ nodes: "wrong" }), { status: 200 }),
    ));
    setApiInstance(new ApiClient("https://api.example.test"));

    await expect(getTeamControlProjection("ws-1", "project-1")).resolves.toMatchObject({
      schema_version: 1,
      workspace_id: "ws-1",
      project_id: "project-1",
      head: 0,
      nodes: {},
    });
  });

  it("rejects a malformed command success instead of inventing state", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ head: "wrong" }), { status: 201 }),
    ));
    setApiInstance(new ApiClient("https://api.example.test"));

    await expect(executeTeamControlCommand("ws-1", "project-1", {
      type: "requirement.start",
      expectedHead: 0,
      commandId: "command-1",
      payload: { id: "requirement-1", text: "Keep the audit trail" },
    })).rejects.toThrow("Malformed Team Control response");
  });

  it("parses session frames and ignores malformed or unrelated event frames", () => {
    const event = {
      schema_version: 1,
      workspace_id: "ws-1",
      project_id: "project-1",
      sequence: 3,
      event_id: "event-1",
      command_id: "command-1",
      type: "work.node.upserted.v1",
      actor_id: "member-1",
      actor_kind: "human",
      payload: {},
      previous_hash: hash,
      hash,
      occurred_at: "2026-08-12T00:00:00Z",
    };
    expect(parseSSEFrame(`id: 3\nevent: session\ndata: ${JSON.stringify(event)}`)?.sequence).toBe(3);
    expect(parseSSEFrame("event: ping\ndata: {}")) .toBeNull();
    expect(parseSSEFrame("event: session\ndata: not-json")).toBeNull();
  });

  it("resumes the authenticated stream and emits only valid session events", async () => {
    const event = {
      schema_version: 1,
      workspace_id: "ws-1",
      project_id: "project-1",
      sequence: 4,
      event_id: "event-4",
      command_id: "command-4",
      type: "work.node.upserted.v1",
      actor_id: "member-1",
      actor_kind: "human",
      payload: {},
      previous_hash: hash,
      hash,
      occurred_at: "2026-08-12T00:00:00Z",
    };
    const stream = new ReadableStream({
      start(controller) {
        controller.enqueue(new TextEncoder().encode(`: connected\n\nid: 4\nevent: session\ndata: ${JSON.stringify(event)}\n\n`));
        controller.close();
      },
    });
    const fetchMock = vi.fn().mockResolvedValue(new Response(stream, { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("https://api.example.test");
    client.setToken("desktop-token");
    setApiInstance(client);
    const received: number[] = [];

    const cursor = await streamTeamControlEvents("ws-1", "project-1", {
      after: 3,
      signal: new AbortController().signal,
      onEvent: (value) => received.push(value.sequence),
    });

    expect(cursor).toBe(4);
    expect(received).toEqual([4]);
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining("/events"),
      expect.objectContaining({
        headers: expect.objectContaining({ "Last-Event-ID": "3" }),
      }),
    );
  });
});
