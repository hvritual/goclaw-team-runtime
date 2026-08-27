/**
 * @vitest-environment jsdom
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement, type ReactNode } from "react";
import { setApiInstance } from "../api";
import type { ApiClient } from "../api/client";
import { useIssueSimilarity } from "./similarity";

const workspace = vi.hoisted(() => ({ id: "workspace-1" }));

vi.mock("../hooks", () => ({
  useWorkspaceId: () => workspace.id,
}));

const AVAILABLE = {
  ranking_version: "lexical-v1",
  candidates: [],
  truncated: false,
  detector_available: true,
};

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

describe("useIssueSimilarity", () => {
  let queryClient: QueryClient;
  let checkIssueSimilarity: ReturnType<typeof vi.fn>;
  let checkExistingIssueSimilarity: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    workspace.id = "workspace-1";
    queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    checkIssueSimilarity = vi.fn().mockResolvedValue(AVAILABLE);
    checkExistingIssueSimilarity = vi.fn().mockResolvedValue(AVAILABLE);
    setApiInstance({
      checkIssueSimilarity,
      checkExistingIssueSimilarity,
    } as unknown as ApiClient);
  });

  afterEach(() => {
    queryClient.clear();
    vi.restoreAllMocks();
  });

  it("debounces non-empty pre-create input without blocking the caller", async () => {
    const { result } = renderHook(
      () => useIssueSimilarity({
        title: " Alpha beta ",
        description: "Details",
        projectId: "project-1",
        debounceMs: 0,
      }),
      { wrapper: createWrapper(queryClient) },
    );

    await waitFor(() => {
      expect(checkIssueSimilarity).toHaveBeenCalledWith({
        title: "Alpha beta",
        description: "Details",
        project_id: "project-1",
      });
    });
    expect(result.current.data).toEqual(AVAILABLE);
    expect(checkExistingIssueSimilarity).not.toHaveBeenCalled();
  });

  it("does not request an empty title and uses the canonical route for an existing Issue", async () => {
    const empty = renderHook(
      () => useIssueSimilarity({ title: "  ", debounceMs: 0 }),
      { wrapper: createWrapper(queryClient) },
    );
    await new Promise((resolve) => setTimeout(resolve, 0));
    expect(empty.result.current.fetchStatus).toBe("idle");
    expect(checkIssueSimilarity).not.toHaveBeenCalled();

    const existing = renderHook(
      () => useIssueSimilarity({
        issueId: "issue-1",
        title: "Renamed title",
        debounceMs: 0,
      }),
      { wrapper: createWrapper(queryClient) },
    );
    await waitFor(() => {
      expect(checkExistingIssueSimilarity).toHaveBeenCalledWith("issue-1");
    });
    expect(existing.result.current.data).toEqual(AVAILABLE);
    expect(checkIssueSimilarity).not.toHaveBeenCalled();
  });

  it("does not reuse a fresh similarity response after a workspace switch", async () => {
    const firstWorkspace = {
      ...AVAILABLE,
      ranking_version: "workspace-1-result",
    };
    const secondWorkspace = {
      ...AVAILABLE,
      ranking_version: "workspace-2-result",
    };
    checkIssueSimilarity.mockImplementation(() => Promise.resolve(
      workspace.id === "workspace-1" ? firstWorkspace : secondWorkspace,
    ));

    const { result, rerender } = renderHook(
      () => useIssueSimilarity({ title: "Same draft", debounceMs: 0 }),
      { wrapper: createWrapper(queryClient) },
    );

    await waitFor(() => {
      expect(result.current.data).toEqual(firstWorkspace);
    });
    expect(checkIssueSimilarity).toHaveBeenCalledTimes(1);

    workspace.id = "workspace-2";
    rerender();

    await waitFor(() => {
      expect(result.current.data).toEqual(secondWorkspace);
    });
    expect(checkIssueSimilarity).toHaveBeenCalledTimes(2);
  });
});
