import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiClient } from "./client";
import { AttachmentResponseSchema } from "./schemas";

const VALID_WORKSPACE_ATTACHMENT = {
  id: "01980000-0000-7000-8000-000000000001",
  workspace_id: "01980000-0000-7000-8000-000000000002",
  issue_id: "01980000-0000-7000-8000-000000000003",
  comment_id: null,
  chat_session_id: null,
  chat_message_id: null,
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

  it("rejects malformed full workspace attachment fields", async () => {
    uploadResponse({ ...VALID_WORKSPACE_ATTACHMENT, size_bytes: "5" });
    const client = new ApiClient("http://localhost:3000");

    await expect(
      client.uploadFile(new File(["hello"], "notes.txt", { type: "text/plain" }), {
        issueId: VALID_WORKSPACE_ATTACHMENT.issue_id,
      }),
    ).rejects.toThrow("Invalid attachment response");
  });

  it("requires every canonical field and exact known actor values", async () => {
    const client = new ApiClient("http://localhost:3000");
    for (const malformed of [
      { ...VALID_WORKSPACE_ATTACHMENT, chat_session_id: undefined },
      { ...VALID_WORKSPACE_ATTACHMENT, chat_message_id: undefined },
      { ...VALID_WORKSPACE_ATTACHMENT, markdown_url: undefined },
      { ...VALID_WORKSPACE_ATTACHMENT, uploader_type: "service" },
      { ...VALID_WORKSPACE_ATTACHMENT, filename: "" },
      { ...VALID_WORKSPACE_ATTACHMENT, content_type: "" },
    ]) {
      uploadResponse(malformed);
      await expect(
        client.uploadFile(new File(["hello"], "notes.txt", { type: "text/plain" }), {
          issueId: VALID_WORKSPACE_ATTACHMENT.issue_id,
        }),
      ).rejects.toThrow("Invalid attachment response");
    }
  });

  it("does not let the direct-object compatibility branch accept malformed workspace data", () => {
    expect(
      AttachmentResponseSchema.safeParse({ ...VALID_WORKSPACE_ATTACHMENT, size_bytes: "5" }).success,
    ).toBe(false);
  });

  it("does not reinterpret a canonical response missing workspace_id as a direct object", async () => {
    const { workspace_id: _, ...missingWorkspace } = VALID_WORKSPACE_ATTACHMENT;
    uploadResponse(missingWorkspace);
    const client = new ApiClient("http://localhost:3000");

    await expect(
      client.uploadFile(new File(["hello"], "notes.txt", { type: "text/plain" }), {
        issueId: VALID_WORKSPACE_ATTACHMENT.issue_id,
      }),
    ).rejects.toThrow("Invalid attachment response");
    expect(AttachmentResponseSchema.safeParse(missingWorkspace).success).toBe(false);
  });

  it("rejects malformed list and metadata success bodies", async () => {
    uploadResponse([{ ...VALID_WORKSPACE_ATTACHMENT, content_type: 42 }]);
    const client = new ApiClient("http://localhost:3000");
    await expect(client.listAttachments(VALID_WORKSPACE_ATTACHMENT.issue_id)).rejects.toThrow(
      "Invalid attachment response",
    );

    uploadResponse({ ...VALID_WORKSPACE_ATTACHMENT, content_type: 42 });
    await expect(client.getAttachment(VALID_WORKSPACE_ATTACHMENT.id)).rejects.toThrow(
      "Invalid attachment response",
    );
  });
});
