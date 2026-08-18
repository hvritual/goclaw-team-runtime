/**
 * @vitest-environment jsdom
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, renderHook } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { setApiInstance } from "../api";
import type { ApiClient } from "../api/client";
import type { PinnedItem } from "../types";
import { useReorderPins } from "./mutations";
import { pinKeys } from "./queries";

vi.mock("../hooks", () => ({ useWorkspaceId: () => "workspace-1" }));
vi.mock("../auth", () => ({
  useAuthStore: (selector: (state: { user: { id: string } }) => unknown) => selector({ user: { id: "user-1" } }),
}));

const pin = (id: string, position: number): PinnedItem => ({
  id,
  workspace_id: "workspace-1",
  user_id: "user-1",
  item_type: id === "pin-1" ? "issue" : "project",
  item_id: `item-${id}`,
  position,
  order_revision: 4,
  created_at: `2026-08-18T00:00:0${position}Z`,
});

function createWrapper(qc: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
  };
}

describe("useReorderPins", () => {
  let qc: QueryClient;
  let reorderPins: ReturnType<typeof vi.fn<(request: { items: { id: string }[]; expected_revision: number }) => Promise<void>>>;

  beforeEach(() => {
    qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    reorderPins = vi.fn();
    setApiInstance({ reorderPins } as unknown as ApiClient);
    qc.setQueryData(pinKeys.list("workspace-1", "user-1"), [pin("pin-1", 1), pin("pin-2", 2)]);
  });

  afterEach(() => {
    qc.clear();
    vi.restoreAllMocks();
  });

  it("sends the complete ID order and refetches after success", async () => {
    reorderPins.mockResolvedValue(undefined);
    const reordered = [pin("pin-2", 2), pin("pin-1", 1)];
    const { result } = renderHook(() => useReorderPins(), {
      wrapper: createWrapper(qc),
    });
    await act(async () => result.current.mutateAsync(reordered));
    expect(reorderPins).toHaveBeenCalledWith({
      items: [{ id: "pin-2" }, { id: "pin-1" }],
      expected_revision: 4,
    });
    expect(qc.getQueryData<PinnedItem[]>(pinKeys.list("workspace-1", "user-1"))?.map((value) => value.id)).toEqual(["pin-2", "pin-1"]);
    expect(qc.getQueryState(pinKeys.list("workspace-1", "user-1"))?.isInvalidated).toBe(true);
  });

  it("restores the exact previous order and refetches after a conflict", async () => {
    reorderPins.mockRejectedValue(new Error("revision conflict"));
    qc.setQueryData(pinKeys.list("workspace-2", "user-1"), [
      { ...pin("foreign", 1), workspace_id: "workspace-2" },
    ]);
    qc.setQueryData(pinKeys.list("workspace-1", "user-2"), [
      { ...pin("other-user", 1), user_id: "user-2" },
    ]);
    const { result } = renderHook(() => useReorderPins(), {
      wrapper: createWrapper(qc),
    });
    await act(async () => {
      await expect(result.current.mutateAsync([pin("pin-2", 2), pin("pin-1", 1)])).rejects.toThrow("revision conflict");
    });
    expect(qc.getQueryData<PinnedItem[]>(pinKeys.list("workspace-1", "user-1"))?.map((value) => value.id)).toEqual(["pin-1", "pin-2"]);
    expect(qc.getQueryState(pinKeys.list("workspace-1", "user-1"))?.isInvalidated).toBe(true);
    expect(qc.getQueryData<PinnedItem[]>(pinKeys.list("workspace-2", "user-1"))?.map((value) => value.id)).toEqual(["foreign"]);
    expect(qc.getQueryData<PinnedItem[]>(pinKeys.list("workspace-1", "user-2"))?.map((value) => value.id)).toEqual(["other-user"]);
  });

  it("removes an optimistic list when no previous cache existed", async () => {
    reorderPins.mockRejectedValue(new Error("offline"));
    qc.removeQueries({ queryKey: pinKeys.list("workspace-1", "user-1"), exact: true });
    const { result } = renderHook(() => useReorderPins(), { wrapper: createWrapper(qc) });
    await act(async () => {
      await expect(result.current.mutateAsync([pin("pin-2", 2), pin("pin-1", 1)])).rejects.toThrow("offline");
    });
    expect(qc.getQueryData(pinKeys.list("workspace-1", "user-1"))).toBeUndefined();
  });
});
