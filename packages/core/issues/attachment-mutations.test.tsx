/**
 * @vitest-environment jsdom
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, renderHook } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { setApiInstance } from "../api";
import type { ApiClient } from "../api/client";
import { issueKeys } from "./queries";
import { useDeleteIssueAttachment } from "./mutations";

vi.mock("../hooks", () => ({ useWorkspaceId: () => "workspace-1" }));

function wrapper(queryClient: QueryClient) {
  return function TestWrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

describe("useDeleteIssueAttachment", () => {
  let queryClient: QueryClient;
  let deleteAttachment: ReturnType<typeof vi.fn<(id: string) => Promise<void>>>;

  beforeEach(() => {
    queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    deleteAttachment = vi.fn().mockResolvedValue(undefined);
    setApiInstance({ deleteAttachment } as unknown as ApiClient);
  });

  afterEach(() => {
    queryClient.clear();
    vi.restoreAllMocks();
  });

  it("deletes through the strict client and invalidates attachments plus Issue detail", async () => {
    const invalidate = vi.spyOn(queryClient, "invalidateQueries");
    const { result } = renderHook(() => useDeleteIssueAttachment("issue-1"), {
      wrapper: wrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync("attachment-a");
    });

    expect(deleteAttachment).toHaveBeenCalledWith("attachment-a");
    expect(invalidate).toHaveBeenCalledWith({ queryKey: issueKeys.attachments("issue-1") });
    expect(invalidate).toHaveBeenCalledWith({ queryKey: issueKeys.detail("workspace-1", "issue-1") });
  });
});
