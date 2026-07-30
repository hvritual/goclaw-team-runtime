/**
 * @vitest-environment jsdom
 */
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { WSClient } from "../api/ws-client";
import { issueKeys, type IssueSortParam } from "../issues/queries";
import type { ListIssuesCache } from "../types";
import { useRealtimeSync, type RealtimeSyncStores } from "./use-realtime-sync";

vi.mock("../platform/workspace-storage", () => ({
  getCurrentWsId: () => "ws-1",
}));

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

const updatedSort: IssueSortParam = {
  sort_by: "updated_at",
  sort_direction: "desc",
};
const updatedBoardKey = issueKeys.listSorted("ws-1", updatedSort);

describe("useRealtimeSync — comment events", () => {
  let qc: QueryClient;

  beforeEach(() => {
    vi.useFakeTimers();
    qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  });

  afterEach(() => {
    vi.useRealTimers();
    qc.clear();
  });

  it("invalidates workspace issue data after a comment change", () => {
    const cached: ListIssuesCache = {
      byStatus: { todo: { issues: [], total: 1 } },
    };
    qc.setQueryData(updatedBoardKey, cached);

    let onAny: ((message: { type: string }) => void) | undefined;
    const ws = {
      onAny: vi.fn((handler) => {
        onAny = handler;
        return () => {};
      }),
      onReconnect: vi.fn(() => () => {}),
    } as unknown as WSClient;

    renderHook(() => useRealtimeSync(ws, createStores()), {
      wrapper: createWrapper(qc),
    });

    onAny?.({ type: "comment:created" });
    act(() => vi.advanceTimersByTime(75));

    expect(qc.getQueryState(updatedBoardKey)?.isInvalidated).toBe(true);
  });

  it("ignores unrelated event prefixes", () => {
    qc.setQueryData(updatedBoardKey, {
      byStatus: { todo: { issues: [], total: 1 } },
    } satisfies ListIssuesCache);

    let onAny: ((message: { type: string }) => void) | undefined;
    const ws = {
      onAny: vi.fn((handler) => {
        onAny = handler;
        return () => {};
      }),
      onReconnect: vi.fn(() => () => {}),
    } as unknown as WSClient;

    renderHook(() => useRealtimeSync(ws, createStores()), {
      wrapper: createWrapper(qc),
    });

    onAny?.({ type: "unknown:created" });
    act(() => vi.advanceTimersByTime(100));

    expect(qc.getQueryState(updatedBoardKey)?.isInvalidated).toBe(false);
  });
});
