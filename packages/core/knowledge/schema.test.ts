import { describe, expect, it } from "vitest";
import { parseWithFallback } from "../api/schema";
import {
  EMPTY_KNOWLEDGE_LIST,
  knowledgeListSchema,
} from "./schema";

describe("knowledge response schemas", () => {
  it("transforms the wire response into the shared knowledge model", () => {
    const result = parseWithFallback(
      {
        entries: [
          {
            id: "knowledge-1",
            workspace_id: "workspace-1",
            project_id: null,
            candidate_id: "candidate-1",
            kind: "lesson",
            status: "published",
            current_revision: 1,
            revisions: [
              {
                number: 1,
                title: "Keep evidence",
                content: "Retry ingestion without deleting source evidence.",
                created_by: "user-1",
                created_at: "2026-07-31T00:00:00Z",
                source_refs: [],
              },
            ],
            created_at: "2026-07-31T00:00:00Z",
            updated_at: "2026-07-31T00:00:00Z",
          },
        ],
        total: 1,
        next_cursor: null,
      },
      knowledgeListSchema,
      EMPTY_KNOWLEDGE_LIST,
      { endpoint: "GET /api/knowledge" },
    );

    expect(result.entries[0]).toMatchObject({
      id: "knowledge-1",
      workspaceId: "workspace-1",
      currentRevision: 1,
      revisions: [{ title: "Keep evidence", createdBy: "user-1" }],
    });
  });

  it("falls back safely when a knowledge response is malformed", () => {
    const result = parseWithFallback(
      {
        entries: [{ id: 42, status: null }],
        total: "one",
      },
      knowledgeListSchema,
      EMPTY_KNOWLEDGE_LIST,
      { endpoint: "GET /api/knowledge" },
    );

    expect(result).toEqual(EMPTY_KNOWLEDGE_LIST);
  });
});
