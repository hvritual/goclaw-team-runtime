import type { Attachment } from "@multica/core/types";
import { contentReferencesAttachment } from "@multica/core/types";

/**
 * Build the full replacement bag expected by the Canonical Issue API.
 * Existing authoritative references survive description autosaves; only new
 * uploads that are actually referenced by the saved markdown are appended.
 */
export function completeIssueAttachmentIDs(
  markdown: string,
  authoritative: readonly Attachment[],
  pending: readonly Attachment[],
): string[] {
  const ids = new Set(authoritative.map((attachment) => attachment.id));
  for (const attachment of pending) {
    if (contentReferencesAttachment(markdown, attachment)) ids.add(attachment.id);
  }
  return [...ids];
}
