import { describe, expect, it } from "vitest";
import type { Attachment } from "@multica/core/types";
import { completeIssueAttachmentIDs } from "./issue-attachment-bag";

function attachment(id: string): Attachment {
  return {
    id,
    workspace_id: "workspace-1",
    issue_id: "issue-1",
    comment_id: null,
    chat_session_id: null,
    chat_message_id: null,
    uploader_type: "member",
    uploader_id: "member-1",
    filename: `${id}.txt`,
    url: `/api/attachments/${id}/download`,
    download_url: `/api/attachments/${id}/download`,
    markdown_url: `/api/attachments/${id}/download`,
    content_type: "text/plain; charset=utf-8",
    size_bytes: 1,
    created_at: "2026-08-15T00:00:00Z",
  };
}

describe("completeIssueAttachmentIDs", () => {
  it("retains the authoritative pre-refresh bag and adds only referenced pending uploads", () => {
    const retained = attachment("attachment-a");
    const pending = attachment("attachment-b");
    const unused = attachment("attachment-c");
    const markdown = `retained text\n\n[attachment-b.txt](${pending.markdown_url})`;

    expect(completeIssueAttachmentIDs(markdown, [retained], [pending, unused])).toEqual([
      "attachment-a",
      "attachment-b",
    ]);
  });

  it("deduplicates an attachment returned by both the authoritative query and upload session", () => {
    const retained = attachment("attachment-a");
    expect(completeIssueAttachmentIDs(retained.markdown_url, [retained], [retained])).toEqual([
      "attachment-a",
    ]);
  });
});
