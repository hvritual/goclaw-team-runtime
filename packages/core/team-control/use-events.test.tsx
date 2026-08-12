/**
 * @vitest-environment jsdom
 */
import { act, renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "../api";
import { teamControlKeys } from "./queries";

const { streamMock } = vi.hoisted(() => ({ streamMock: vi.fn() }));

vi.mock("./client", async (importOriginal) => {
  const original = await importOriginal<typeof import("./client")>();
  return { ...original, streamTeamControlEvents: streamMock };
});

import { useTeamControlEvents } from "./use-events";

function wrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

describe("useTeamControlEvents", () => {
  beforeEach(() => {
    streamMock.mockReset();
  });

  it("invalidates only the matching project projection", async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const invalidate = vi.spyOn(queryClient, "invalidateQueries");
    streamMock.mockImplementation(async (_workspaceId, _projectId, options) => {
      options.onOpen?.();
      options.onEvent({ workspace_id: "ws-1", project_id: "project-1", sequence: 1 });
      options.onEvent({ workspace_id: "ws-1", project_id: "other-project", sequence: 2 });
      await new Promise<void>((resolve) => options.signal.addEventListener("abort", () => resolve(), { once: true }));
      return 2;
    });

    const { result, unmount } = renderHook(
      () => useTeamControlEvents("ws-1", "project-1"),
      { wrapper: wrapper(queryClient) },
    );

    await waitFor(() => expect(result.current).toBe("connected"));
    expect(invalidate).toHaveBeenCalledWith({
      queryKey: teamControlKeys.projection("ws-1", "project-1"),
    });
    expect(invalidate).not.toHaveBeenCalledWith({
      queryKey: teamControlKeys.projection("ws-1", "other-project"),
    });
    act(() => unmount());
  });

  it("preserves the last observed cursor when a stream fails", async () => {
    vi.useFakeTimers();
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    streamMock
      .mockImplementationOnce(async (_workspaceId, _projectId, options) => {
        options.onCursor?.(7);
        throw new Error("connection reset");
      })
      .mockImplementationOnce(async (_workspaceId, _projectId, options) => {
        expect(options.after).toBe(7);
        await new Promise<void>((resolve) => options.signal.addEventListener("abort", () => resolve(), { once: true }));
        return 7;
      });

    const { unmount } = renderHook(
      () => useTeamControlEvents("ws-1", "project-1", { initialCursor: 5 }),
      { wrapper: wrapper(queryClient) },
    );
    await act(async () => { await vi.advanceTimersByTimeAsync(2_000); });
    expect(streamMock).toHaveBeenCalledTimes(2);
    unmount();
    vi.useRealTimers();
  });

  it("does not connect while disabled or retry terminal authorization failures", async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const disabled = renderHook(
      () => useTeamControlEvents("ws-1", "project-1", { enabled: false }),
      { wrapper: wrapper(queryClient) },
    );
    expect(streamMock).not.toHaveBeenCalled();
    disabled.unmount();

    streamMock.mockRejectedValue(new ApiError("denied", 403, "Forbidden"));
    const terminal = renderHook(
      () => useTeamControlEvents("ws-1", "project-1", { enabled: true }),
      { wrapper: wrapper(queryClient) },
    );
    await waitFor(() => expect(terminal.result.current).toBe("offline"));
    expect(streamMock).toHaveBeenCalledTimes(1);
    terminal.unmount();
  });
});
