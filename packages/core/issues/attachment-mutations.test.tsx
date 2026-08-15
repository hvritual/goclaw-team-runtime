/**
 * @vitest-environment jsdom
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, renderHook } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { setApiInstance } from "../api";
import type { ApiClient } from "../api/client";
import type { Attachment } from "../types";
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

  it("deletes through the strict client, removes the cached row, and invalidates attachments plus Issue detail", async () => {
    const retained = [
      {
        id: "attachment-a",
        workspace_id: "workspace-1",
        issue_id: "issue-1",
        comment_id: null,
        chat_session_id: null,
        chat_message_id: null,
        uploader_type: "member",
        uploader_id: "user-1",
        filename: "attachment-a.txt",
        url: "/api/attachments/attachment-a/download",
        download_url: "/api/attachments/attachment-a/download",
        markdown_url: "/api/attachments/attachment-a/download",
        content_type: "text/plain",
        size_bytes: 1,
        created_at: "2026-08-15T00:00:00Z",
      },
      {
        id: "attachment-b",
        workspace_id: "workspace-1",
        issue_id: "issue-1",
        comment_id: null,
        chat_session_id: null,
        chat_message_id: null,
        uploader_type: "member",
        uploader_id: "user-1",
        filename: "attachment-b.txt",
        url: "/api/attachments/attachment-b/download",
        download_url: "/api/attachments/attachment-b/download",
        markdown_url: "/api/attachments/attachment-b/download",
        content_type: "text/plain",
        size_bytes: 1,
        created_at: "2026-08-15T00:00:01Z",
      },
    ] satisfies Attachment[];
    queryClient.setQueryData(issueKeys.attachments("issue-1"), retained);
    const invalidate = vi.spyOn(queryClient, "invalidateQueries");
    const { result } = renderHook(() => useDeleteIssueAttachment("issue-1"), {
      wrapper: wrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync("attachment-a");
    });

    expect(deleteAttachment).toHaveBeenCalledWith("attachment-a");
    expect(queryClient.getQueryData(issueKeys.attachments("issue-1"))).toEqual([retained[1]]);
    expect(invalidate).toHaveBeenCalledWith({ queryKey: issueKeys.attachments("issue-1") });
    expect(invalidate).toHaveBeenCalledWith({ queryKey: issueKeys.detail("workspace-1", "issue-1") });
  });

  it("keeps the cached row when deletion fails", async () => {
    const retained = [{ id: "attachment-a" }] as Attachment[];
    queryClient.setQueryData(issueKeys.attachments("issue-1"), retained);
    deleteAttachment.mockRejectedValueOnce(new Error("delete failed"));
    const { result } = renderHook(() => useDeleteIssueAttachment("issue-1"), {
      wrapper: wrapper(queryClient),
    });

    await act(async () => {
      await expect(result.current.mutateAsync("attachment-a")).rejects.toThrow("delete failed");
    });

    expect(queryClient.getQueryData(issueKeys.attachments("issue-1"))).toEqual(retained);
  });
});
