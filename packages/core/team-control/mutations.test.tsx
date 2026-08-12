/**
 * @vitest-environment jsdom
 */
import { act, renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiClient, setApiInstance } from "../api";
import { teamControlKeys } from "./queries";
import { useTeamControlCommand } from "./mutations";

function wrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

describe("Team Control command mutation", () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  });

  it("does not optimistically advance governed state", async () => {
    let resolveRequest: ((response: Response) => void) | undefined;
    vi.stubGlobal("fetch", vi.fn().mockImplementation(() => new Promise<Response>((resolve) => {
      resolveRequest = resolve;
    })));
    setApiInstance(new ApiClient("https://api.example.test"));
    const key = teamControlKeys.projection("ws-1", "project-1");
    queryClient.setQueryData(key, { head: 7 });
    const { result } = renderHook(
      () => useTeamControlCommand("ws-1", "project-1"),
      { wrapper: wrapper(queryClient) },
    );

    let request: Promise<unknown> | undefined;
    act(() => {
      request = result.current.mutateAsync({
        type: "requirement.start",
        expectedHead: 7,
        payload: { id: "requirement-1", text: "Need" },
      });
    });

    await waitFor(() => expect(resolveRequest).toBeTypeOf("function"));
    expect(queryClient.getQueryData(key)).toEqual({ head: 7 });
    resolveRequest?.(new Response(JSON.stringify({
      events: [], head: 8, head_hash: "a".repeat(64), replayed: false,
    }), { status: 201 }));
    await act(async () => { await request; });
  });

  it("invalidates the exact projection on a CAS conflict", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({
      type: "about:blank",
      title: "conflict",
      status: 409,
      code: "conflict",
      detail: "the authoritative state changed; refresh and retry",
    }), { status: 409, statusText: "Conflict" })));
    setApiInstance(new ApiClient("https://api.example.test"));
    const invalidate = vi.spyOn(queryClient, "invalidateQueries");
    const { result } = renderHook(
      () => useTeamControlCommand("ws-1", "project-1"),
      { wrapper: wrapper(queryClient) },
    );

    await act(async () => {
      await result.current.mutateAsync({
        type: "requirement.start",
        expectedHead: 3,
        commandId: "conflicting-command",
        payload: { id: "requirement-1", text: "Need" },
      }).catch(() => undefined);
    });

    expect(invalidate).toHaveBeenCalledWith({
      queryKey: teamControlKeys.projection("ws-1", "project-1"),
    });
    expect(invalidate).not.toHaveBeenCalledWith({
      queryKey: teamControlKeys.projection("ws-1", "project-2"),
    });
  });
});
