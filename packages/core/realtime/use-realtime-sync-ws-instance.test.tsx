/**
 * @vitest-environment jsdom
 */
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { WSClient } from "../api/ws-client";
import type { RealtimeSyncStores } from "./use-realtime-sync";
import { useRealtimeSync } from "./use-realtime-sync";

vi.mock("../platform/workspace-storage", () => ({
  getCurrentWsId: () => "ws-1",
}));

type AnyHandler = (message: { type: string; payload: unknown }) => void;

function createMockWs() {
  let onAnyHandler: AnyHandler | undefined;
  let reconnectHandler: (() => void) | undefined;
  const ws = {
    onAny: vi.fn((handler: AnyHandler) => {
      onAnyHandler = handler;
      return () => {};
    }),
    onReconnect: vi.fn((handler: () => void) => {
      reconnectHandler = handler;
      return () => {};
    }),
  } as unknown as WSClient;
  return {
    ws,
    emit: (type: string) => onAnyHandler?.({ type, payload: {} }),
    reconnect: () => reconnectHandler?.(),
  };
}

function createStores(): RealtimeSyncStores {
  return {
    authStore: Object.assign(() => ({}), {
      getState: () => ({ user: { id: "u1" } }),
      subscribe: () => () => {},
      setState: () => {},
      destroy: () => {},
    }),
  } as unknown as RealtimeSyncStores;
}

function createWrapper(qc: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
  };
}

describe("useRealtimeSync", () => {
  let qc: QueryClient;
  let stores: RealtimeSyncStores;

  beforeEach(() => {
    vi.useFakeTimers();
    qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    stores = createStores();
  });

  afterEach(() => {
    vi.useRealTimers();
    qc.clear();
  });

  it("does not invalidate on the first socket mount or while disconnected", () => {
    const first = createMockWs();
    const invalidate = vi.spyOn(qc, "invalidateQueries");
    const { rerender } = renderHook(
      ({ ws }) => useRealtimeSync(ws, stores),
      {
        initialProps: { ws: first.ws as WSClient | null },
        wrapper: createWrapper(qc),
      },
    );

    expect(invalidate).not.toHaveBeenCalled();
    rerender({ ws: null });
    expect(invalidate).not.toHaveBeenCalled();
  });

  it("refreshes workspace and per-issue detail caches when the socket is replaced", () => {
    const first = createMockWs();
    const invalidate = vi.spyOn(qc, "invalidateQueries");
    const { rerender } = renderHook(
      ({ ws }) => useRealtimeSync(ws, stores),
      {
        initialProps: { ws: first.ws as WSClient | null },
        wrapper: createWrapper(qc),
      },
    );

    rerender({ ws: null });
    invalidate.mockClear();
    rerender({ ws: createMockWs().ws });

    const keys = invalidate.mock.calls.map(([options]) => options?.queryKey);
    expect(keys).toContainEqual(["workspaces", "ws-1"]);
    expect(keys).toContainEqual(["issues", "ws-1"]);
    expect(keys).toContainEqual(["projects", "ws-1"]);
    expect(keys).toContainEqual(["tasks", "ws-1"]);
    expect(keys).toContainEqual(["workspaces", "ws-1", "skills"]);
    expect(keys).toContainEqual(["issues", "timeline"]);
    expect(keys).toContainEqual(["issues", "reactions"]);
    expect(keys).toContainEqual(["issues", "subscribers"]);
    expect(keys).toContainEqual(["issues", "attachments"]);
  });

  it("debounces retained-domain events into one workspace refresh", () => {
    const socket = createMockWs();
    const invalidate = vi.spyOn(qc, "invalidateQueries");
    renderHook(() => useRealtimeSync(socket.ws, stores), {
      wrapper: createWrapper(qc),
    });

    socket.emit("task:updated");
    socket.emit("issue:updated");
    expect(invalidate).not.toHaveBeenCalled();

    act(() => vi.advanceTimersByTime(75));
    expect(invalidate).toHaveBeenCalledTimes(12);
  });

  it("tolerates duplicate committed events with one authoritative refresh", () => {
    const socket = createMockWs();
    const invalidate = vi.spyOn(qc, "invalidateQueries");
    renderHook(() => useRealtimeSync(socket.ws, stores), { wrapper: createWrapper(qc) });

    socket.emit("issue_metadata:changed");
    socket.emit("issue_metadata:changed");
    act(() => vi.advanceTimersByTime(75));

    expect(invalidate).toHaveBeenCalledTimes(12);
  });

  it("ignores events outside the retained domains", () => {
    const socket = createMockWs();
    const invalidate = vi.spyOn(qc, "invalidateQueries");
    renderHook(() => useRealtimeSync(socket.ws, stores), {
      wrapper: createWrapper(qc),
    });

    socket.emit("unknown:updated");
    act(() => vi.advanceTimersByTime(100));
    expect(invalidate).not.toHaveBeenCalled();
  });

  it("refreshes immediately after a reconnect", () => {
    const socket = createMockWs();
    const invalidate = vi.spyOn(qc, "invalidateQueries");
    renderHook(() => useRealtimeSync(socket.ws, stores), {
      wrapper: createWrapper(qc),
    });

    socket.reconnect();
    expect(invalidate).toHaveBeenCalledTimes(12);
  });
});
