/**
 * @vitest-environment jsdom
 */
import { act, renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
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
});
