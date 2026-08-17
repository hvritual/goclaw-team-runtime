/**
 * @vitest-environment jsdom
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, renderHook } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { setApiInstance } from "../api";
import type { ApiClient } from "../api/client";
import { useCreateTask, useDeleteTask, usePromoteTask, useUpdateTask } from "./mutations";
import { taskKeys } from "./queries";
import { issueKeys } from "../issues/queries";

vi.mock("../hooks", () => ({ useWorkspaceId: () => "workspace-1" }));

function wrapper(queryClient: QueryClient) {
  return function TestWrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

describe("Task mutation cache invalidation", () => {
  let queryClient: QueryClient;
  let promoteTask: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    promoteTask = vi.fn().mockResolvedValue({});
    setApiInstance({
      createTask: vi.fn().mockResolvedValue({}),
      updateTask: vi.fn().mockResolvedValue({}),
      deleteTask: vi.fn().mockResolvedValue({}),
      promoteTask,
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

  it("invalidates the promoted Task detail and both Task and Issue collections", async () => {
    const invalidate = vi.spyOn(queryClient, "invalidateQueries");
    const promote = renderHook(() => usePromoteTask(), { wrapper: wrapper(queryClient) });

    await act(async () => {
      await promote.result.current.mutateAsync({ id: "task-1", expected_revision: 1, complete_task: true });
    });

    expect(invalidate).toHaveBeenCalledWith({ queryKey: taskKeys.detail("workspace-1", "task-1") });
    expect(invalidate).toHaveBeenCalledWith({ queryKey: taskKeys.all("workspace-1") });
    expect(invalidate).toHaveBeenCalledWith({ queryKey: issueKeys.all("workspace-1") });
  });

  it("retains one promotion key across failure and rotates it after success", async () => {
    promoteTask.mockRejectedValueOnce(new Error("response lost")).mockResolvedValue({});
    const promote = renderHook(() => usePromoteTask(), { wrapper: wrapper(queryClient) });
    const command = { id: "task-1", expected_revision: 1, complete_task: true };

    await act(async () => {
      await expect(promote.result.current.mutateAsync(command)).rejects.toThrow("response lost");
      await promote.result.current.mutateAsync(command);
      await promote.result.current.mutateAsync(command);
    });

    const firstKey = promoteTask.mock.calls[0]?.[1]?.idempotency_key;
    expect(firstKey).toBeTruthy();
    expect(promoteTask.mock.calls[1]?.[1]?.idempotency_key).toBe(firstKey);
    expect(promoteTask.mock.calls[2]?.[1]?.idempotency_key).not.toBe(firstKey);
  });
});
