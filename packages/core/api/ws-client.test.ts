import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { WSClient } from "./ws-client";
import type { WSMessage } from "../types/events";

// Capture URL passed to WebSocket so we can assert the connect-time
// query string.  We don't simulate the full WS lifecycle here — only the
// upgrade URL construction, which is what carries client identity.
class FakeWebSocket {
  static lastUrl: string | null = null;
  static lastInstance: FakeWebSocket | null = null;
  // Fields read by WSClient.connect()/disconnect(), all no-op here.
  onopen: (() => void) | null = null;
  onmessage: ((ev: { data: string }) => void) | null = null;
  onclose: (() => void) | null = null;
  onerror: (() => void) | null = null;
  readyState = 0;
  constructor(url: string) {
    FakeWebSocket.lastUrl = url;
    FakeWebSocket.lastInstance = this;
  }
  close() {}
  send() {}
}

const canonicalIssueEvent = {
  id: "issue-1", workspace_id: "workspace-1", number: 1, identifier: "ONE-1",
  title: "Issue", description: null, status: "todo", priority: "none",
  assignee_type: null, assignee_id: null, creator_type: "member", creator_id: "member-1",
  parent_issue_id: null, project_id: null, position: 1, stage: null,
  start_date: null, due_date: null, metadata: {}, properties: {},
  created_at: "2026-08-13T00:00:00Z", updated_at: "2026-08-13T00:00:00Z",
};

