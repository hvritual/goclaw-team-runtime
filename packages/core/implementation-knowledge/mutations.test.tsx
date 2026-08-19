/**
 * @vitest-environment jsdom
 */
import { act, renderHook } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { setApiInstance } from "../api";
import type { ApiClient } from "../api/client";
import { issueKeys } from "../issues/queries";
import { projectKeys } from "../projects/queries";
import { taskKeys } from "../tasks/queries";
import {
  useArchiveProjectRetrospective,
  useCreateProjectRetrospective,
  useCreateProjectRetrospectiveTarget,
  useUpdateProjectRetrospective,
} from "./mutations";
import { implementationKnowledgeKeys } from "./queries";

vi.mock("../hooks", () => ({ useWorkspaceId: () => "workspace-1" }));

const CREATE_INPUT = {
  content: {
    summary: "Delivery",
    successes: [],
    problems: [],
    lessons: ["Review sooner"],
    actionItems: [{ title: "Schedule review" }],
  },
  participants: [{ memberId: "member-1", role: "participant" as const }],
};

function wrapper(queryClient: QueryClient) {
  return function TestWrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

describe("Project Retrospective mutations", () => {
  let queryClient: QueryClient;
  let create: ReturnType<typeof vi.fn>;
  let target: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    queryClient = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
    create = vi.fn().mockResolvedValue({ id: "retro-1" });
    target = vi.fn().mockResolvedValue({ actionItemId: "action-1" });
    setApiInstance({
      createProjectRetrospective: create,
      updateProjectRetrospective: vi.fn().mockResolvedValue({ id: "retro-1" }),
      archiveProjectRetrospective: vi.fn().mockResolvedValue({ id: "retro-1" }),
      createProjectRetrospectiveTarget: target,
    } as unknown as ApiClient);
  });

  afterEach(() => {
    queryClient.clear();
    vi.restoreAllMocks();
  });

  it("retains one create key across failure and rotates it after success", async () => {
    create.mockRejectedValueOnce(new Error("response lost")).mockResolvedValue({ id: "retro-1" });
    const mutation = renderHook(
      () => useCreateProjectRetrospective("project-1"),
      { wrapper: wrapper(queryClient) },
    );

    await act(async () => {
      await expect(mutation.result.current.mutateAsync(CREATE_INPUT)).rejects.toThrow("response lost");
      await mutation.result.current.mutateAsync(CREATE_INPUT);
      await mutation.result.current.mutateAsync(CREATE_INPUT);
    });

    const firstKey = create.mock.calls[0]?.[1]?.idempotencyKey;
    expect(firstKey).toBeTruthy();
    expect(create.mock.calls[1]?.[1]?.idempotencyKey).toBe(firstKey);
    expect(create.mock.calls[2]?.[1]?.idempotencyKey).not.toBe(firstKey);
  });

  it("uses different target keys for a changed target command and retains failures", async () => {
    target.mockRejectedValueOnce(new Error("response lost")).mockResolvedValue({ actionItemId: "action-1" });
    const mutation = renderHook(
      () => useCreateProjectRetrospectiveTarget("project-1"),
      { wrapper: wrapper(queryClient) },
    );
    const task = { retrospectiveId: "retro-1", actionItemId: "action-1", targetKind: "task" as const };

    await act(async () => {
      await expect(mutation.result.current.mutateAsync(task)).rejects.toThrow("response lost");
      await mutation.result.current.mutateAsync(task);
      await mutation.result.current.mutateAsync({ ...task, targetKind: "issue" });
    });

    const failedKey = target.mock.calls[0]?.[3]?.idempotencyKey;
    expect(failedKey).toBeTruthy();
    expect(target.mock.calls[1]?.[3]?.idempotencyKey).toBe(failedKey);
    expect(target.mock.calls[2]?.[3]?.idempotencyKey).not.toBe(failedKey);
  });

  it("invalidates only Retrospective dependencies after successful commands", async () => {
    const invalidate = vi.spyOn(queryClient, "invalidateQueries");
    const createMutation = renderHook(
      () => useCreateProjectRetrospective("project-1"),
      { wrapper: wrapper(queryClient) },
    );
    const updateMutation = renderHook(
      () => useUpdateProjectRetrospective("project-1"),
      { wrapper: wrapper(queryClient) },
    );
    const archiveMutation = renderHook(
      () => useArchiveProjectRetrospective("project-1"),
      { wrapper: wrapper(queryClient) },
    );
    const targetMutation = renderHook(
      () => useCreateProjectRetrospectiveTarget("project-1"),
      { wrapper: wrapper(queryClient) },
    );

    await act(async () => {
      await createMutation.result.current.mutateAsync(CREATE_INPUT);
      await updateMutation.result.current.mutateAsync({
        retrospectiveId: "retro-1",
        expectedRevision: 1,
        action: "publish",
      });
      await archiveMutation.result.current.mutateAsync({
        retrospectiveId: "retro-1",
        expectedRevision: 2,
      });
      await targetMutation.result.current.mutateAsync({
        retrospectiveId: "retro-1",
        actionItemId: "action-1",
        targetKind: "issue",
      });
    });

    expect(invalidate).toHaveBeenCalledWith({
      queryKey: implementationKnowledgeKeys.retrospectiveLists("workspace-1", "project-1"),
    });
    expect(invalidate).toHaveBeenCalledWith({
      queryKey: implementationKnowledgeKeys.retrospectiveDetail("workspace-1", "project-1", "retro-1"),
    });
    expect(invalidate).toHaveBeenCalledWith({ queryKey: projectKeys.detail("workspace-1", "project-1") });
    expect(invalidate).toHaveBeenCalledWith({ queryKey: taskKeys.all("workspace-1") });
    expect(invalidate).toHaveBeenCalledWith({ queryKey: issueKeys.all("workspace-1") });
    expect(invalidate.mock.calls.some(([request]) => request?.queryKey?.[0] === "knowledge")).toBe(false);
  });
});
