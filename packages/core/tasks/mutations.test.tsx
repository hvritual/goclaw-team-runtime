/**
 * @vitest-environment jsdom
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, renderHook } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { setApiInstance } from "../api";
import type { ApiClient } from "../api/client";
import { useCreateTask, useDeleteTask, useUpdateTask } from "./mutations";
import { taskKeys } from "./queries";

vi.mock("../hooks", () => ({ useWorkspaceId: () => "workspace-1" }));

function wrapper(queryClient: QueryClient) {
  return function TestWrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

describe("Task mutation cache invalidation", () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    setApiInstance({
      createTask: vi.fn().mockResolvedValue({}),
      updateTask: vi.fn().mockResolvedValue({}),
      deleteTask: vi.fn().mockResolvedValue({}),
    } as unknown as ApiClient);
  });

  afterEach(() => {
    queryClient.clear();
    vi.restoreAllMocks();
  });

  it("invalidates finite and infinite Task lists after create, update, and archive", async () => {
    const invalidate = vi.spyOn(queryClient, "invalidateQueries");
    const create = renderHook(() => useCreateTask(), { wrapper: wrapper(queryClient) });
    const update = renderHook(() => useUpdateTask(), { wrapper: wrapper(queryClient) });
    const archive = renderHook(() => useDeleteTask(), { wrapper: wrapper(queryClient) });

    await act(async () => {
      await create.result.current.mutateAsync({ title: "Task" });
      await update.result.current.mutateAsync({ id: "task-1", title: "Updated", expected_revision: 1 });
      await archive.result.current.mutateAsync({ id: "task-1", expectedRevision: 2 });
    });

    expect(invalidate).toHaveBeenCalledWith({ queryKey: taskKeys.all("workspace-1") });
    const allKey = JSON.stringify(taskKeys.all("workspace-1"));
    expect(invalidate.mock.calls.filter(([request]) => JSON.stringify(request?.queryKey) === allKey)).toHaveLength(3);
  });
});