describe("WSClient", () => {
  beforeEach(() => {
    FakeWebSocket.lastUrl = null;
    FakeWebSocket.lastInstance = null;
    vi.stubGlobal("WebSocket", FakeWebSocket as unknown as typeof WebSocket);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("includes client identity in the upgrade URL when configured", () => {
    const ws = new WSClient("ws://example.test/ws", {
      identity: { platform: "desktop", version: "1.2.3", os: "macos" },
    });
    ws.setAuth("tok", "acme");
    ws.connect();

    const url = new URL(FakeWebSocket.lastUrl!);
    expect(url.searchParams.get("workspace_slug")).toBe("acme");
    expect(url.searchParams.get("client_platform")).toBe("desktop");
    expect(url.searchParams.get("client_version")).toBe("1.2.3");
    expect(url.searchParams.get("client_os")).toBe("macos");
    // Token must never appear in the URL — it is delivered as the first
    // WS message in token mode.
    expect(url.searchParams.has("token")).toBe(false);
  });

  it("omits client_* params when identity is not configured", () => {
    const ws = new WSClient("ws://example.test/ws");
    ws.setAuth("tok", "acme");
    ws.connect();

    const url = new URL(FakeWebSocket.lastUrl!);
    expect(url.searchParams.has("client_platform")).toBe(false);
    expect(url.searchParams.has("client_version")).toBe(false);
    expect(url.searchParams.has("client_os")).toBe(false);
  });

  it("only includes the identity fields that are set", () => {
    const ws = new WSClient("ws://example.test/ws", {
      identity: { platform: "cli" },
    });
    ws.setAuth("tok", "acme");
    ws.connect();

    const url = new URL(FakeWebSocket.lastUrl!);
    expect(url.searchParams.get("client_platform")).toBe("cli");
    expect(url.searchParams.has("client_version")).toBe(false);
    expect(url.searchParams.has("client_os")).toBe(false);
  });

  it("truncates the logged payload when an unparseable frame is large", () => {
    const logger = {
      debug: vi.fn(),
      info: vi.fn(),
      warn: vi.fn(),
      error: vi.fn(),
    };
    const ws = new WSClient("ws://example.test/ws", { logger });
    ws.connect();

    const huge = "x".repeat(5000);
    FakeWebSocket.lastInstance!.onmessage?.({ data: huge });

    expect(logger.warn).toHaveBeenCalledTimes(1);
    const [, summary] = logger.warn.mock.calls[0] as [string, string];
    expect(summary.length).toBeLessThan(huge.length);
    expect(summary).toContain("truncated");
    expect(summary).toContain("5000");
    expect(summary.startsWith("x".repeat(200))).toBe(true);
  });

  it("logs and skips malformed frames without breaking later messages", () => {
    const logger = {
      debug: vi.fn(),
      info: vi.fn(),
      warn: vi.fn(),
      error: vi.fn(),
    };
    const ws = new WSClient("ws://example.test/ws", { logger });
    const handler = vi.fn();
    ws.on("issue:updated", handler);
    ws.connect();

    expect(() => {
      FakeWebSocket.lastInstance!.onmessage?.({ data: `{"type":"issue` });
    }).not.toThrow();

    FakeWebSocket.lastInstance!.onmessage?.({
      data: JSON.stringify({
        type: "issue:updated",
        payload: { issue: canonicalIssueEvent },
      }),
    });

    expect(logger.warn).toHaveBeenCalledWith(
      "ws: received unparseable message",
      `{"type":"issue`,
    );
    expect(handler).toHaveBeenCalledWith(
      { issue: canonicalIssueEvent },
      undefined,
      undefined,
    );
  });

  it("drops frames without a string type without throwing, and keeps dispatching", () => {
    // Regression for MUL-3418: a frame whose parsed JSON lacks a string `type`
    // (an out-of-protocol frame, or a bare JSON primitive) used to throw an
    // uncaught TypeError out of onmessage via `msg.type.split(...)` in a
    // downstream onAny handler, flooding `$exception` telemetry.
    const logger = {
      debug: vi.fn(),
      info: vi.fn(),
      warn: vi.fn(),
      error: vi.fn(),
    };
    const ws = new WSClient("ws://example.test/ws", { logger });

    // A downstream consumer that assumes a string type, exactly like the
    // realtime sync's onAny dispatcher.
    const anyHandler = vi.fn((msg: WSMessage) => msg.type.split(":")[0]);
    ws.onAny(anyHandler);
    const issueHandler = vi.fn();
    ws.on("issue:updated", issueHandler);
    ws.connect();

    const badFrames = [
      JSON.stringify({ payload: {} }), // object, no type
      "42", // bare number
      "true", // bare bool
      "[]", // array
    ];
    for (const data of badFrames) {
      expect(() => {
        FakeWebSocket.lastInstance!.onmessage?.({ data });
      }).not.toThrow();
    }

    // Bad frames never reached any handler.
    expect(anyHandler).not.toHaveBeenCalled();
    expect(issueHandler).not.toHaveBeenCalled();

    // A valid frame after the bad ones still dispatches normally.
    FakeWebSocket.lastInstance!.onmessage?.({
      data: JSON.stringify({ type: "issue:updated", payload: { issue: canonicalIssueEvent } }),
    });
    expect(issueHandler).toHaveBeenCalledWith({ issue: canonicalIssueEvent }, undefined, undefined);
    expect(anyHandler).toHaveBeenCalledTimes(1);

    // The drop is logged at most once per connection despite four bad frames.
    expect(logger.warn).toHaveBeenCalledTimes(1);
    expect(logger.warn.mock.calls[0]?.[0]).toBe(
      "ws: dropping frame without a string type",
    );
  });

  it("passes actor_id and actor_type to event handlers", () => {
    const ws = new WSClient("ws://example.test/ws");
    ws.setAuth("tok", "acme");
    ws.connect();

    const handler = vi.fn();
    ws.on("issue:created", handler);

    const fakeWs = (ws as any).ws as FakeWebSocket;
    fakeWs.onmessage?.({
      data: JSON.stringify({
        type: "issue:created",
        payload: { issue: canonicalIssueEvent },
        actor_id: "user-123",
        actor_type: "user",
      }),
    });

    expect(handler).toHaveBeenCalledWith(
      { issue: canonicalIssueEvent },
      "user-123",
      "user",
    );
  });

  it("drops malformed canonical Issue event payloads at the socket boundary", () => {
    const ws = new WSClient("ws://example.test/ws");
    const metadata = vi.fn();
    const updated = vi.fn();
    const created = vi.fn();
    ws.on("issue_metadata:changed", metadata);
    ws.on("issue:updated", updated);
    ws.on("issue:created", created);
    ws.connect();

    FakeWebSocket.lastInstance!.onmessage?.({
      data: JSON.stringify({
        type: "issue_metadata:changed",
        payload: { issue_id: "issue-1", metadata: { nested: { rejected: true } } },
      }),
    });
    FakeWebSocket.lastInstance!.onmessage?.({
      data: JSON.stringify({ type: "issue:updated", payload: { issue: { title: "missing identity" } } }),
    });
    FakeWebSocket.lastInstance!.onmessage?.({
      data: JSON.stringify({ type: "issue:created", payload: { id: "legacy-bare-id" } }),
    });
    expect(metadata).not.toHaveBeenCalled();
    expect(updated).not.toHaveBeenCalled();
    expect(created).not.toHaveBeenCalled();
  });

  it("accepts additive legacy fields while requiring the canonical Issue shape", () => {
    const ws = new WSClient("ws://example.test/ws");
    const created = vi.fn();
    const updated = vi.fn();
    ws.on("issue:created", created);
    ws.on("issue:updated", updated);
    ws.connect();

    const createdPayload = {
      issue: { ...canonicalIssueEvent, labels: [{ id: "label-1" }], attachments: [{ id: "attachment-1" }] },
      labels: [{ id: "label-1" }],
      attachments: [{ id: "attachment-1" }],
    };
    const updatedPayload = {
      issue: { ...canonicalIssueEvent, legacy_projection: "retained" },
      change: { field: "status", from: "todo", to: "done" },
      prev_issue: { ...canonicalIssueEvent, status: "todo" },
    };

    FakeWebSocket.lastInstance!.onmessage?.({
      data: JSON.stringify({ type: "issue:created", payload: createdPayload }),
    });
    FakeWebSocket.lastInstance!.onmessage?.({
      data: JSON.stringify({ type: "issue:updated", payload: updatedPayload }),
    });

    expect(created).toHaveBeenCalledWith(createdPayload, undefined, undefined);
    expect(updated).toHaveBeenCalledWith(updatedPayload, undefined, undefined);
  });

  it("drops canonical Issue events with invalid known enum and date values", () => {
    const ws = new WSClient("ws://example.test/ws");
    const updated = vi.fn();
    ws.on("issue:updated", updated);
    ws.connect();

    const invalidKnownFields = [
      { status: "owned" },
      { priority: "critical" },
      { assignee_type: "service" },
      { creator_type: "robot" },
      { start_date: "2026-02-30" },
      { due_date: "2026/08/13" },
      { created_at: "yesterday" },
      { updated_at: "2026-08-13" },
    ];
    for (const invalid of invalidKnownFields) {
      FakeWebSocket.lastInstance!.onmessage?.({
        data: JSON.stringify({
          type: "issue:updated",
          payload: { issue: { ...canonicalIssueEvent, ...invalid } },
        }),
      });
    }

    expect(updated).not.toHaveBeenCalled();
  });

  it("accepts agent assignees and creators in canonical Issue events", () => {
    const ws = new WSClient("ws://example.test/ws");
    const created = vi.fn();
    const updated = vi.fn();
    ws.on("issue:created", created);
    ws.on("issue:updated", updated);
    ws.connect();

    const agentIssue = {
      ...canonicalIssueEvent,
      assignee_type: "agent",
      assignee_id: "agent-1",
      creator_type: "agent",
      creator_id: "agent-1",
    };
    FakeWebSocket.lastInstance!.onmessage?.({
      data: JSON.stringify({ type: "issue:created", payload: { issue: agentIssue } }),
    });
    FakeWebSocket.lastInstance!.onmessage?.({
      data: JSON.stringify({ type: "issue:updated", payload: { issue: agentIssue } }),
    });

    expect(created).toHaveBeenCalledWith({ issue: agentIssue }, undefined, undefined);
    expect(updated).toHaveBeenCalledWith({ issue: agentIssue }, undefined, undefined);
  });

  it("validates collaboration event payloads before dispatch while retaining additive fields", () => {
    const ws = new WSClient("ws://example.test/ws");
    const timestamp = "2026-08-14T01:02:03Z";
    const comment = {
      id: "comment-1", issue_id: "issue-1", author_type: "member", author_id: "user-1",
      content: "Decision", type: "comment", parent_id: null, reactions: [], attachments: [],
      created_at: timestamp, updated_at: timestamp, resolved_at: null,
      resolved_by_type: null, resolved_by_id: null, retained_extension: true,
    };
    const validFrames = [
      ["comment:created", { comment }],
      ["comment:updated", { comment }],
      ["comment:resolved", { comment: { ...comment, resolved_at: timestamp, resolved_by_type: "member", resolved_by_id: "user-1" } }],
      ["comment:unresolved", { comment }],
      ["comment:deleted", { comment_id: "comment-1", issue_id: "issue-1" }],
      ["reaction:added", { reaction: { id: "reaction-1", comment_id: "comment-1", actor_type: "member", actor_id: "user-1", emoji: "x", created_at: timestamp }, issue_id: "issue-1" }],
      ["reaction:removed", { comment_id: "comment-1", issue_id: "issue-1", emoji: "x", actor_type: "member", actor_id: "user-1" }],
      ["issue_reaction:added", { reaction: { id: "issue-reaction-1", issue_id: "issue-1", actor_type: "agent", actor_id: "agent-1", emoji: "x", created_at: timestamp }, issue_id: "issue-1" }],
      ["issue_reaction:removed", { issue_id: "issue-1", emoji: "x", actor_type: "agent", actor_id: "agent-1" }],
      ["subscriber:added", { issue_id: "issue-1", user_type: "member", user_id: "user-1", reason: "manual" }],
      ["subscriber:removed", { issue_id: "issue-1", user_type: "member", user_id: "user-1" }],
      ["activity:created", { issue_id: "issue-1", entry: { type: "activity", id: "activity-1", actor_type: "member", actor_id: "user-1", action: "status_changed", details: { from: "todo", to: "done" }, created_at: timestamp } }],
    ] as const;
    const handlers = new Map(validFrames.map(([type]) => [type, vi.fn()]));
    for (const [type, handler] of handlers) ws.on(type, handler);
    ws.connect();
    for (const [type, payload] of validFrames) {
      FakeWebSocket.lastInstance!.onmessage?.({ data: JSON.stringify({ type, payload }) });
      expect(handlers.get(type)).toHaveBeenCalledWith(payload, undefined, undefined);
    }

    const invalidFrames = [
      ["comment:created", { comment: { ...comment, author_type: "service" } }],
      ["comment:deleted", { comment_id: "comment-1" }],
      ["reaction:added", { reaction: { id: "reaction-1" }, issue_id: "issue-1" }],
      ["subscriber:added", { issue_id: "issue-1", user_type: "agent", user_id: "agent-1", reason: "manual" }],
      ["activity:created", { issue_id: "issue-1", entry: { type: "activity", id: "activity-1" } }],
    ] as const;
    for (const [type, payload] of invalidFrames) {
      const handler = handlers.get(type)!;
      handler.mockClear();
      FakeWebSocket.lastInstance!.onmessage?.({ data: JSON.stringify({ type, payload }) });
      expect(handler).not.toHaveBeenCalled();
    }
  });

  it("validates label and property event payloads before catalog cache consumers run", () => {
    const ws = new WSClient("ws://example.test/ws");
    const timestamp = "2026-08-15T01:02:03Z";
    const label = {
      id: "label-1", workspace_id: "workspace-1", resource_type: "issue",
      name: "Priority", description: "", color: "#112233", usage_count: 1,
      created_at: timestamp, updated_at: timestamp, retained_extension: true,
    };
    const property = {
      id: "property-1", workspace_id: "workspace-1", name: "Estimate", type: "number",
      description: "", icon: "hash", config: { options: [] }, position: 0,
      archived: false, archived_at: null, usage_count: 1,
      created_at: timestamp, updated_at: timestamp, retained_extension: true,
    };
    const validFrames = [
      ["label:created", { label }],
      ["label:updated", { label }],
      ["label:deleted", { label_id: "label-1" }],
      ["issue_labels:changed", { issue_id: "issue-1", labels: [label] }],
      ["property:created", { property }],
      ["property:updated", { property }],
      ["issue_properties:changed", { issue_id: "issue-1", properties: { "property-1": 2.5 } }],
    ] as const;
    const handlers = new Map(validFrames.map(([type]) => [type, vi.fn()]));
    for (const [type, handler] of handlers) ws.on(type, handler);
    ws.connect();

    for (const [type, payload] of validFrames) {
      FakeWebSocket.lastInstance!.onmessage?.({ data: JSON.stringify({ type, payload }) });
      expect(handlers.get(type)).toHaveBeenCalledWith(payload, undefined, undefined);
    }

    const legacySkillUpdate = { label_id: "label-skill", resource_type: "skill" };
    const labelUpdatedHandler = handlers.get("label:updated")!;
    labelUpdatedHandler.mockClear();
    FakeWebSocket.lastInstance!.onmessage?.({ data: JSON.stringify({ type: "label:updated", payload: legacySkillUpdate }) });
    expect(labelUpdatedHandler).toHaveBeenCalledWith(legacySkillUpdate, undefined, undefined);

    const legacyPropertyUpdate = { property: { ...property, config: {} } };
    const propertyUpdatedHandler = handlers.get("property:updated")!;
    propertyUpdatedHandler.mockClear();
    FakeWebSocket.lastInstance!.onmessage?.({ data: JSON.stringify({ type: "property:updated", payload: legacyPropertyUpdate }) });
    expect(propertyUpdatedHandler).toHaveBeenCalledWith(legacyPropertyUpdate, undefined, undefined);

    const invalidFrames = [
      ["label:created", { label: { ...label, id: 42 } }],
      ["label:updated", { label: { ...label, color: null } }],
      ["label:deleted", { label_id: "" }],
      ["issue_labels:changed", { issue_id: "issue-1", labels: [{ id: "label-1" }] }],
      ["property:created", { property: { ...property, type: null } }],
      ["property:updated", { property: { ...property, config: { options: [{ id: 1 }] } } }],
      ["issue_properties:changed", { issue_id: "issue-1", properties: { "property-1": { nested: true } } }],
    ] as const;
    for (const [type, payload] of invalidFrames) {
      const handler = handlers.get(type)!;
      handler.mockClear();
      FakeWebSocket.lastInstance!.onmessage?.({ data: JSON.stringify({ type, payload }) });
      expect(handler).not.toHaveBeenCalled();
    }
  });

  // ── Reconnect backoff tests ────────────────────────────────────────

  describe("reconnect backoff", () => {
    let setTimeoutSpy: ReturnType<typeof vi.spyOn>;

    beforeEach(() => {
      vi.useFakeTimers();
      setTimeoutSpy = vi.spyOn(globalThis, "setTimeout");
    });

    afterEach(() => {
      setTimeoutSpy.mockRestore();
      vi.useRealTimers();
    });

    /** Return the delay (ms) passed to the most recent setTimeout call. */
    function lastTimerDelay(): number {
      const calls = setTimeoutSpy.mock.calls;
      return calls[calls.length - 1]?.[1] as number;
    }

    function simulateDisconnect() {
      FakeWebSocket.lastInstance!.onclose?.();
    }

    /** Simulate the server acknowledging the auth token. In token mode the
     *  client sends `{type:"auth"}` on open, and the server replies with
     *  `{type:"auth_ack"}`. Only then does `onAuthenticated()` fire and
     *  reset the reconnect counter. */
    function simulateAuthAck() {
      FakeWebSocket.lastInstance!.onmessage?.({
        data: JSON.stringify({ type: "auth_ack" }),
      });
    }

    it("uses ~1000ms base delay for the first reconnect attempt", () => {
      // Pin Math.random so jitter is deterministic: random()=0.5 → jitter=0.
      vi.stubGlobal(
        "Math",
        new Proxy(Math, {
          get(target, prop) {
            if (prop === "random") return () => 0.5;
            return (target as any)[prop];
          },
        }),
      );

      const ws = new WSClient("ws://example.test/ws");
      ws.connect();
      simulateDisconnect();

      expect(lastTimerDelay()).toBe(1000);
    });

    it("doubles the base delay on consecutive failures (exponential)", () => {
      vi.stubGlobal(
        "Math",
        new Proxy(Math, {
          get(target, prop) {
            if (prop === "random") return () => 0.5;
            return (target as any)[prop];
          },
        }),
      );

      const ws = new WSClient("ws://example.test/ws");
      ws.connect();

      // Attempt 1: base = 1000 * 2^0 = 1000
      simulateDisconnect();
      expect(lastTimerDelay()).toBe(1000);

      // Fire the timer → connect() → new FakeWebSocket → simulate disconnect
      vi.advanceTimersByTime(1000);
      // Attempt 2: base = 1000 * 2^1 = 2000
      simulateDisconnect();
      expect(lastTimerDelay()).toBe(2000);

      vi.advanceTimersByTime(2000);
      // Attempt 3: base = 1000 * 2^2 = 4000
      simulateDisconnect();
      expect(lastTimerDelay()).toBe(4000);

      vi.advanceTimersByTime(4000);
      // Attempt 4: base = 1000 * 2^3 = 8000
      simulateDisconnect();
      expect(lastTimerDelay()).toBe(8000);
    });

    it("caps the delay at 30s even after many failures", () => {
      vi.stubGlobal(
        "Math",
        new Proxy(Math, {
          get(target, prop) {
            if (prop === "random") return () => 0.5;
            return (target as any)[prop];
          },
        }),
      );

      const ws = new WSClient("ws://example.test/ws");
      ws.connect();

      // Drive through enough failures to exceed the cap:
      // 2^5 = 32000 > 30000, so attempt 6 should be capped.
      const delays = [1000, 2000, 4000, 8000, 16000];
      for (const d of delays) {
        simulateDisconnect();
        expect(lastTimerDelay()).toBe(d);
        vi.advanceTimersByTime(d);
      }

      // Attempt 6: 1000 * 2^5 = 32000 → capped to 30000
      simulateDisconnect();
      expect(lastTimerDelay()).toBe(30000);
    });

    it("applies jitter so delays vary with Math.random", () => {
      // Stub Math.random to alternate between 0 (jitter = -20%) and
      // 1 (jitter = +20%), producing deterministic min/max delays.
      let callCount = 0;
      vi.stubGlobal(
        "Math",
        new Proxy(Math, {
          get(target, prop) {
            if (prop === "random") return () => (callCount++ % 2 === 0 ? 0 : 1);
            return (target as any)[prop];
          },
        }),
      );

      const ws = new WSClient("ws://example.test/ws");

      // First disconnect: random()=0 → jitter = 1000 * 0.2 * (0*2-1) = -200 → 800ms
      ws.connect();
      simulateDisconnect();
      const delay1 = lastTimerDelay();

      // Second disconnect: random()=1 → jitter = 1000 * 0.2 * (1*2-1) = +200 → 1200ms
      vi.clearAllTimers();
      ws.disconnect();
      ws.connect();
      simulateDisconnect();
      const delay2 = lastTimerDelay();

      // The two delays must differ and fall within [800, 1200].
      expect(delay1).toBe(800);
      expect(delay2).toBe(1200);
    });

    it("resets the attempt counter on successful authentication", () => {
      vi.stubGlobal(
        "Math",
        new Proxy(Math, {
          get(target, prop) {
            if (prop === "random") return () => 0.5;
            return (target as any)[prop];
          },
        }),
      );

      const ws = new WSClient("ws://example.test/ws");
      ws.setAuth("tok", "acme");
      ws.connect();

      // Two failures: delays 1000, 2000
      simulateDisconnect();
      expect(lastTimerDelay()).toBe(1000);
      vi.advanceTimersByTime(1000);

      simulateDisconnect();
      expect(lastTimerDelay()).toBe(2000);
      vi.advanceTimersByTime(2000);

      // Successful connection resets the counter.
      simulateAuthAck();

      // Next failure should be back to the base delay.
      simulateDisconnect();
      expect(lastTimerDelay()).toBe(1000);
    });

    it("keeps retrying indefinitely with capped delay", () => {
      vi.stubGlobal(
        "Math",
        new Proxy(Math, {
          get(target, prop) {
            if (prop === "random") return () => 0.5;
            return (target as any)[prop];
          },
        }),
      );

      const logger = {
        debug: vi.fn(),
        info: vi.fn(),
        warn: vi.fn(),
        error: vi.fn(),
      };
      const ws = new WSClient("ws://example.test/ws", { logger });
      ws.connect();

      // Drive past the old 20-attempt limit — all should schedule a reconnect.
      for (let i = 0; i < 25; i++) {
        simulateDisconnect();
        const delay = lastTimerDelay();
        // Once past attempt 5 (base > 30000), delay should be capped at 30s.
        if (i >= 5) {
          expect(delay).toBe(30000);
        }
        vi.advanceTimersByTime(delay);
      }

      // The 26th disconnect should STILL schedule a reconnect (no give-up).
      const timerCountBefore = setTimeoutSpy.mock.calls.length;
      simulateDisconnect();
      expect(setTimeoutSpy.mock.calls.length).toBe(timerCountBefore + 1);
      expect(lastTimerDelay()).toBe(30000);
      // No "giving up" error should have been logged.
      expect(logger.error).not.toHaveBeenCalled();
    });

    it("disconnect() cancels a pending reconnect and resets the counter", () => {
      vi.stubGlobal(
        "Math",
        new Proxy(Math, {
          get(target, prop) {
            if (prop === "random") return () => 0.5;
            return (target as any)[prop];
          },
        }),
      );

      const ws = new WSClient("ws://example.test/ws");
      ws.connect();

      // Two failures to bump the counter.
      simulateDisconnect();
      vi.advanceTimersByTime(1000);
      simulateDisconnect();
      // A reconnect timer is now pending.

      // Explicit disconnect should cancel the pending timer.
      ws.disconnect();
      vi.advanceTimersByTime(10_000);
      // No new WebSocket should have been created after the disconnect.
      // The last URL should still be from the second connect() call.

      // A fresh connect() after disconnect should start from attempt 0.
      ws.connect();
      simulateDisconnect();
      expect(lastTimerDelay()).toBe(1000);
    });
  });
});
