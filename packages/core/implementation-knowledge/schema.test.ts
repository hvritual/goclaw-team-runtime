import { describe, expect, it } from "vitest";
import { parseWithFallback } from "../api/schema";
import {
  EMPTY_ACCEPTANCE_CONCLUSION_LIST,
  EMPTY_RETROSPECTIVE_LIST,
  acceptanceConclusionListSchema,
  acceptanceConclusionSchema,
  projectRetrospectiveListSchema,
  projectRetrospectiveSchema,
} from "./schema";

describe("implementation knowledge response schemas", () => {
  it("transforms acceptance conclusions at the API boundary", () => {
    const result = parseWithFallback(
      {
        acceptance_conclusions: [{
          id: "acceptance-1",
          workspace_id: "workspace-1",
          issue_id: "issue-1",
          result: "accepted",
          rationale: "Acceptance checks passed.",
          evidence_refs: ["artifact://report/v1"],
          actor_id: "member-1",
          created_at: "2026-08-01T00:00:00Z",
          updated_at: "2026-08-01T00:00:00Z",
        }],
        total: 1,
      },
      acceptanceConclusionListSchema,
      EMPTY_ACCEPTANCE_CONCLUSION_LIST,
      { endpoint: "GET /api/issues/:id/acceptance-conclusions" },
    );

    expect(result.acceptanceConclusions[0]).toMatchObject({
      workspaceId: "workspace-1",
      issueId: "issue-1",
      evidenceRefs: ["artifact://report/v1"],
      actorId: "member-1",
    });
  });

  it("falls back when acceptance conclusions are malformed", () => {
    const result = parseWithFallback(
      { acceptance_conclusions: [{ id: 42 }], total: "one" },
      acceptanceConclusionListSchema,
      EMPTY_ACCEPTANCE_CONCLUSION_LIST,
      { endpoint: "GET /api/issues/:id/acceptance-conclusions" },
    );
    expect(result).toEqual(EMPTY_ACCEPTANCE_CONCLUSION_LIST);
  });

  it("rejects malformed singular acceptance and retrospective mutation responses", () => {
    expect(acceptanceConclusionSchema.safeParse({ id: 42, result: "accepted" }).success).toBe(false);
    expect(projectRetrospectiveSchema.safeParse({ id: "retro-1", lessons: "none" }).success).toBe(false);
  });

  it("transforms retrospectives and falls back on malformed responses", () => {
    const valid = parseWithFallback(
      {
        retrospectives: [{
          id: "retro-1",
          workspace_id: "workspace-1",
          project_id: "project-1",
          summary: "Delivery completed.",
          successes: ["Small batches"],
          problems: [],
          lessons: ["Review sooner"],
          follow_up_refs: ["issue://next"],
          actor_id: "member-1",
          created_at: "2026-08-01T00:00:00Z",
          updated_at: "2026-08-01T00:00:00Z",
        }],
        total: 1,
      },
      projectRetrospectiveListSchema,
      EMPTY_RETROSPECTIVE_LIST,
      { endpoint: "GET /api/projects/:id/retrospectives" },
    );
    expect(valid.retrospectives[0]).toMatchObject({
      projectId: "project-1",
      followUpRefs: ["issue://next"],
    });

    const malformed = parseWithFallback(
      { retrospectives: [{ id: null, lessons: "none" }], total: 1 },
      projectRetrospectiveListSchema,
      EMPTY_RETROSPECTIVE_LIST,
      { endpoint: "GET /api/projects/:id/retrospectives" },
    );
    expect(malformed).toEqual(EMPTY_RETROSPECTIVE_LIST);
  });
});
