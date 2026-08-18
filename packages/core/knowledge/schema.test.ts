import { describe, expect, it } from "vitest";
import { parseWithFallback } from "../api/schema";
import {
  EMPTY_KNOWLEDGE_CANDIDATE_LIST,
  EMPTY_KNOWLEDGE_LIST,
  commentKnowledgeProposalResponseSchema,
  knowledgeCandidateListSchema,
  knowledgeCandidateSchema,
  knowledgeEntrySchema,
  knowledgeListSchema,
  reviewKnowledgeResponseSchema,
} from "./schema";

const EMPTY_ENTRY = {
  id: "",
  workspaceId: "",
  projectId: null,
  candidateId: null,
  kind: "reference" as const,
  status: "published" as const,
  currentRevision: 0,
  revisions: [],
  createdAt: "",
  updatedAt: "",
};

const EMPTY_CANDIDATE = {
  id: "",
  workspaceId: "",
  projectId: null,
  knowledgeId: null,
  targetRevision: 0,
  kind: "reference" as const,
  title: "",
  content: "",
  reason: "",
  status: "candidate" as const,
  revision: 0,
  proposedBy: "",
  sourceRefs: [],
  createdAt: "",
  updatedAt: "",
};

describe("knowledge response schemas", () => {
  it("transforms a comment decision capture response", () => {
    const result = parseWithFallback(
      {
        queued: true,
        evidence_id: "evidence-1",
        source_revision: "2026-07-31T08:30:00Z",
      },
      commentKnowledgeProposalResponseSchema,
      { queued: false, evidenceId: null, sourceRevision: "" },
      { endpoint: "POST /api/comments/:id/knowledge-proposals" },
    );

    expect(result).toEqual({
      queued: true,
      evidenceId: "evidence-1",
      sourceRevision: "2026-07-31T08:30:00Z",
    });
  });

  it("falls back safely when a comment decision capture response is malformed", () => {
    const fallback = { queued: false, evidenceId: null, sourceRevision: "" };
    const result = parseWithFallback(
      { queued: "yes", source_revision: 42 },
      commentKnowledgeProposalResponseSchema,
      fallback,
      { endpoint: "POST /api/comments/:id/knowledge-proposals" },
    );

    expect(result).toEqual(fallback);
  });

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
            revision: {
              number: 1,
              supersedes_revision: 0,
              title: "Keep evidence",
              content: "Retry ingestion without deleting source evidence.",
              created_by: "user-1",
              created_at: "2026-07-31T00:00:00Z",
              source_refs: [{
                type: "project_requirement_item",
                id: "baseline-1:scope-1",
                revision: "2",
                citation: "Approved requirement scope-1 revision 2",
                asset_id: null,
                asset_version_id: null,
              }],
            },
            citation: "Approved requirement scope-1 revision 2",
            matched_by: "source",
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
      revisions: [
        {
          title: "Keep evidence",
          createdBy: "user-1",
          supersedesRevision: 0,
          sourceRefs: [{ type: "project_requirement_item", citation: "Approved requirement scope-1 revision 2" }],
        },
      ],
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

  it("falls back safely when a knowledge entry response is malformed", () => {
    const result = parseWithFallback(
      { id: 42, current_revision: "one" },
      knowledgeEntrySchema,
      EMPTY_ENTRY,
      { endpoint: "GET /api/knowledge/:id" },
    );

    expect(result).toEqual(EMPTY_ENTRY);
  });

  it("falls back safely when a proposal response is malformed", () => {
    const result = parseWithFallback(
      { id: null, revision: "one" },
      knowledgeCandidateSchema,
      EMPTY_CANDIDATE,
      { endpoint: "POST /api/knowledge/proposals" },
    );

    expect(result).toEqual(EMPTY_CANDIDATE);
  });

  it("falls back safely when a candidate list response is malformed", () => {
    const result = parseWithFallback(
      { candidates: [{ id: 42 }], total: "one" },
      knowledgeCandidateListSchema,
      EMPTY_KNOWLEDGE_CANDIDATE_LIST,
      { endpoint: "GET /api/knowledge/candidates" },
    );

    expect(result).toEqual(EMPTY_KNOWLEDGE_CANDIDATE_LIST);
  });

  it("falls back safely when a review response is malformed", () => {
    const fallback = { candidate: EMPTY_CANDIDATE, entry: null };
    const result = parseWithFallback(
      { candidate: { id: 42 }, entry: "published" },
      reviewKnowledgeResponseSchema,
      fallback,
      { endpoint: "POST /api/knowledge/candidates/:id/review" },
    );

    expect(result).toEqual(fallback);
  });
});
