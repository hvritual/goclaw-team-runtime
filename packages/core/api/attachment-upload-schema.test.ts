import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiClient } from "./client";

const VALID_WORKSPACE_ATTACHMENT = {
  id: "01980000-0000-7000-8000-000000000001",
  workspace_id: "01980000-0000-7000-8000-000000000002",
  issue_id: "01980000-0000-7000-8000-000000000003",
  comment_id: null,
  uploader_type: "member",
  uploader_id: "01980000-0000-7000-8000-000000000004",
  filename: "notes.txt",
  url: "/api/attachments/01980000-0000-7000-8000-000000000001/download",
  download_url: "/api/attachments/01980000-0000-7000-8000-000000000001/download",
  markdown_url: "http://localhost:8080/api/attachments/01980000-0000-7000-8000-000000000001/download",
  content_type: "text/plain; charset=utf-8",
  size_bytes: 5,
  created_at: "2026-08-02T00:00:00Z",
};

function uploadResponse(value: unknown): void {
  vi.stubGlobal(
    "fetch",
    vi.fn().mockResolvedValue(
      new Response(JSON.stringify(value), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    ),
  );
}

describe("attachment upload API boundary", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("accepts the native SQLite workspace attachment response", async () => {
    uploadResponse(VALID_WORKSPACE_ATTACHMENT);
    const client = new ApiClient("http://localhost:3000");

    await expect(
      client.uploadFile(new File(["hello"], "notes.txt", { type: "text/plain" }), {
        issueId: VALID_WORKSPACE_ATTACHMENT.issue_id,
      }),
    ).resolves.toEqual(VALID_WORKSPACE_ATTACHMENT);
  });

  it("accepts the personal direct-object compatibility response", async () => {
    const direct = {
      id: "01980000-0000-7000-8000-000000000005",
      url: "/uploads/users/01980000-0000-7000-8000-000000000004/01980000-0000-7000-8000-000000000005.png",
      download_url: "/uploads/users/01980000-0000-7000-8000-000000000004/01980000-0000-7000-8000-000000000005.png",
      markdown_url: "/uploads/users/01980000-0000-7000-8000-000000000004/01980000-0000-7000-8000-000000000005.png",
      filename: "avatar.png",
    };
    uploadResponse(direct);
    const client = new ApiClient("http://localhost:3000");

    await expect(
      client.uploadFile(new File(["avatar"], "avatar.png", { type: "image/png" })),
    ).resolves.toEqual(direct);
  });

  it("rejects malformed success data instead of reporting a false upload success", async () => {
    uploadResponse({ ...VALID_WORKSPACE_ATTACHMENT, download_url: 42 });
    const client = new ApiClient("http://localhost:3000");

    await expect(
      client.uploadFile(new File(["hello"], "notes.txt", { type: "text/plain" })),
    ).rejects.toThrow("Invalid attachment response");
  });
});
