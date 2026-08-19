/**
 * @vitest-environment jsdom
 */
import { act, renderHook } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { setApiInstance } from "../api";
import type { ApiClient } from "../api/client";
import {
  projectRequirementKeys,
  useCreateProjectOutlineNode,
  useSaveProjectRequirementDraft,
} from "./queries";

const CONTENT = {
  problemStatement: "Define scope",
  goals: [],
  inScope: [],
  outOfScope: [],
  constraints: [],
  acceptanceCriteria: [],
  dependencies: [],
};

function wrapper(queryClient: QueryClient) {
  return function TestWrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );
  };
}

describe("project requirement mutation idempotency", () => {
  let queryClient: QueryClient;
  let saveProjectRequirementDraft: ReturnType<typeof vi.fn>;
  let createProjectOutlineNode: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: { mutations: { retry: false } },
    });
    saveProjectRequirementDraft = vi.fn().mockResolvedValue({});
    createProjectOutlineNode = vi.fn().mockResolvedValue({});
    setApiInstance({
      saveProjectRequirementDraft,
      createProjectOutlineNode,
    } as unknown as ApiClient);
  });

  afterEach(() => {
    queryClient.clear();
    vi.restoreAllMocks();
  });

  it("retains one initial-baseline key across failure and rotates it after success", async () => {
    saveProjectRequirementDraft
      .mockRejectedValueOnce(new Error("response lost"))
      .mockResolvedValue({});
    const mutation = renderHook(
      () => useSaveProjectRequirementDraft("workspace-1", "project-1"),
      { wrapper: wrapper(queryClient) }
    );
    const input = {
      expectedRevision: 0,
      content: CONTENT,
      changeSummary: "Initial baseline",
      materialChange: false,
    };

    await act(async () => {
      await expect(mutation.result.current.mutateAsync(input)).rejects.toThrow(
        "response lost"
      );
      await mutation.result.current.mutateAsync(input);
      await mutation.result.current.mutateAsync(input);
    });

    const firstKey =
      saveProjectRequirementDraft.mock.calls[0]?.[1]?.idempotencyKey;
    expect(firstKey).toBeTruthy();
    expect(saveProjectRequirementDraft.mock.calls[1]?.[1]?.idempotencyKey).toBe(
      firstKey
    );
    expect(
      saveProjectRequirementDraft.mock.calls[2]?.[1]?.idempotencyKey
    ).not.toBe(firstKey);
  });

  it("retains one root-node key across failure and rotates it after success", async () => {
    createProjectOutlineNode
      .mockRejectedValueOnce(new Error("response lost"))
      .mockResolvedValue({});
    const mutation = renderHook(
      () => useCreateProjectOutlineNode("workspace-1", "project-1"),
      { wrapper: wrapper(queryClient) }
    );
    const input = { expectedRevision: 0, title: "Delivery" };

    await act(async () => {
      await expect(mutation.result.current.mutateAsync(input)).rejects.toThrow(
        "response lost"
      );
      await mutation.result.current.mutateAsync(input);
      await mutation.result.current.mutateAsync(input);
    });

    const firstKey =
      createProjectOutlineNode.mock.calls[0]?.[1]?.idempotencyKey;
    expect(firstKey).toBeTruthy();
    expect(createProjectOutlineNode.mock.calls[1]?.[1]?.idempotencyKey).toBe(
      firstKey
    );
    expect(
      createProjectOutlineNode.mock.calls[2]?.[1]?.idempotencyKey
    ).not.toBe(firstKey);
  });

  it("invalidates baseline and coverage after a Requirement mutation", async () => {
    const invalidate = vi.spyOn(queryClient, "invalidateQueries");
    const mutation = renderHook(
      () => useSaveProjectRequirementDraft("workspace-1", "project-1"),
      { wrapper: wrapper(queryClient) }
    );

    await act(async () => {
      await mutation.result.current.mutateAsync({
        expectedRevision: 8,
        content: CONTENT,
        changeSummary: "Refresh coverage",
        materialChange: false,
      });
    });

    const keys = invalidate.mock.calls.map(([options]) => options?.queryKey);
    expect(keys).toContainEqual(
      projectRequirementKeys.detail("workspace-1", "project-1")
    );
    expect(keys).toContainEqual(
      projectRequirementKeys.coverage("workspace-1", "project-1")
    );
  });
});
